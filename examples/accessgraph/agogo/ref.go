// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"sort"
)

// Line-for-line port of oracle.py: top-down enumeration of derivations,
// memoised per (user, delegation budget), string-keyed maps throughout. It is
// the readable reference; fast.go is the optimized version and is checked
// against it.

type refSupport struct {
	role, scope string
	h           int
	c           string
}

type refKey struct {
	u      string
	budget int
}

type refIndex struct {
	parent  map[string]*string
	perms   map[string][]string
	inherit map[string][]string
	up      map[string][]string
	allowBy map[string][]Grant
	denyBy  map[string][]Grant
	delegTo map[string][]Deleg
}

// IDWidth is the number of digits after the letter in a grant or delegation id
// (G000001, D000001). A database with another width is outside the domain.
const IDWidth = 6

const maxIDs = 999999

func checkIDWidth(db *DB) error {
	ok := func(id string, letter byte) bool {
		if len(id) != IDWidth+1 || id[0] != letter {
			return false
		}
		for i := 1; i < len(id); i++ {
			if id[i] < '0' || id[i] > '9' {
				return false
			}
		}
		return true
	}
	for _, g := range db.Grants {
		if !ok(g.ID, 'G') {
			return fmt.Errorf("id width: grant id %q is not G plus %d digits", g.ID, IDWidth)
		}
	}
	for _, d := range db.Delegations {
		if !ok(d.ID, 'D') {
			return fmt.Errorf("id width: delegation id %q is not D plus %d digits", d.ID, IDWidth)
		}
	}
	return nil
}

func checkRefs(db *DB) error {
	set := func(xs []string) map[string]bool {
		m := make(map[string]bool, len(xs))
		for _, x := range xs {
			m[x] = true
		}
		return m
	}
	users, groups := set(db.Users), set(db.Groups)
	roles, res := map[string]bool{}, map[string]bool{}
	for _, r := range db.Roles {
		roles[r.ID] = true
	}
	for _, r := range db.Resources {
		res[r.ID] = true
	}
	subj := func(s string) bool { return users[s] || groups[s] }
	for _, r := range db.Roles {
		for _, p := range r.Inherits {
			if !roles[p] {
				return fmt.Errorf("dangling reference: role %s inherits %s", r.ID, p)
			}
		}
	}
	for _, r := range db.Resources {
		if r.Parent != nil && !res[*r.Parent] {
			return fmt.Errorf("dangling reference: resource %s parent", r.ID)
		}
	}
	for _, m := range db.Members {
		if !subj(m.Member) || !groups[m.Group] {
			return fmt.Errorf("dangling reference: member %v", m)
		}
	}
	for _, g := range db.Grants {
		if !subj(g.Subject) || !roles[g.Role] || !res[g.Scope] || (g.Effect != "allow" && g.Effect != "deny") {
			return fmt.Errorf("dangling reference: grant %s", g.ID)
		}
	}
	for _, d := range db.Delegations {
		if !users[d.From] || !users[d.To] || !roles[d.Role] || !res[d.Scope] {
			return fmt.Errorf("dangling reference: delegation %s", d.ID)
		}
	}
	for _, q := range db.Requests {
		if !users[q.User] || !res[q.Resource] {
			return fmt.Errorf("dangling reference: request %s", q.ID)
		}
	}
	return nil
}

func buildRefIndex(db *DB) *refIndex {
	ix := &refIndex{
		parent: map[string]*string{}, perms: map[string][]string{}, inherit: map[string][]string{},
		up: map[string][]string{}, allowBy: map[string][]Grant{}, denyBy: map[string][]Grant{},
		delegTo: map[string][]Deleg{},
	}
	for _, r := range db.Resources {
		ix.parent[r.ID] = r.Parent
	}
	for _, r := range db.Roles {
		ix.perms[r.ID] = r.Perms
		ix.inherit[r.ID] = r.Inherits
	}
	for _, m := range db.Members {
		ix.up[m.Member] = append(ix.up[m.Member], m.Group)
	}
	for _, g := range db.Grants {
		if g.Effect == "allow" {
			ix.allowBy[g.Subject] = append(ix.allowBy[g.Subject], g)
		} else {
			ix.denyBy[g.Subject] = append(ix.denyBy[g.Subject], g)
		}
	}
	for _, d := range db.Delegations {
		ix.delegTo[d.To] = append(ix.delegTo[d.To], d)
	}
	return ix
}

// ancOrSelf walks parent links from res, with the same cycle cap as the original.
func ancOrSelf(parent map[string]*string, res string) []string {
	var out []string
	cur := &res
	for cur != nil && len(out) <= len(parent) {
		out = append(out, *cur)
		cur = parent[*cur]
	}
	return out
}

func reach(start string, edges map[string][]string) map[string]bool {
	seen := map[string]bool{start: true}
	stack := []string{start}
	for len(stack) > 0 {
		x := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, y := range edges[x] {
			if !seen[y] {
				seen[y] = true
				stack = append(stack, y)
			}
		}
	}
	return seen
}

func refLess(h1 int, c1 string, h2 int, c2 string) bool {
	if h1 != h2 {
		return h1 < h2
	}
	return c1 < c2
}

// EvalRef evaluates every request of db. It is defined only on referentially
// clean databases.
func EvalRef(db *DB, maxHops int) (map[string]Result, error) {
	if err := checkIDWidth(db); err != nil {
		return nil, err
	}
	if err := checkRefs(db); err != nil {
		return nil, err
	}
	ix := buildRefIndex(db)
	subjCache := map[string]map[string]bool{}
	permCache := map[string]map[string]bool{}
	memo := map[refKey][]refSupport{}

	subjects := func(u string) map[string]bool {
		s, ok := subjCache[u]
		if !ok {
			s = reach(u, ix.up)
			subjCache[u] = s
		}
		return s
	}
	perms := func(role string) map[string]bool {
		p, ok := permCache[role]
		if !ok {
			p = map[string]bool{}
			for r := range reach(role, ix.inherit) {
				for _, a := range ix.perms[r] {
					p[a] = true
				}
			}
			permCache[role] = p
		}
		return p
	}

	var chains func(u string, budget int) []refSupport
	chains = func(u string, budget int) []refSupport {
		k := refKey{u, budget}
		if v, ok := memo[k]; ok {
			return v
		}
		var out []refSupport
		for s := range subjects(u) {
			for _, g := range ix.allowBy[s] {
				out = append(out, refSupport{g.Role, g.Scope, 0, g.ID})
			}
		}
		if budget > 0 {
			for _, d := range ix.delegTo[u] {
				covering := map[string]bool{}
				for _, a := range ancOrSelf(ix.parent, d.Scope) {
					covering[a] = true
				}
				for _, s := range chains(d.From, budget-1) {
					if s.role == d.Role && covering[s.scope] {
						out = append(out, refSupport{s.role, d.Scope, s.h + 1, s.c + ">" + d.ID})
					}
				}
			}
		}
		memo[k] = out
		return out
	}

	type rs struct{ role, scope string }
	type hc struct {
		h int
		c string
	}
	heldCache := map[string]map[rs]hc{}
	held := func(u string) map[rs]hc {
		if h, ok := heldCache[u]; ok {
			return h
		}
		best := map[rs]hc{}
		for _, s := range chains(u, maxHops) {
			k := rs{s.role, s.scope}
			if cur, ok := best[k]; !ok || refLess(s.h, s.c, cur.h, cur.c) {
				best[k] = hc{s.h, s.c}
			}
		}
		heldCache[u] = best
		return best
	}

	result := make(map[string]Result, len(db.Requests))
	for _, q := range db.Requests {
		anc := map[string]bool{}
		for _, a := range ancOrSelf(ix.parent, q.Resource) {
			anc[a] = true
		}
		var best *hc
		for k, v := range held(q.User) {
			if anc[k.scope] && perms(k.role)[q.Action] {
				v := v
				if best == nil || refLess(v.h, v.c, best.h, best.c) {
					best = &v
				}
			}
		}
		denied := []string{}
		for s := range subjects(q.User) {
			for _, g := range ix.denyBy[s] {
				if anc[g.Scope] && perms(g.Role)[q.Action] {
					denied = append(denied, g.ID)
				}
			}
		}
		sort.Strings(denied)
		allow := best != nil && len(denied) == 0
		r := Result{Allow: allow, DeniedBy: denied}
		if allow {
			via := best.c
			r.Via = &via
		}
		result[q.ID] = r
	}
	return result, nil
}
