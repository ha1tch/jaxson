// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

import (
	"bytes"
	"hash/maphash"
	"math"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// A string-keyed index whose key is a `concat` island (the usual composite
// key, "actor|company") is built without making a string per element. Each
// key is written to one byte slab, looked up in an open-addressing table
// kept in parallel arrays (hash, offset, length, first element), and dropped
// from the slab again if it was seen before. Only the distinct keys become
// entries, and they are carved out of three allocations: one string for the
// whole slab, one block of entries and one array of positions.

var slabSeed = maphash.MakeSeed()

// slabTable is the lookup side of such an index, kept after the build: slots
// holds entry number plus one (zero is empty), and hash gives the full hash of
// each entry so the table can grow without rehashing the keys.
type slabTable struct {
	slots []int32
	hash  []uint64
	block []indexEntry
	mask  uint64
}

func (t *slabTable) find(key string) (*indexEntry, bool) {
	h := maphash.String(slabSeed, key)
	for i := h & t.mask; ; i = (i + 1) & t.mask {
		s := t.slots[i]
		if s == 0 {
			return nil, false
		}
		if t.hash[s-1] == h && t.block[s-1].flag.text == key {
			return &t.block[s-1], true
		}
	}
}

// find returns the entry for flag, from whichever table the index has.
func (d *indexData) find(flag flagKey) (*indexEntry, bool) {
	if d.tab != nil {
		if flag.kind != 's' {
			return nil, false
		}
		return d.tab.find(flag.text)
	}
	e, ok := d.byKey[flag]
	return e, ok
}

// buildSlab is build for a key that fn appends to a slab. The order of
// charges and failures is that of build: per element, the element charge,
// then the key (its bindings, then the operator), then the repeat check.
func (s *IndexSet) buildSlab(sp indexSpec, srcPath []any, arr []any, fn func([]byte) []byte) *indexData {
	d := &indexData{spec: sp, srcPath: srcPath}
	if a := s.ambient; a != nil {
		old := *a
		defer func() { *a = old }()
	}
	var (
		slab  = make([]byte, 0, 16*len(arr)+64)
		slots = make([]int32, 16)
		mask  = uint64(len(slots) - 1)
		hash  []uint64 // per distinct key, in order of first appearance
		off   []int32
		ln    []int32
		first []int32
		count []int32
		elem  = make([]int32, len(arr)) // the key number of each element
	)
	s.m.WithLocalSet("item", func(set func(any)) {
		for i, el := range arr {
			s.m.ChargeEvent(EventIndexElement, 0)
			if a := s.ambient; a != nil {
				if srcPath == nil {
					*a = Ambient{}
				} else {
					*a = Ambient{src: srcPath, at: i, lazy: true}
				}
			}
			set(el)
			start := len(slab)
			slab = fn(slab)
			key := slab[start:]
			h := maphash.Bytes(slabSeed, key)
			p := h & mask
			for ; slots[p] != 0; p = (p + 1) & mask {
				id := slots[p] - 1
				if hash[id] == h && bytes.Equal(slab[off[id]:off[id]+ln[id]], key) {
					break
				}
			}
			if id := slots[p] - 1; id >= 0 {
				if !sp.multi {
					jaxson.Fail(CatShapeError, "", "%s: key %s repeats (elements %d and %d); an index is a function unless it declares multi", sp.label, jaxson.Show(string(key)), first[id], i)
				}
				slab = slab[:start]
				count[id]++
				elem[i] = id
				continue
			}
			id := int32(len(hash))
			slots[p] = id + 1
			hash = append(hash, h)
			off = append(off, int32(start))
			ln = append(ln, int32(len(key)))
			first = append(first, int32(i))
			count = append(count, 1)
			elem[i] = id
			if uint64(len(hash))*2 > uint64(len(slots)) {
				slots = make([]int32, 2*len(slots))
				mask = uint64(len(slots) - 1)
				for j, hj := range hash {
					q := hj & mask
					for slots[q] != 0 {
						q = (q + 1) & mask
					}
					slots[q] = int32(j) + 1
				}
			}
		}
	})
	nd := len(hash)
	all := string(slab)
	block := make([]indexEntry, nd)
	d.ordered = make([]*indexEntry, nd)
	posAt := make([]int, len(arr))
	var at int32
	for id := range block {
		e := &block[id]
		e.flag = flagKey{'s', all[off[id] : off[id]+ln[id]]}
		e.pos = posAt[at : at : at+count[id]]
		at += count[id]
		d.ordered[id] = e
	}
	for i, id := range elem {
		e := &block[id]
		e.pos = append(e.pos, i)
	}
	d.tab = &slabTable{slots: slots, hash: hash, block: block, mask: mask}
	return d
}

// slabOK says whether positions and offsets fit the compact arrays.
func slabOK(n int) bool { return n < math.MaxInt32/64 }
