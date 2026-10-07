// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Object is how the machine holds a JSON object at run time.
//
// A Go map[string]any hashes the member name on every lookup. The members of
// the objects a program reads are almost always few (an event has three or
// four) and almost always named in the program text, so an Object keeps them
// in a short slice and finds one by comparing keys. A key is a Key, a
// canonical handle to its string (unique.Handle[string]): two keys are equal
// if and only if their strings are, and comparing them is one pointer
// comparison, which is about four times quicker than hashing the name. The
// program text's member names are turned into Keys once, when it is compiled;
// the scanner interns the names it reads.
//
// An Object with more than objIndexAt members also keeps a map from name to
// position, so that a large object (an index of customers, say) is not
// scanned. Members are in no particular order: nothing may depend on it, and
// everything that lists them (Show, `keys`, Equal, the host API) uses
// SortedKeys or compares as sets.
//
// The public edges of the package, Run, ParseJSON and the Output it returns,
// still use map[string]any; Data and Legacy convert at those edges.
//
// An Object is not safe for concurrent writes. The machine that owns one is
// the only writer, and a value shared with a reader (input, a literal) is
// never written.

import (
	"sort"
	"unique"
)

// Key is a canonical handle to an object member name.
type Key = unique.Handle[string]

// MakeKey returns the Key for s.
func MakeKey(s string) Key { return unique.Make(s) }

type objEnt struct {
	k Key
	v any
}

// objIndexAt is the member count above which an Object builds its index.
const objIndexAt = 12

// Object is a JSON object value. The zero value is not usable: use NewObject.
type Object struct {
	ents   []objEnt
	idx    map[string]int32 // name -> position in ents; nil while small
	inline [4]objEnt        // backing store for the first few members
}

// NewObject returns an empty object with room for n members.
func NewObject(n int) *Object {
	o := &Object{}
	if n <= len(o.inline) {
		o.ents = o.inline[:0]
	} else {
		o.ents = make([]objEnt, 0, n)
	}
	return o
}

// Len is the number of members. A nil *Object reads as empty, for Len, Get
// and Has.
func (o *Object) Len() int {
	if o == nil {
		return 0
	}
	return len(o.ents)
}

func (o *Object) find(s string) int {
	if o == nil {
		return -1
	}
	if o.idx != nil {
		if i, ok := o.idx[s]; ok {
			return int(i)
		}
		return -1
	}
	for i := range o.ents {
		if o.ents[i].k.Value() == s {
			return i
		}
	}
	return -1
}

func (o *Object) findKey(k Key) int {
	if o.idx != nil {
		if i, ok := o.idx[k.Value()]; ok {
			return int(i)
		}
		return -1
	}
	for i := range o.ents {
		if o.ents[i].k == k {
			return i
		}
	}
	return -1
}

// Get returns the member named s.
func (o *Object) Get(s string) (any, bool) {
	if i := o.find(s); i >= 0 {
		return o.ents[i].v, true
	}
	return nil, false
}

// GetKey is Get with the name already made into a Key: the quick path.
func (o *Object) GetKey(k Key) (any, bool) {
	if i := o.findKey(k); i >= 0 {
		return o.ents[i].v, true
	}
	return nil, false
}

// Has reports whether there is a member named s.
func (o *Object) Has(s string) bool { return o.find(s) >= 0 }

// Set sets the member named s, adding it if there is none.
func (o *Object) Set(s string, v any) {
	if i := o.find(s); i >= 0 {
		o.ents[i].v = v
		return
	}
	o.add(MakeKey(s), v)
}

// SetKey is Set with a Key.
func (o *Object) SetKey(k Key, v any) {
	if i := o.findKey(k); i >= 0 {
		o.ents[i].v = v
		return
	}
	o.add(k, v)
}

// add appends a member that is known not to be present.
func (o *Object) add(k Key, v any) {
	o.ents = append(o.ents, objEnt{k, v})
	if o.idx != nil {
		o.idx[k.Value()] = int32(len(o.ents) - 1)
	} else if len(o.ents) > objIndexAt {
		o.idx = make(map[string]int32, 2*len(o.ents))
		for i := range o.ents {
			o.idx[o.ents[i].k.Value()] = int32(i)
		}
	}
}

// Delete removes the member named s and reports whether there was one.
func (o *Object) Delete(s string) bool {
	i := o.find(s)
	if i < 0 {
		return false
	}
	last := len(o.ents) - 1
	if o.idx != nil {
		delete(o.idx, o.ents[i].k.Value())
		if i != last {
			o.idx[o.ents[last].k.Value()] = int32(i)
		}
	}
	o.ents[i] = o.ents[last]
	o.ents[last] = objEnt{}
	o.ents = o.ents[:last]
	return true
}

// SortedKeys returns the member names in code point order.
func (o *Object) SortedKeys() []string {
	ks := make([]string, len(o.ents))
	for i := range o.ents {
		ks[i] = o.ents[i].k.Value()
	}
	sort.Strings(ks)
	return ks
}

// Range calls fn for each member, in no particular order.
func (o *Object) Range(fn func(name string, v any)) {
	for i := range o.ents {
		fn(o.ents[i].k.Value(), o.ents[i].v)
	}
}

// ---------------------------------------------------------------- conversion

// Data converts a value built from map[string]any into the machine's
// representation (objects as *Object). A value with no map in it is returned
// as it is, not copied; arrays that contain a map are copied. An *Object in
// the value is kept. The argument is not changed.
func Data(v any) any {
	d, _ := toData(v, nil)
	return d
}

func toData(v any, in map[string]Key) (any, bool) {
	switch t := v.(type) {
	case map[string]any:
		if in == nil {
			in = map[string]Key{}
		}
		o := NewObject(len(t))
		for k, x := range t {
			key, ok := in[k]
			if !ok {
				key = MakeKey(k)
				in[k] = key
			}
			d, _ := toData(x, in)
			o.ents = append(o.ents, objEnt{key, d})
		}
		if len(o.ents) > objIndexAt {
			o.idx = make(map[string]int32, 2*len(o.ents))
			for i := range o.ents {
				o.idx[o.ents[i].k.Value()] = int32(i)
			}
		}
		return o, true
	case []any:
		var out []any
		for i, x := range t {
			d, changed := toData(x, in)
			if changed && out == nil {
				out = make([]any, len(t))
				copy(out, t[:i])
			}
			if out != nil {
				out[i] = d
			}
		}
		if out != nil {
			return out, true
		}
	}
	return v, false
}

// Legacy is the inverse of Data: objects become map[string]any, which is what
// Run returns and what callers of the package's public edges hold. A value
// with no *Object in it is returned as it is.
func Legacy(v any) any {
	d, _ := toLegacy(v)
	return d
}

func toLegacy(v any) (any, bool) {
	switch t := v.(type) {
	case *Object:
		m := make(map[string]any, len(t.ents))
		for i := range t.ents {
			d, _ := toLegacy(t.ents[i].v)
			m[t.ents[i].k.Value()] = d
		}
		return m, true
	case []any:
		var out []any
		for i, x := range t {
			d, changed := toLegacy(x)
			if changed && out == nil {
				out = make([]any, len(t))
				copy(out, t[:i])
			}
			if out != nil {
				out[i] = d
			}
		}
		if out != nil {
			return out, true
		}
	}
	return v, false
}
