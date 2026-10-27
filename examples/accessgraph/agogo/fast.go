// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"slices"
)

// Optimized version of the reference in ref.go. Same algorithm (top-down,
// memoised per user and delegation budget) and the same results, with the
// representation changed:
//
//   - every id is interned to an int32 once, so no string is hashed or
//     compared in the search;
//   - a role's permissions are a 64-bit mask (its closure over inheritance);
//   - "scope is an ancestor of resource" is two integer comparisons, using
//     an Euler tour of the resource tree;
//   - adjacency lists are flat arrays (CSR), built in one counting pass;
//   - a chain is a pointer into a path arena, and its string form is built
//     only to break a tie or to report the winner, byte-for-byte as before.
//
// The resource parents must form a forest. The reference tolerates a parent
// cycle through a length cap; this version rejects it, because a cycle has no
// defined ancestor set.

type csr struct{ off, val []int32 }

func buildCSR(n int, keys, vals []int32) csr {
	off := make([]int32, n+1)
	for _, k := range keys {
		off[k+1]++
	}
	for i := 0; i < n; i++ {
		off[i+1] += off[i]
	}
	val := make([]int32, len(keys))
	next := slices.Clone(off[:n])
	for i, k := range keys {
		val[next[k]] = vals[i]
		next[k]++
	}
	return csr{off, val}
}

func (c csr) row(i int32) []int32 { return c.val[c.off[i]:c.off[i+1]] }

// sup is one derivation of "holds role on scope".
type sup struct{ role, scope, h, grant, tail int32 }

type pnode struct{ prev, deleg int32 }

type fast struct {
	maxHops int
	nUsers  int

	up       csr // subject -> groups it is directly a member of
	permMask []uint64
	tin      []int32
	tout     []int32

	gID             []string
	gRole, gScope   []int32
	allowBy, denyBy csr // subject -> grant index

	dID                  []string
	dFrom, dRole, dScope []int32
	delegTo              csr // user -> delegation index

	qUser, qRes []int32
	qBit        []uint64

	subjCache [][]int32
	subjDone  []bool
	memo      [][]sup
	memoDone  []bool
	heldCache [][]sup
	heldDone  []bool
	path      []pnode
	stamp     []uint32
	epoch     uint32
	heldIdx   map[uint64]int32 // fallback when roles x resources is large
	heldStamp []uint32         // stamped table, one slot per (role, scope)
	heldPos   []int32
	heldEpoch uint32
	nRes      int
	tmp       [][]sup // per-budget scratch for chains
	subjTmp   []int32
	heldTmp   []sup
	bufA      []byte
	bufB      []byte
}

func newFast(db *DB, maxHops int) (*fast, error) {
	if err := checkIDWidth(db); err != nil {
		return nil, err
	}
	f := &fast{maxHops: maxHops, nUsers: len(db.Users)}
	nSubj := len(db.Users) + len(db.Groups)
	subj := make(map[string]int32, nSubj)
	for i, u := range db.Users {
		if _, dup := subj[u]; dup {
			return nil, fmt.Errorf("duplicate subject %s", u)
		}
		subj[u] = int32(i)
	}
	for i, g := range db.Groups {
		if _, dup := subj[g]; dup {
			return nil, fmt.Errorf("subject %s is both a user and a group, or repeated", g)
		}
		subj[g] = int32(len(db.Users) + i)
	}
	roleIdx := make(map[string]int32, len(db.Roles))
	for i, r := range db.Roles {
		if _, dup := roleIdx[r.ID]; dup {
			return nil, fmt.Errorf("duplicate role %s", r.ID)
		}
		roleIdx[r.ID] = int32(i)
	}
	resIdx := make(map[string]int32, len(db.Resources))
	for i, r := range db.Resources {
		if _, dup := resIdx[r.ID]; dup {
			return nil, fmt.Errorf("duplicate resource %s", r.ID)
		}
		resIdx[r.ID] = int32(i)
	}

	// roles: own permission masks, then the closure over inheritance
	actBit := map[string]uint64{}
	own := make([]uint64, len(db.Roles))
	for i, r := range db.Roles {
		for _, a := range r.Perms {
			b, ok := actBit[a]
			if !ok {
				if len(actBit) == 64 {
					return nil, fmt.Errorf("more than 64 distinct actions")
				}
				b = 1 << uint(len(actBit))
				actBit[a] = b
			}
			own[i] |= b
		}
	}
	inh := make([][]int32, len(db.Roles))
	for i, r := range db.Roles {
		for _, p := range r.Inherits {
			j, ok := roleIdx[p]
			if !ok {
				return nil, fmt.Errorf("dangling reference: role %s inherits %s", r.ID, p)
			}
			inh[i] = append(inh[i], j)
		}
	}
	f.permMask = make([]uint64, len(db.Roles))
	seen := make([]int32, len(db.Roles))
	for i := range db.Roles {
		stack := []int32{int32(i)}
		seen[i] = int32(i) + 1
		for len(stack) > 0 {
			x := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			f.permMask[i] |= own[x]
			for _, y := range inh[x] {
				if seen[y] != int32(i)+1 {
					seen[y] = int32(i) + 1
					stack = append(stack, y)
				}
			}
		}
	}

	// resources: parent links, then an Euler tour for O(1) ancestor tests
	nRes := len(db.Resources)
	children := make([][]int32, nRes)
	var roots []int32
	for i, r := range db.Resources {
		if r.Parent == nil {
			roots = append(roots, int32(i))
			continue
		}
		p, ok := resIdx[*r.Parent]
		if !ok {
			return nil, fmt.Errorf("dangling reference: resource %s parent", r.ID)
		}
		children[p] = append(children[p], int32(i))
	}
	f.tin, f.tout = make([]int32, nRes), make([]int32, nRes)
	type frame struct {
		n int32
		i int
	}
	var timer int32
	visited := 0
	for _, root := range roots {
		stack := []frame{{root, 0}}
		f.tin[root] = timer
		timer++
		visited++
		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			if top.i < len(children[top.n]) {
				c := children[top.n][top.i]
				top.i++
				f.tin[c] = timer
				timer++
				visited++
				stack = append(stack, frame{c, 0})
			} else {
				f.tout[top.n] = timer
				stack = stack[:len(stack)-1]
			}
		}
	}
	if visited != nRes {
		return nil, fmt.Errorf("resource parents contain a cycle")
	}

	// membership
	mk, mv := make([]int32, len(db.Members)), make([]int32, len(db.Members))
	for i, m := range db.Members {
		a, ok1 := subj[m.Member]
		b, ok2 := subj[m.Group]
		if !ok1 || !ok2 || b < int32(f.nUsers) {
			return nil, fmt.Errorf("dangling reference: member %v", m)
		}
		mk[i], mv[i] = a, b
	}
	f.up = buildCSR(nSubj, mk, mv)

	// grants
	n := len(db.Grants)
	f.gID, f.gRole, f.gScope = make([]string, n), make([]int32, n), make([]int32, n)
	var ak, av, dk, dv []int32
	for i, g := range db.Grants {
		s, ok1 := subj[g.Subject]
		r, ok2 := roleIdx[g.Role]
		sc, ok3 := resIdx[g.Scope]
		if !ok1 || !ok2 || !ok3 || (g.Effect != "allow" && g.Effect != "deny") {
			return nil, fmt.Errorf("dangling reference: grant %s", g.ID)
		}
		f.gID[i], f.gRole[i], f.gScope[i] = g.ID, r, sc
		if g.Effect == "allow" {
			ak, av = append(ak, s), append(av, int32(i))
		} else {
			dk, dv = append(dk, s), append(dv, int32(i))
		}
	}
	f.allowBy, f.denyBy = buildCSR(nSubj, ak, av), buildCSR(nSubj, dk, dv)

	// delegations
	m := len(db.Delegations)
	f.dID, f.dFrom = make([]string, m), make([]int32, m)
	f.dRole, f.dScope = make([]int32, m), make([]int32, m)
	tk, tv := make([]int32, m), make([]int32, m)
	for i, d := range db.Delegations {
		from, ok1 := subj[d.From]
		to, ok2 := subj[d.To]
		r, ok3 := roleIdx[d.Role]
		sc, ok4 := resIdx[d.Scope]
		if !ok1 || !ok2 || !ok3 || !ok4 || from >= int32(f.nUsers) || to >= int32(f.nUsers) {
			return nil, fmt.Errorf("dangling reference: delegation %s", d.ID)
		}
		f.dID[i], f.dFrom[i], f.dRole[i], f.dScope[i] = d.ID, from, r, sc
		tk[i], tv[i] = to, int32(i)
	}
	f.delegTo = buildCSR(f.nUsers, tk, tv)

	// requests
	q := len(db.Requests)
	f.qUser, f.qRes, f.qBit = make([]int32, q), make([]int32, q), make([]uint64, q)
	for i, r := range db.Requests {
		u, ok1 := subj[r.User]
		res, ok2 := resIdx[r.Resource]
		if !ok1 || !ok2 || u >= int32(f.nUsers) {
			return nil, fmt.Errorf("dangling reference: request %s", r.ID)
		}
		f.qUser[i], f.qRes[i], f.qBit[i] = u, res, actBit[r.Action] // 0 if no role has it
	}

	f.subjCache, f.subjDone = make([][]int32, f.nUsers), make([]bool, f.nUsers)
	f.memo, f.memoDone = make([][]sup, f.nUsers*(maxHops+1)), make([]bool, f.nUsers*(maxHops+1))
	f.heldCache, f.heldDone = make([][]sup, f.nUsers), make([]bool, f.nUsers)
	f.stamp = make([]uint32, nSubj)
	f.nRes = nRes
	if len(db.Roles)*nRes <= 1<<24 {
		f.heldStamp = make([]uint32, len(db.Roles)*nRes)
		f.heldPos = make([]int32, len(db.Roles)*nRes)
	} else {
		f.heldIdx = map[uint64]int32{}
	}
	f.tmp = make([][]sup, maxHops+1)
	return f, nil
}

func (f *fast) isAnc(scope, res int32) bool {
	return f.tin[scope] <= f.tin[res] && f.tin[res] < f.tout[scope]
}

// subjects returns u plus every group reachable through membership edges.
func (f *fast) subjects(u int32) []int32 {
	if f.subjDone[u] {
		return f.subjCache[u]
	}
	f.epoch++
	out := append(f.subjTmp[:0], u)
	f.stamp[u] = f.epoch
	for i := 0; i < len(out); i++ {
		for _, g := range f.up.row(out[i]) {
			if f.stamp[g] != f.epoch {
				f.stamp[g] = f.epoch
				out = append(out, g)
			}
		}
	}
	f.subjTmp = out
	f.subjDone[u], f.subjCache[u] = true, slices.Clone(out)
	return f.subjCache[u]
}

func (f *fast) chains(u int32, budget int) []sup {
	k := int(u)*(f.maxHops+1) + budget
	if f.memoDone[k] {
		return f.memo[k]
	}
	out := f.tmp[budget][:0]
	for _, s := range f.subjects(u) {
		for _, g := range f.allowBy.row(s) {
			out = append(out, sup{f.gRole[g], f.gScope[g], 0, g, -1})
		}
	}
	if budget > 0 {
		for _, d := range f.delegTo.row(u) {
			role, dscope := f.dRole[d], f.dScope[d]
			for _, s := range f.chains(f.dFrom[d], budget-1) {
				if s.role == role && f.isAnc(s.scope, dscope) {
					f.path = append(f.path, pnode{s.tail, d})
					out = append(out, sup{role, dscope, s.h + 1, s.grant, int32(len(f.path) - 1)})
				}
			}
		}
	}
	f.tmp[budget] = out // keep the grown buffer for the next call at this budget
	f.memoDone[k], f.memo[k] = true, slices.Clone(out)
	return f.memo[k]
}

// appendChain writes the chain string, "G0007>D0003>...", to buf.
func (f *fast) appendChain(buf []byte, s sup) []byte {
	buf = append(buf, f.gID[s.grant]...)
	if s.tail >= 0 {
		var tmp [16]int32
		ds := tmp[:0]
		for t := s.tail; t >= 0; t = f.path[t].prev {
			ds = append(ds, f.path[t].deleg)
		}
		for i := len(ds) - 1; i >= 0; i-- {
			buf = append(buf, '>')
			buf = append(buf, f.dID[ds[i]]...)
		}
	}
	return buf
}

// better reports whether a sorts before b: fewer hops, then the smaller chain
// string. Strings are built only when the hop counts tie.
func (f *fast) better(a, b sup) bool {
	if a.h != b.h {
		return a.h < b.h
	}
	f.bufA = f.appendChain(f.bufA[:0], a)
	f.bufB = f.appendChain(f.bufB[:0], b)
	return bytes.Compare(f.bufA, f.bufB) < 0
}

// held returns, for each (role, scope) the user holds, its best derivation.
func (f *fast) held(u int32) []sup {
	if f.heldDone[u] {
		return f.heldCache[u]
	}
	entries := f.heldTmp[:0]
	if f.heldStamp != nil {
		f.heldEpoch++
		for _, s := range f.chains(u, f.maxHops) {
			k := int(s.role)*f.nRes + int(s.scope)
			if f.heldStamp[k] != f.heldEpoch {
				f.heldStamp[k] = f.heldEpoch
				f.heldPos[k] = int32(len(entries))
				entries = append(entries, s)
			} else if i := f.heldPos[k]; f.better(s, entries[i]) {
				entries[i] = s
			}
		}
	} else {
		clear(f.heldIdx)
		for _, s := range f.chains(u, f.maxHops) {
			k := uint64(uint32(s.role))<<32 | uint64(uint32(s.scope))
			if i, ok := f.heldIdx[k]; !ok {
				f.heldIdx[k] = int32(len(entries))
				entries = append(entries, s)
			} else if f.better(s, entries[i]) {
				entries[i] = s
			}
		}
	}
	f.heldTmp = entries
	f.heldDone[u], f.heldCache[u] = true, slices.Clone(entries)
	return f.heldCache[u]
}

var noIDs = []string{}

func (f *fast) eval(db *DB) []Result {
	out := make([]Result, len(f.qUser))
	for i := range out {
		u, res, bit := f.qUser[i], f.qRes[i], f.qBit[i]
		var best sup
		have := false
		denied := noIDs
		if bit != 0 {
			for _, e := range f.held(u) {
				if f.isAnc(e.scope, res) && f.permMask[e.role]&bit != 0 && (!have || f.better(e, best)) {
					best, have = e, true
				}
			}
			for _, s := range f.subjects(u) {
				for _, g := range f.denyBy.row(s) {
					if f.isAnc(f.gScope[g], res) && f.permMask[f.gRole[g]]&bit != 0 {
						denied = append(denied[:len(denied):len(denied)], f.gID[g])
					}
				}
			}
			if len(denied) > 1 {
				slices.Sort(denied)
			}
		}
		r := Result{DeniedBy: denied}
		if have && len(denied) == 0 {
			via := string(f.appendChain(nil, best))
			r.Allow, r.Via = true, &via
		}
		out[i] = r
	}
	return out
}

// EvalFast evaluates every request of db, in request order.
func EvalFast(db *DB, maxHops int) ([]Result, error) {
	f, err := newFast(db, maxHops)
	if err != nil {
		return nil, err
	}
	return f.eval(db), nil
}
