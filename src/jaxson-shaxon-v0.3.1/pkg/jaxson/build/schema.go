// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package build

// ---------------------------------------------------------------- schemas

// Schema is a contract schema (an inputSchema or outputSchema, or a part
// of one). Sealed; make one with Integer, Number, String, Boolean, Null,
// Anything, Array, Object or AnyOf.
type Schema interface{ schemaNode() map[string]any }

type sch map[string]any

func (s sch) schemaNode() map[string]any { return s }

func (s sch) set(k string, v any) sch {
	n := make(sch, len(s)+1)
	for a, b := range s {
		n[a] = b
	}
	n[k] = v
	return n
}

// NumberSchema is a number schema; Integer() is its integer form.
type NumberSchema struct{ s sch }

// Number is any exact decimal number.
func Number() NumberSchema { return NumberSchema{sch{"type": "number"}} }

// Integer is a number with no fractional part.
func Integer() NumberSchema { return NumberSchema{sch{"type": "number", "int": true}} }

func (n NumberSchema) schemaNode() map[string]any { return n.s }

// Min is the inclusive lower bound.
func (n NumberSchema) Min(v int) NumberSchema { return NumberSchema{n.s.set("min", v)} }

// Max is the inclusive upper bound.
func (n NumberSchema) Max(v int) NumberSchema { return NumberSchema{n.s.set("max", v)} }

// Between sets both inclusive bounds.
func (n NumberSchema) Between(lo, hi int) NumberSchema { return n.Min(lo).Max(hi) }

// Nullable also admits null.
func (n NumberSchema) Nullable() NumberSchema { return NumberSchema{n.s.set("nullable", true)} }

// StringSchema is a string schema.
type StringSchema struct{ s sch }

// String is a string. Length is counted in code points.
func String() StringSchema { return StringSchema{sch{"type": "string"}} }

func (t StringSchema) schemaNode() map[string]any { return t.s }

// MinLen is the minimum length in code points.
func (t StringSchema) MinLen(n int) StringSchema { return StringSchema{t.s.set("minLen", n)} }

// MaxLen is the maximum length in code points.
func (t StringSchema) MaxLen(n int) StringSchema { return StringSchema{t.s.set("maxLen", n)} }

// Enum restricts the value to the listed strings.
func (t StringSchema) Enum(vals ...string) StringSchema {
	a := make([]any, len(vals))
	for i, v := range vals {
		a[i] = v
	}
	return StringSchema{t.s.set("enum", a)}
}

// Nullable also admits null.
func (t StringSchema) Nullable() StringSchema { return StringSchema{t.s.set("nullable", true)} }

// Boolean is true or false.
func Boolean() Schema { return sch{"type": "boolean"} }

// NullOnly admits only null.
func NullOnly() Schema { return sch{"type": "null"} }

// Anything admits any value.
func Anything() Schema { return sch{"type": "any"} }

// AnyOf admits a value matching at least one alternative.
func AnyOf(alts ...Schema) Schema {
	a := make([]any, len(alts))
	for i, s := range alts {
		a[i] = s.schemaNode()
	}
	return sch{"anyOf": a}
}

// ArraySchema is an array schema.
type ArraySchema struct{ s sch }

// Array is an array whose elements match items.
func Array(items Schema) ArraySchema {
	return ArraySchema{sch{"type": "array", "items": items.schemaNode()}}
}

func (a ArraySchema) schemaNode() map[string]any { return a.s }

// MinItems is the minimum element count.
func (a ArraySchema) MinItems(n int) ArraySchema { return ArraySchema{a.s.set("minItems", n)} }

// MaxItems is the maximum element count.
func (a ArraySchema) MaxItems(n int) ArraySchema { return ArraySchema{a.s.set("maxItems", n)} }

// Len fixes the element count exactly.
func (a ArraySchema) Len(n int) ArraySchema { return a.MinItems(n).MaxItems(n) }

// Nullable also admits null.
func (a ArraySchema) Nullable() ArraySchema { return ArraySchema{a.s.set("nullable", true)} }

// ObjectSchema is an object schema. Objects are closed by default.
type ObjectSchema struct{ s sch }

// Object begins an object schema; add members with Field and Req.
func Object() ObjectSchema { return ObjectSchema{sch{"type": "object"}} }

func (o ObjectSchema) schemaNode() map[string]any { return o.s }

func (o ObjectSchema) field(name string, s Schema, required bool) ObjectSchema {
	fields := map[string]any{}
	if old, ok := o.s["fields"].(map[string]any); ok {
		for k, v := range old {
			fields[k] = v
		}
	}
	fields[name] = s.schemaNode()
	n := o.s.set("fields", fields)
	if required {
		var req []any
		if old, ok := o.s["required"].([]any); ok {
			req = append(req, old...)
		}
		n = n.set("required", append(req, name))
	}
	return ObjectSchema{n}
}

// Field declares an optional member.
func (o ObjectSchema) Field(name string, s Schema) ObjectSchema { return o.field(name, s, false) }

// Req declares a required member.
func (o ObjectSchema) Req(name string, s Schema) ObjectSchema { return o.field(name, s, true) }

// Open allows members that are not declared.
func (o ObjectSchema) Open() ObjectSchema { return ObjectSchema{o.s.set("extra", "allow")} }

// Nullable also admits null.
func (o ObjectSchema) Nullable() ObjectSchema { return ObjectSchema{o.s.set("nullable", true)} }
