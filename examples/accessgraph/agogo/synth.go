// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
)

// Go port of synth.py. The structure is the same: nested and cyclic groups, deny
// grants below allow grants, invalid delegations, and chains longer than
// MaxHops. The random stream is not Python's, so the same seed gives a
// different database; every database is checked against the Python oracle
// instead of being compared by seed.

const (
	MaxHops     = 3 // a delegation chain may be at most this long
	maxResDepth = 6 // deepest resource (root is depth 0)
)

var actions = []string{"read", "write", "delete", "approve", "admin"}

type Preset struct {
	Users, Groups, Roles, Resources, Grants, Denies, Delegations, Chains, Requests int
}

// Presets: the first three are the Python presets. large and xlarge are the
// 1,000 and 5,000 user sizes used for the scale timings. Ids are fixed width
// (four digits for grants and delegations), which caps grants and
// delegations below 10,000.
var Presets = map[string]Preset{
	"tiny":   {12, 5, 6, 14, 16, 4, 8, 2, 40},
	"small":  {60, 14, 10, 60, 120, 18, 40, 6, 200},
	"medium": {200, 36, 16, 200, 450, 60, 140, 16, 400},
	"large":  {1000, 180, 16, 1000, 2250, 300, 700, 80, 2000},
	"xlarge": {5000, 900, 16, 5000, 8000, 900, 2800, 320, 10000},
}

var PresetOrder = []string{"tiny", "small", "medium", "large", "xlarge"}

type rng struct{ r *rand.Rand }

func newRng(seed uint64) *rng {
	return &rng{rand.New(rand.NewPCG(seed, 0x9E3779B97F4A7C15^seed))}
}

func (g *rng) float() float64 { return g.r.Float64() }

func (g *rng) between(a, b int) int { return a + g.r.IntN(b-a+1) } // inclusive

func pick[T any](g *rng, xs []T) T { return xs[g.r.IntN(len(xs))] }

// sample returns k distinct elements of xs. k is small in every use, so
// rejection on indices is cheaper than shuffling a copy.
func sample[T any](g *rng, xs []T, k int) []T {
	if k > len(xs) {
		k = len(xs)
	}
	idx := make([]int, 0, k)
	for len(idx) < k {
		i := g.r.IntN(len(xs))
		if !slices.Contains(idx, i) {
			idx = append(idx, i)
		}
	}
	out := make([]T, k)
	for j, i := range idx {
		out[j] = xs[i]
	}
	return out
}

func shuffle[T any](g *rng, xs []T) {
	g.r.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
}

func pickInt(g *rng, xs ...int) int { return xs[g.r.IntN(len(xs))] }

// Generate builds a database from a seed and a preset name.
func Generate(seed uint64, size string, cycles bool) (*DB, error) {
	p, ok := Presets[size]
	if !ok {
		return nil, fmt.Errorf("unknown size %q", size)
	}
	if p.Grants+p.Denies > maxIDs || p.Users < 8 || p.Groups < 2 || p.Roles < 1 {
		return nil, fmt.Errorf("preset %q is outside the id width or minimum sizes", size)
	}
	g := newRng(seed)

	users := make([]string, p.Users)
	userSet := make(map[string]bool, p.Users)
	for i := range users {
		users[i] = fmt.Sprintf("u%04d", i)
		userSet[users[i]] = true
	}
	groups := make([]string, p.Groups)
	groupSet := make(map[string]bool, p.Groups)
	for i := range groups {
		groups[i] = fmt.Sprintf("g%03d", i)
		groupSet[groups[i]] = true
	}
	roleIDs := make([]string, p.Roles)
	roleIdx := make(map[string]int, p.Roles)
	for i := range roleIDs {
		roleIDs[i] = fmt.Sprintf("r%02d", i)
		roleIdx[roleIDs[i]] = i
	}

	// roles: acyclic inheritance (a role inherits only from lower-numbered ones)
	roles := make([]Role, p.Roles)
	for i, rid := range roleIDs {
		perms := sample(g, actions, pickInt(g, 1, 1, 2, 2, 3))
		sort.Strings(perms)
		k := min(i, pickInt(g, 0, 0, 1, 1, 2))
		inherits := []string{}
		if k > 0 {
			inherits = sample(g, roleIDs[:i], k)
			sort.Strings(inherits)
		}
		roles[i] = Role{ID: rid, Perms: perms, Inherits: inherits}
	}

	// resources: a tree, bounded depth, biased towards deep chains
	resources := make([]Resource, p.Resources)
	resources[0] = Resource{ID: "res0000"}
	depth := make([]int, p.Resources)
	pool := []int{0} // indices of resources with depth < maxResDepth, in order
	children := make(map[string][]string)
	for i := 1; i < p.Resources; i++ {
		recent := pool
		if len(recent) > 10 {
			recent = recent[len(recent)-10:]
		}
		src := pool
		if g.float() < 0.5 {
			src = recent
		}
		par := pick(g, src)
		parent := resources[par].ID
		resources[i] = Resource{ID: fmt.Sprintf("res%04d", i), Parent: &parent}
		depth[i] = depth[par] + 1
		if depth[i] < maxResDepth {
			pool = append(pool, i)
		}
		children[parent] = append(children[parent], resources[i].ID)
	}
	descCache := make(map[string][]string)
	descOrSelf := func(res string) []string {
		if d, ok := descCache[res]; ok {
			return d
		}
		var out []string
		stack := []string{res}
		for len(stack) > 0 {
			x := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			out = append(out, x)
			stack = append(stack, children[x]...)
		}
		sort.Strings(out)
		descCache[res] = out
		return out
	}

	// membership: groups nest into lower-numbered groups; users join 0-3 groups
	var members []Member
	for j := 1; j < len(groups); j++ {
		parents := sample(g, groups[:j], min(j, pickInt(g, 0, 1, 1, 2)))
		sort.Strings(parents)
		for _, par := range parents {
			members = append(members, Member{groups[j], par})
		}
	}
	for _, u := range users {
		if g.float() < 0.1 {
			continue
		}
		gs := sample(g, groups, pickInt(g, 1, 1, 2, 3))
		sort.Strings(gs)
		for _, grp := range gs {
			members = append(members, Member{u, grp})
		}
	}
	if cycles {
		var gg []Member
		for _, m := range members {
			if groupSet[m.Member] {
				gg = append(gg, m)
			}
		}
		if len(gg) > 0 {
			m := pick(g, gg)
			members = append(members, Member{m.Group, m.Member}) // 2-cycle
		}
		x := pick(g, groups)
		members = append(members, Member{x, x}) // self loop
	}
	directUsers := make(map[string][]string)
	for _, m := range members {
		if userSet[m.Member] {
			directUsers[m.Group] = append(directUsers[m.Group], m.Member)
		}
	}

	// grants (allow and deny share one id space)
	type rawGrant struct{ subject, role, scope, effect string }
	var raw []rawGrant
	for _, e := range []struct {
		effect string
		n      int
	}{{"allow", p.Grants}, {"deny", p.Denies}} {
		for i := 0; i < e.n; i++ {
			var subject string
			if g.float() < 0.5 {
				subject = pick(g, users)
			} else {
				subject = pick(g, groups)
			}
			raw = append(raw, rawGrant{subject, pick(g, roleIDs), pick(g, resources).ID, e.effect})
		}
	}
	shuffle(g, raw)
	grants := make([]Grant, len(raw))
	var allows []Grant
	for i, r := range raw {
		grants[i] = Grant{fmt.Sprintf("G%0*d", IDWidth, i+1), r.subject, r.role, r.scope, r.effect}
		if r.effect == "allow" {
			allows = append(allows, grants[i])
		}
	}
	holderOf := func(gr Grant) string {
		if userSet[gr.Subject] {
			return gr.Subject
		}
		if pool := directUsers[gr.Subject]; len(pool) > 0 {
			return pick(g, pool)
		}
		return ""
	}
	otherUser := func(excl func(string) bool) string {
		for {
			u := pick(g, users)
			if !excl(u) {
				return u
			}
		}
	}

	// delegations: explicit chains (some longer than MaxHops), then singles
	type rawDeleg struct{ from, to, role, scope string }
	var rawD []rawDeleg
	for i := 0; i < p.Chains; i++ {
		gr := pick(g, allows)
		cur := holderOf(gr)
		if cur == "" {
			continue
		}
		scope := gr.Scope
		seen := map[string]bool{cur: true}
		for n := g.between(2, MaxHops+2); n > 0; n-- {
			to := otherUser(func(u string) bool { return seen[u] })
			seen[to] = true
			scope = pick(g, descOrSelf(scope))
			rawD = append(rawD, rawDeleg{cur, to, gr.Role, scope})
			cur = to
		}
	}
	for i := 0; i < p.Delegations; i++ {
		var frm, role, scope string
		if g.float() < 0.6 {
			gr := pick(g, allows)
			frm = holderOf(gr)
			if frm == "" {
				frm = pick(g, users)
			}
			role, scope = gr.Role, pick(g, descOrSelf(gr.Scope))
		} else {
			frm, role, scope = pick(g, users), pick(g, roleIDs), pick(g, resources).ID
		}
		to := otherUser(func(u string) bool { return u == frm })
		rawD = append(rawD, rawDeleg{frm, to, role, scope})
	}
	if len(rawD) > maxIDs {
		return nil, fmt.Errorf("%d delegations exceed the id width", len(rawD))
	}
	shuffle(g, rawD)
	delegations := make([]Deleg, len(rawD))
	for i, d := range rawD {
		delegations[i] = Deleg{fmt.Sprintf("D%0*d", IDWidth, i+1), d.from, d.to, d.role, d.scope}
	}

	// requests: some anchored on grants / delegations, the rest uniform
	requests := make([]Request, p.Requests)
	for i := range requests {
		var user, action, res string
		roll := g.float()
		switch {
		case roll < 0.3 && len(grants) > 0:
			gr := pick(g, grants)
			user = holderOf(gr)
			if user == "" {
				user = pick(g, users)
			}
			action = pick(g, roles[roleIdx[gr.Role]].Perms)
			res = pick(g, descOrSelf(gr.Scope))
		case roll < 0.5 && len(delegations) > 0:
			d := pick(g, delegations)
			user = d.To
			action = pick(g, roles[roleIdx[d.Role]].Perms)
			res = pick(g, descOrSelf(d.Scope))
		default:
			user, action = pick(g, users), pick(g, actions)
			res = pick(g, resources).ID
		}
		requests[i] = Request{fmt.Sprintf("q%04d", i+1), user, action, res}
	}

	return &DB{
		Meta:  Meta{Seed: seed, Size: size, Cycles: cycles, MaxHops: MaxHops},
		Users: users, Groups: groups, Roles: roles, Resources: resources,
		Members: members, Grants: grants, Delegations: delegations, Requests: requests,
	}, nil
}
