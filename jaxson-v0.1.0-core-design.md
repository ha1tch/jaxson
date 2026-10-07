# Jaxson v1 core design

Status: proposal. Checked against a throwaway reference interpreter and 28 golden fixtures, all passing (section 14).

## Introduction

Applications often need small pieces of logic that change without a redeploy: a pricing rule, a payload reshaping, a validation step. Embedded scripting languages bring ambient state, I/O and host-specific behaviour. Ad hoc JSON rule formats leave the edges undefined: floating-point drift, missing versus `null`, mutation during iteration. Jaxson is logic expressed as data, small enough to specify completely, that gives the same result, or the same error, on every conforming runtime.

A Jaxson package is one JSON document: the input, an input contract, a program, an output contract, and optionally declared limits. A runtime validates the input, runs the program, then validates the output. The program is a list of instructions from a closed set of eight, working on four named roots, with calculation done in pure compute islands. Numbers are exact decimals, nothing is coerced, contracts reject unknown members, every program terminates, and there is no clock, randomness or I/O.

**Good for:** business rules and calculations (especially where money is involved), payload transformation between two contracts, running logic you did not write, and keeping several runtimes in agreement.

**Not for:** text processing, numerical work, regular expressions or unbounded iteration. Anything beyond the core belongs in later profiles behind explicit extension names.

## 0. Stance

queryfy, jsonplate and ual's compute islands are ancestors, not dependencies. Jaxson takes an idea from each and defines its own semantics, so that the specification is complete without reading anyone's source code.

| Ancestor | Idea kept | What Jaxson does differently |
|---|---|---|
| queryfy | Input and output are gated by declarative contracts | The contract language is JSON-native, closed, and defined in this spec |
| jsonplate | A JSON document that is its own output shape, with holes | Explicit `$tpl` form; absent data is an error unless marked `$opt`, which omits the member |
| ual compute | An island with declared inputs that cannot touch the outside | `$compute` binds its operands up front (`with`), and its body cannot reach state at all |

## 1. The package

```json
{
  "jaxson": "1.0",
  "limits": { "steps": 100000 },
  "input": {},
  "inputSchema": {},
  "program": [],
  "outputSchema": {}
}
```

- `jaxson` is mandatory. A runtime that does not implement that version fails with `VERSION_ERROR` before doing anything else.
- `limits` is optional, and part of the contract (section 9).
- A produced result is not a package member. A test fixture adds one more member, `expect`, holding either `{"output": ...}` or `{"error": {"category": ..., "code": ...}}`. Expected values and produced values never share a name.
- Duplicate object keys anywhere in a package are a `PARSE_ERROR`.

## 2. Values and numbers

Values are JSON values. Numbers are **exact decimals**, not floats.

- A JSON number is read as the exact decimal it spells. `0.1 + 0.2` is `0.3`.
- Magnitude limit: at most 38 significant digits when written as a plain decimal (no exponent). An operation whose exact result exceeds that fails with `NUMERIC_OVERFLOW`. Nothing is ever rounded silently.
- Equality is by value: `1`, `1.0` and `1.00` are the same number. Negative zero does not exist.
- Serialisation is canonical: no exponent, no trailing zeros, no leading `+`.
- The only operations that can lose digits are `div` and `round`, and both take an explicit scale and rounding mode.
- No host numeric type is visible. A Go runtime uses `big.Rat` or a scaled `big.Int`; a Rust runtime uses whatever it likes; results are identical.

## 3. Roots and paths

There are four roots.

| Root | Readable | Writable | Contents |
|---|---|---|---|
| `input` | yes | no | the validated input |
| `state` | yes | yes | starts as `{}` |
| `output` | yes | yes | starts as `null`; its final value is the result |
| `local` | yes | no | loop bindings (`for`), lexically scoped |

A **path** is a JSON array of segments. The first segment is a root name. Later segments are:

- a string, which is an object member;
- a non-negative integer, which is an array index;
- an operand yielding either (a dynamic segment).

```json
["input", "items", 0, "price"]
["input", "items", {"$path": ["local", "i"]}, "price"]
["input"]
```

Consequences:

- Any key is addressable, including `first-name`, `2024`, `$ref`, `a.b` and the empty string.
- A root array input works: `["input", 0]`.
- There is no string path syntax in the core. A surface syntax may offer `input.items[0].price` as sugar and compile to segments.
- Negative indices do not exist. Use `len` in a compute block.
- Reading a missing member or an out-of-range index is `EXECUTION_ERROR` / `MISSING_PATH`. Reading through the wrong container type is `TYPE_ERROR`. Both codes are stable and distinct, so a program's failure can be classified without parsing prose.
- Reads return copies. There is no aliasing anywhere in the language.

## 4. Operand forms: the `$` namespace

Anywhere an instruction wants a value, it takes an **operand**. An operand is one of:

| Operand | Meaning |
|---|---|
| plain JSON | a literal; it must contain no `$` key at any depth |
| `{"$lit": X}` | X, uninterpreted, whatever it contains |
| `{"$path": [...]}` | the value at that path (a copy) |
| `{"$compute": {...}}` | the result of a compute island (section 7) |
| `{"$tpl": T}` | T with its forms substituted (section 5) |

Rules:

1. An object with any key beginning with `$` is a form, and must have exactly one key, and that key must be a known form. Anything else is a `PROGRAM_ERROR`.
2. A plain literal that contains a `$` key anywhere inside is a `PROGRAM_ERROR` with the hint "use `$tpl` or `$lit`". Substitution is therefore never accidental, and never accidentally absent.
3. `$lit` is the escape hatch. `{"$lit": {"$path": ["x"]}}` is the object `{"$path": ["x"]}`.

Jaxson deliberately uses `$path`, not `$ref`. A jsonplate cannot be mistaken for a Jaxson operand, and lifting one into a `$tpl` is a mechanical rename plus a decision about `$opt`.

## 5. Templates

`$tpl` is jsonplate's idea with the sharp edges removed.

```json
{"$tpl": {
  "name":     {"$path": ["input", "name"]},
  "nickname": {"$opt":  ["input", "nickname"]},
  "tags":     ["a", {"$opt": ["input", "tag"]}]
}}
```

- Inside a template, any form is substituted at any depth; everything else is copied. The members of a template object are evaluated in code-point order of their keys, as `$compute` bindings are, so a template with several failing members always fails on the first in that order.
- `{"$opt": path}` is valid only inside a template. If the path resolves, its value is used, **including `null`**. If the path is missing, the member (or array element) is omitted entirely.
- Only `MISSING_PATH` is absorbed. A `TYPE_ERROR` is still an error. A malformed path is caught at program validation, never at run time.
- So the template distinguishes present-`null` from absent, which the jsonplate original cannot.
- A template's root may not itself be `$opt`.

## 6. Instructions

Eight instructions. No labels, no jumps, no `while`.

| `op` | Fields | Semantics |
|---|---|---|
| `set` | `path`, `value` | Evaluate `value`, then the path's segments left to right, then write. The parent must already exist; there is no autovivification. A new object member may be created; an array index must be in range. Setting `["state"]` or `["output"]` replaces the whole root. |
| `delete` | `path` | Remove an object member (must exist) or an array element (later elements shift down). |
| `append` | `path`, `value` | Push onto the array at `path`. |
| `insert` | `path`, `at`, `value` | Insert into the array at `path`; `at` may equal the length. |
| `if` | `cond`, `then`, `else`? | `cond` must be a boolean. No truthiness. |
| `for` | `in`, `as`, `index`?, `do` | Iterate a **snapshot** of an array, in index order. Binds `local.<as>` and optionally `local.<index>`. |
| `assert` | `that`, `msg`? | `that` must be a boolean; false is `ASSERTION_FAILED`. |
| `halt` | none | Successful early termination. |

Notes:

- `for` iterates a copy taken when the loop starts, so mutating the collection in the body cannot change the iteration. This is the whole answer to "mutation during iteration".
- A `for` binding name must not collide with an active binding. Shadowing is a `PROGRAM_ERROR`, so `local.x` never depends on scope rules.
- `for` over an object is not possible. Iterate `["keys", obj]`, which is sorted by code point.
- Unknown fields on an instruction are a `PROGRAM_ERROR`; nothing is ignored.

## 7. Compute islands

```json
{"$compute": {
  "with": { "t": {"$path": ["state", "total"]}, "p": {"$path": ["local", "item", "price"]} },
  "expr": ["add", {"$v": "t"}, {"$v": "p"}]
}}
```

- `with` binds names to operands. It is the only place the island touches the outside, and it is resolved in name order (code point).
- `expr` is a small expression: a scalar literal, a binding `{"$v": name}`, or an application `["op", arg, ...]`. An `expr` cannot contain `$path`, `$tpl` or `$compute`; it has no way to read state. Purity is structural, and a validator can list exactly which paths a program reads.
- Bare strings in `expr` are string literals, never variable names. Arrays are always applications; build an array with `["list", ...]`.
- Type-strict: no coercion anywhere. `["add", "1", 2]` is `TYPE_ERROR`.

Operations (v1 closed set):

| Group | Operations |
|---|---|
| arithmetic | `add` `sub` `mul` (2 or more), `neg` `abs`, `min` `max` (1 or more), `mod` (integers, sign of dividend, zero divisor is `DIV_ZERO`) |
| division and rounding | `["div", a, b, scale, mode?]`, `["round", x, scale, mode?]`. Scale 0 to 38. Modes: `half_even` (default), `half_up`, `down` (toward zero) |
| comparison | `eq` `ne` (deep, any types), `lt` `le` `gt` `ge` (two numbers or two strings; strings by code point) |
| logic | `and` `or` `not`; booleans only; `and` and `or` short-circuit left to right, `select` (`cond`, `then`, `else`) evaluates only the chosen branch |
| strings | `concat`, `len` (code points), `to_string` (canonical number), `to_number` (strict JSON grammar) |
| collections | `list`, `get` (missing is `MISSING_PATH`), `get_or`, `has`, `keys` (sorted), `len`, `type_of` |

Not included, on purpose: loops, locale-dependent string operations, floating point, transcendental functions, regular expressions. A later profile can add them behind explicit extension names; the core stays total and portable.

## 8. Contracts

Schemas are JSON documents in a closed vocabulary. They are pure predicates: no coercion, no defaults, no transformation. The validated document is byte-for-byte the document the program sees.

| Schema | Keywords |
|---|---|
| `{"type":"any"}` | none |
| `{"type":"null"}`, `{"type":"boolean"}` | none |
| `{"type":"number"}` | `int`, `min`, `max` |
| `{"type":"string"}` | `minLen`, `maxLen`, `enum` |
| `{"type":"array"}` | `items`, `minItems`, `maxItems` |
| `{"type":"object"}` | `fields`, `required`, `extra` (`"reject"` by default, or `"allow"`) |
| `{"anyOf":[...]}` | alternatives |

- Any schema may add `"nullable": true`.
- Optional and nullable are independent. A field may be required and nullable, optional and non-nullable, and so on. Present-`null` and absent are never conflated.
- Objects are closed by default. The contract says what may be there; anything else is rejected.
- An unknown type or keyword is a `SCHEMA_ERROR`, detected before input validation. Nothing is skipped or approximated.
- Validation reports the first failure in a fixed order: required names in listed order, then members in code-point order, each either validated against its field schema or rejected as unexpected. Diagnostics are therefore reproducible.
- Deliberately absent: `pattern`. Regex engines disagree at the edges. If it is added, it is a named profile with a fixed grammar.

A worthwhile v1.1 idea rather than a v1 obligation: since schemas are data and missing reads are errors, a validator can prove some reads safe (every step lands on a `required` member) and can prove which output members a program can never set. The design keeps that door open.

## 9. Determinism, limits and termination

- Every control construct is structured, and every loop iterates a finite snapshot, so every program terminates. Running time is bounded by the program size and input size.
- The **step count** is the semantic limit. One step per instruction executed, per loop iteration begun, and per compute operator applied. Exceeding the declared `limits.steps` is `RESOURCE_ERROR`, at exactly the same point on every runtime.
- Limits are part of the contract. A runtime either accepts the declared limits or rejects the package before starting. It never applies a limit the package did not declare, so a time or memory cutoff cannot change a result.
- Default `steps` is 100000 when `limits` is absent.
- No clock, no randomness, no ambient state. Object member order is never observable: `keys` is sorted, and equality ignores order.

## 10. Errors

Each failure has a category and, for execution errors, a stable code.

| Category | Raised when | Codes |
|---|---|---|
| `PARSE_ERROR` | Not JSON, or duplicate keys | |
| `VERSION_ERROR` | `jaxson` missing or unsupported; unsupported limits | |
| `SCHEMA_ERROR` | A contract schema is malformed | |
| `PROGRAM_ERROR` | Static validation fails | |
| `INPUT_ERROR` | Input violates `inputSchema` | |
| `EXECUTION_ERROR` | A valid program fails | `MISSING_PATH` `TYPE_ERROR` `BAD_INDEX` `DIV_ZERO` `NUMERIC_OVERFLOW` `ASSERTION_FAILED` `BAD_MODE` `BAD_SCALE` `BAD_NUMBER` |
| `RESOURCE_ERROR` | Declared step limit exceeded | `STEPS` |
| `OUTPUT_ERROR` | Output violates `outputSchema` | |

Order: parse, version, schemas, program validation, input, execution, output. An earlier stage's failure means later stages do not run.

Static validation (`PROGRAM_ERROR`) covers: unknown instruction, missing or unknown fields, unknown form, a literal containing a `$` key, `$opt` outside a template, a write to `input` or `local`, a path with a bad root or a negative or fractional literal index, an undeclared `local`, a shadowed `for` binding, an undeclared `$v`, an unknown compute operation, and wrong arity.

## 11. Worked example

A summing program, taken from the earlier draft (its Part 2.48):

```json
[
  {"op": "set", "path": ["state", "total"], "value": 0},
  {"op": "for", "in": {"$path": ["input", "items"]}, "as": "item", "do": [
    {"op": "set", "path": ["state", "total"], "value": {"$compute": {
      "with": {"t": {"$path": ["state", "total"]}, "p": {"$path": ["local", "item", "price"]}},
      "expr": ["add", {"$v": "t"}, {"$v": "p"}]}}}
  ]},
  {"op": "set", "path": ["output"], "value": {"$tpl": {"total": {"$path": ["state", "total"]}}}}
]
```

A non-normative surface syntax could make it readable and compile to exactly the above:

```
state.total = 0
for item in input.items {
  state.total = compute(t = state.total, p = item.price) { t + p }
}
output = { total: state.total }
```

The surface form is a compiler concern, and can borrow ual's look freely.

## 12. Mapping to the earlier five-part draft

This document replaces or fills in parts of an earlier five-part draft, which is a separate document. The left column gives section numbers in that draft.

| Earlier draft | Change |
|---|---|
| 1.2, 3.4, 3.24 | Package members fixed as in section 1; `jaxson` mandatory; `expect` for fixtures; "queryfy schema" becomes the contract schema of section 8 |
| 1.11, 2.23 to 2.27, 3.15, 3.16, 3.43 | Compute is the island of section 7, with a closed operation set |
| 1.9, 2.5 to 2.8, 3.9 to 3.14 | Roots, segment-array paths, `$` forms and `$opt` replace the open reference grammar; `$ref` retired |
| 1.13, 2.19 to 2.22 | Snapshot iteration, no shadowing |
| 2.35 to 2.36, 3.21 | Labels and jumps deleted from v1 |
| 2.11, 2.30, 2.32 | Mutation semantics as in section 6 (including shifting on array delete) |
| 1.19, 4.34 to 4.39, 5.34 to 5.35 | Limits move into the contract; step counting is semantic |
| 4.30 | Adds `VERSION_ERROR` and `SCHEMA_ERROR` |
| 4.19, 4.65, 5.21 | Numeric model of section 2 |
| Parts 4 and 5 | Roughly half can be cut; the Jaxson-specific normative content is the lifecycle order, error categories and the fixture format |

## 13. Open questions

The points where the design is least settled.

1. **Exact decimals with a 38-digit bound** versus IEEE doubles. Decimals are the safer default for a determinism claim and for anything that touches money, but they make a JavaScript runtime more work.
2. **Closed objects by default.** It makes contracts stricter than most people expect. The strictness is intended for a contract language, but it will surprise authors used to JSON Schema.
3. **`$opt` omitting the member.** It is the neatest way found so far to keep present-`null` and absent apart in output, but it is a new idea and deserves fixtures from real payloads before it is frozen.
4. **No string path syntax in the core.** It costs verbosity for a gain in unambiguity, and rests on a surface syntax being written.
5. **Step accounting.** One step per operator application is arbitrary but deterministic. The exact table needs to be part of the spec, and this proposal fixes it only by example.

## 14. The checking harness

- `jaxrun.go`: about 1,300 lines of Go (`math/big` for numbers, no dependencies), a deliberately plain interpreter of the above. It is a specification check, not a product.
- `fixtures.json`: 28 golden fixtures in the package format with `expect`. All 28 pass. I confirmed that each failure fixture fails for its intended reason (the runner prints the message), and that corrupting an expected value makes the runner fail.

What the fixtures cover: the summing example, `$tpl` with `$opt` (absent omitted, `null` kept), null versus missing, exact decimals and all three rounding modes, closed objects, input and output contract failures, snapshot iteration, nested loops with `index`, binding shadowing, literal-contains-form, `$lit`, the declared step limit, array delete shifting, short-circuit and lazy `select`, dynamic path segments, `halt`, code-point length, a root-array input, keys no string path could name, write-to-input, unknown instruction, unsupported version, malformed schema, no coercion, numeric overflow, and `$opt` outside a template.

What is implemented in the runner but has **no fixture yet**: `insert`, `assert`, `if`, `sub`, `neg`, `abs`, `min`, `max`, `mod`, `lt` `le` `gt` `ge`, `not`, `to_number`, `keys`, `get`, `type_of`, `list`, and the `anyOf`, `enum`, `min`, `max`, `minLen`, `maxLen`, `minItems`, `maxItems` and `extra: "allow"` schema keywords. What the runner does **not** implement: duplicate-key detection (so `PARSE_ERROR` for duplicates is specified but untested) and the `limits` rejection path beyond `steps`. Treat those as the next fixtures to write.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
