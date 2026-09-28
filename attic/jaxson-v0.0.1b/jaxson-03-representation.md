# 3. Representation

## 3.1 Overview

Jaxson programs and their associated execution contracts are represented using JSON.

Part 2 defines the semantics of the Jaxson language independently of its representation. This part defines how those semantics are encoded as JSON documents.

The representation is intended to satisfy four requirements:

1. **Unambiguity** — a valid Jaxson document MUST have one unambiguous interpretation.
2. **Inspectability** — a Jaxson program SHOULD remain understandable as ordinary JSON.
3. **Machine validation** — the representation MUST be amenable to structural validation.
4. **Separation of concerns** — program structure, execution data, validation contracts and produced output MUST remain distinguishable.

The representation MUST NOT introduce semantics which are absent from the language definition.

A conforming implementation MAY internally transform a Jaxson program into another representation before execution, provided that the observable behaviour remains consistent with this specification.

---

# 3.2 Representation Units

A Jaxson execution consists of five logical JSON documents:

```text
input
input schema
program
output schema
output
```

These documents MAY be stored independently or combined into a single package.

The logical relationship is:

```text
┌───────────────┐
│     Input     │
└───────┬───────┘
        │
        ▼
┌───────────────────┐
│  Input Schema     │
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│  Jaxson Program   │
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│     Output        │
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│  Output Schema    │
└───────────────────┘
```

The output does not need to exist before execution.

For an executable test or reproducibility fixture, however, an expected output document MAY be supplied alongside the program and contracts.

---

# 3.3 Separate Representation

A Jaxson execution MAY be represented using separate files.

A conventional layout is:

```text
input.json
input.queryfy.json
program.jaxson.json
output.queryfy.json
output.json
```

The filenames are not themselves part of the language.

An implementation MAY use different filenames or storage mechanisms.

The important property is that each logical component remains independently identifiable.

For example:

```text
input.json
```

contains only the input document.

```text
input.queryfy.json
```

contains the input validation schema.

```text
program.jaxson.json
```

contains the Jaxson program.

```text
output.queryfy.json
```

contains the output validation schema.

```text
output.json
```

contains the resulting or expected output document.

---

# 3.4 Bundled Representation

The same five logical components MAY be represented by a single JSON document.

A bundled Jaxson document contains named members corresponding to:

* input;
* input schema;
* program;
* output schema; and
* output.

Conceptually:

```json
{
  "input": {},
  "inputSchema": {},
  "program": [],
  "outputSchema": {},
  "output": {}
}
```

The exact member names defined by the canonical Jaxson representation are normative.

The example illustrates the structure rather than defining alternative spellings.

A bundled document MUST NOT duplicate the same logical component under multiple names.

An implementation MAY support additional metadata members where those members are explicitly defined as non-semantic metadata.

---

# 3.5 Program Representation

A Jaxson program is represented as an ordered JSON array of instructions.

Conceptually:

```json
[
  {
    "op": "..."
  },
  {
    "op": "..."
  }
]
```

The array order defines the initial sequential execution order.

Each element of the program array MUST represent exactly one instruction.

An instruction MUST be represented as a JSON object.

A primitive JSON value such as a string or number MUST NOT itself constitute an instruction.

This distinction allows ordinary JSON values to remain data while instruction objects represent executable operations.

---

# 3.6 Instruction Objects

An instruction object contains an operation identifier and the fields required by that operation.

The canonical operation member is:

```text
op
```

Conceptually:

```json
{
  "op": "set",
  "...": "operation-specific fields"
}
```

The value of `op` MUST be a string.

The operation identifier determines the instruction semantics.

An instruction MUST NOT contain an ambiguous combination of operation identifiers.

For example, an instruction MUST NOT attempt to specify both:

```json
{
  "op": "set",
  "operation": "compute"
}
```

unless the language explicitly defines such fields as ordinary metadata.

---

# 3.7 Instruction Fields

Instruction-specific fields are defined by Part 2.

Fields which are not recognised by the relevant instruction definition MUST NOT silently alter its semantics.

The specification SHOULD distinguish between:

* required fields;
* optional fields;
* default values;
* metadata;
* extension fields.

A conforming implementation MUST reject structurally invalid instructions rather than guessing their intended meaning.

---

# 3.8 Program Structure and Execution Structure

The JSON representation of a program is not necessarily identical to its runtime representation.

For example, a program represented as:

```json
[
  { "op": "set" },
  { "op": "if" },
  { "op": "compute" }
]
```

may internally be converted to an instruction table, control-flow graph, bytecode sequence or other representation.

The JSON array remains the canonical external representation.

The program counter described in Part 2 is therefore an execution concept, not necessarily a literal property of the JSON document.

A Jaxson document MUST NOT require an implementation to expose internal runtime structures merely to execute the program.

---

# 3.9 Operands

Operands are represented explicitly.

A Jaxson operand MUST be distinguishable as one of the following conceptual categories:

1. literal value;
2. reference;
3. computation;
4. instruction-specific value.

This distinction prevents ordinary JSON data from accidentally acquiring executable semantics.

For example, a literal string:

```json
"hello"
```

remains the string `"hello"`.

A reference to a value must have an explicit representation.

A computed value must likewise have an explicit representation.

The representation MUST NOT rely on guessing whether a string "looks like" a path or expression.

---

# 3.10 Literal Values

A literal is a JSON value embedded directly in the program.

Examples include:

```json
42
```

```json
"hello"
```

```json
true
```

```json
null
```

```json
[1, 2, 3]
```

and:

```json
{
  "name": "example"
}
```

A literal is data, not executable code.

A literal object MUST therefore not become an instruction merely because it contains members resembling instruction fields unless it occurs in a position defined to contain an instruction.

Context determines whether an object is an instruction or ordinary data.

---

# 3.11 References

A reference MUST have a representation which unambiguously identifies it as a reference rather than a literal object.

The canonical representation SHOULD use a dedicated reference member.

Conceptually:

```json
{
  "$ref": "path.to.value"
}
```

The precise reference grammar is inherited from the Jaxson/queryfy integration defined in Part 2.

A reference object MUST contain only the members permitted by the reference definition.

An object containing `$ref` together with unrelated data MUST NOT silently be interpreted as a reference unless the language explicitly defines such a form.

This preserves the useful property established by jsonplate that executable reference syntax is structurally unambiguous.

---

# 3.12 Reference Contexts

A reference path is interpreted relative to a defined context.

Jaxson MUST define the contexts available to references.

At minimum, the representation needs to distinguish between:

* input data;
* mutable working state;
* output state, where applicable;
* iteration context.

A reference MUST NOT depend upon undocumented assumptions about which context is searched first.

If multiple contexts are available, the reference syntax SHOULD make the selected context explicit or establish a single deterministic resolution rule.

---

# 3.13 Reference Paths

Reference paths use the path semantics established by queryfy.

A path MAY identify:

* an object member;
* a nested object member;
* an array element;
* a nested combination of objects and arrays.

For example, a conceptual path may identify:

```text
items[0].price
```

The representation of the path remains data.

The reference wrapper gives the path executable meaning.

The Jaxson specification MUST NOT create a second incompatible path language merely for instruction operands.

---

# 3.14 Destination References

Instructions which mutate state require a destination.

A destination is represented separately from an ordinary value operand because it identifies **where** state is to be modified.

Conceptually:

```json
{
  "op": "set",
  "target": {
    "...": "destination"
  },
  "value": {
    "...": "operand"
  }
}
```

The target is not itself a value to be assigned.

It identifies a mutable location.

A target MUST therefore be validated according to destination semantics rather than ordinary reference semantics.

---

# 3.15 Compute Representation

A `compute` block is represented explicitly as a computational construct.

It MUST be distinguishable from ordinary instruction operands.

Conceptually:

```json
{
  "compute": {
    "...": "computation"
  }
}
```

The exact computational representation is defined by the language semantics.

A computation MAY contain:

* literal operands;
* references;
* nested computational operations where permitted;
* an operator;
* operation-specific parameters.

Arithmetic MUST NOT be encoded as an implicit property of an ordinary instruction.

For example, an assignment such as:

```text
set x = a + b
```

is not represented by placing an arbitrary arithmetic expression in the value field.

Instead, the representation explicitly identifies the computation.

---

# 3.16 Computation Trees

Where computation permits nested operations, the representation forms a tree.

Conceptually:

```text
          add
         /   \
       ref   multiply
       a     /      \
            ref    literal
             b        2
```

This could be represented as nested JSON objects.

The representation MUST preserve the distinction between:

* operator;
* operand;
* reference;
* literal.

A computation tree is evaluated according to the semantics of Part 2.

The representation MUST NOT introduce implicit evaluation rules based merely on JSON nesting.

---

# 3.17 Blocks

Structured instructions may contain blocks of instructions.

A block is represented as a JSON array.

Conceptually:

```json
{
  "op": "if",
  "condition": {},
  "then": [
    {},
    {}
  ],
  "else": [
    {}
  ]
}
```

Each element of a block MUST be a valid instruction.

Blocks may appear wherever the language defines a structured instruction body.

A block is not itself a program unless it occurs in a context where a program or instruction sequence is expected.

---

# 3.18 Conditional Representation

A conditional instruction contains:

* a condition;
* a body to execute when the condition is true;
* optionally, a body to execute when it is false.

Conceptually:

```json
{
  "op": "if",
  "condition": {},
  "then": [],
  "else": []
}
```

The `condition` member represents an operand which MUST resolve to a boolean.

The `then` and `else` members contain instruction arrays.

The absence of an `else` block means that no alternative instruction sequence is executed when the condition is false.

A conditional MUST NOT evaluate both branches.

---

# 3.19 Iteration Representation

An iteration instruction identifies:

* the collection;
* the iteration context;
* the body.

Conceptually:

```json
{
  "op": "for",
  "in": {},
  "as": "...",
  "do": []
}
```

The collection operand identifies the JSON array to iterate.

The iteration variable identifies the current element within the iteration body.

The body contains an ordered array of instructions.

The exact member names are defined by the canonical representation.

The representation MUST make the collection, iteration variable and body structurally unambiguous.

---

# 3.20 Nested Blocks

Blocks MAY contain instructions which themselves contain blocks.

For example:

```text
program
  └── iteration
       └── conditional
            └── compute
```

The representation must preserve this nesting.

Nested blocks MUST NOT require special syntax distinct from ordinary JSON structure beyond the fields defined by their containing instruction.

This allows the representation to remain structurally regular.

---

# 3.21 Labels and Control Targets

If the language exposes labels, a label is represented as a distinct program construct or instruction.

Conceptually:

```json
{
  "op": "label",
  "name": "loop"
}
```

A control-flow instruction may then identify that target.

Conceptually:

```json
{
  "op": "jump",
  "to": "loop"
}
```

Labels MUST be unique within their defined program scope.

A target which cannot be resolved is a program-definition error.

The representation MUST NOT permit ambiguous targets.

If structured control flow is used instead of explicit labels and jumps, this section does not require labels to exist. The representation model must nevertheless provide whatever target mechanism Part 2 defines.

---

# 3.22 Program Metadata

A Jaxson program MAY contain metadata.

Metadata may include:

* program name;
* version;
* description;
* author;
* documentation;
* implementation hints;
* debugging information.

Metadata MUST NOT alter program semantics unless the specification explicitly defines it as semantic.

This distinction is important because metadata may safely be added, removed or changed without changing the meaning of a program.

A conforming implementation MUST NOT execute arbitrary metadata fields as instructions.

---

# 3.23 Package Metadata

A bundled Jaxson document MAY contain package metadata in addition to the five execution components.

Metadata may describe:

* package identity;
* specification version;
* program version;
* creation information;
* descriptive information;
* provenance.

Metadata is not part of the execution state unless explicitly defined as such.

The canonical package representation MUST clearly distinguish metadata from:

```text
input
input schema
program
output schema
output
```

---

# 3.24 Schema Representation

The input and output schemas are queryfy documents.

Jaxson does not redefine the queryfy schema language.

The schema is embedded or referenced according to the package representation.

Conceptually:

```json
{
  "inputSchema": {
    "...": "queryfy schema"
  },
  "outputSchema": {
    "...": "queryfy schema"
  }
}
```

The contents of those members are interpreted according to queryfy.

A Jaxson implementation MUST NOT reinterpret a queryfy schema according to unrelated Jaxson semantics.

---

# 3.25 Input Representation

The `input` member of a bundled package contains the complete input JSON document.

It is data, not an instruction sequence.

For example:

```json
{
  "input": {
    "customer": {
      "id": 123
    }
  }
}
```

The contents of `input` are interpreted according to the input schema and the reference semantics available to the program.

The input MUST NOT contain executable Jaxson instructions merely because some of its values happen to resemble instruction objects.

---

# 3.26 Output Representation

The `output` member of a complete execution package contains the resulting JSON document.

In a test fixture, it MAY contain an expected output.

In a package used solely to execute a program, the output member MAY be absent before execution if the package format permits an incomplete execution package.

The distinction between:

* required input;
* optional expected output;
* produced output

MUST be explicit in the package definition.

An implementation MUST NOT confuse an expected output fixture with a value which the program is required to consume.

---

# 3.27 Complete Bundled Document

A complete bundled representation can conceptually be expressed as:

```json
{
  "input": <JSON value>,
  "inputSchema": <queryfy schema>,
  "program": [
    <instruction>,
    <instruction>,
    ...
  ],
  "outputSchema": <queryfy schema>,
  "output": <JSON value>
}
```

The five members have distinct roles:

| Member         | Meaning                        |
| -------------- | ------------------------------ |
| `input`        | Input data                     |
| `inputSchema`  | Input contract                 |
| `program`      | Executable Jaxson instructions |
| `outputSchema` | Output contract                |
| `output`       | Produced or expected output    |

The canonical specification MUST define whether all five members are required for a complete package and which may be omitted in intermediate execution states.

---

# 3.28 Program-Only Representation

A Jaxson program may be distributed independently of its execution data.

For example:

```json
[
  {
    "op": "..."
  }
]
```

Such a document is a valid **program representation**, but it is not by itself a complete execution contract.

A runtime executing a program-only document MUST receive the missing execution components through another defined mechanism.

This distinction is important:

> A Jaxson program can exist independently as code, but a Jaxson execution is defined by the complete contract.

---

# 3.29 Structural Validation

A Jaxson representation MUST be structurally validated before execution.

Structural validation determines whether the JSON document is a valid representation of the Jaxson language.

Examples include:

* invalid instruction objects;
* unknown required operation;
* missing instruction fields;
* invalid operand representation;
* malformed blocks;
* duplicate labels;
* unresolved control targets;
* malformed references.

Structural validation is distinct from input and output validation.

The three validation layers are:

```text
Jaxson representation
        │
        ▼
Program validation
        │
        ▼
Input validation
        │
        ▼
Execution
        │
        ▼
Output validation
```

---

# 3.30 Validation of Instructions

Each instruction has a defined structural schema.

A validator SHOULD determine, before execution where possible:

* whether the operation exists;
* whether required fields exist;
* whether fields have valid JSON types;
* whether mutually exclusive fields are used correctly;
* whether nested instructions are valid;
* whether references have valid structure;
* whether control-flow targets can be resolved.

An implementation SHOULD fail program validation before executing an invalid program.

This prevents runtime behaviour from depending on how far an invalid program happened to execute before encountering its defect.

---

# 3.31 Unknown Operations

An operation identifier which is not recognised by the implementation is invalid unless the implementation explicitly supports a defined extension mechanism.

An implementation MUST NOT silently ignore an unknown operation.

Ignoring an operation would alter program semantics while potentially appearing to the caller as successful execution.

Unknown operations therefore constitute a program-definition error.

---

# 3.32 Extensions

Jaxson MAY support extensions.

An extension mechanism MUST distinguish extension semantics from core language semantics.

An implementation MUST NOT treat an implementation-specific operation as part of portable Jaxson unless that operation is defined by a specification extension or implementation profile.

Extensions SHOULD be namespaced or otherwise identifiable where there is a risk of collision with future core operations.

A program using non-portable extensions is not necessarily invalid, but its portability MUST be explicitly identifiable.

---

# 3.33 Canonical Serialisation

JSON permits insignificant variation in whitespace and member ordering.

Two Jaxson documents which differ only in JSON formatting represent the same document.

For example:

```json
{"op":"set","target":"x","value":1}
```

and:

```json
{
  "value": 1,
  "target": "x",
  "op": "set"
}
```

represent equivalent JSON objects, assuming both satisfy the same Jaxson structural requirements.

Program instruction order, however, is significant because instructions are stored in an array.

Array order MUST therefore be preserved.

---

# 3.34 Numbers

Jaxson inherits JSON number representation at the document boundary.

The precise numeric semantics used by `compute` are defined separately from JSON parsing.

In particular, a runtime MUST NOT assume that the numeric semantics of its implementation language are automatically the numeric semantics of Jaxson.

If Jaxson specifies integer, decimal or arbitrary-precision behaviour, the representation remains JSON while the computation semantics define how those numbers are interpreted.

This separation permits a runtime to use an appropriate internal representation without exposing implementation-specific numeric behaviour.

---

# 3.35 Strings

Strings used as ordinary JSON values remain ordinary values.

Strings used as:

* operation identifiers;
* labels;
* variable names;
* paths;
* metadata

acquire their specialised meaning from their structural position.

A string MUST NOT acquire executable meaning solely because of its contents.

For example, the string:

```json
"items[0].price"
```

is an ordinary string unless it occurs in a position defined as a path or reference.

This is an important property of the representation.

---

# 3.36 Null Representation

JSON `null` is represented normally.

The representation MUST preserve the distinction between:

```json
null
```

and:

```text
missing member
```

A missing instruction field and an instruction field explicitly containing `null` are not automatically equivalent.

Whether a particular field permits `null` is determined by the instruction definition.

---

# 3.37 Empty Structures

An empty program is represented by an empty array:

```json
[]
```

An empty instruction block is similarly represented by an empty array.

An empty object:

```json
{}
```

is an ordinary JSON object and MUST NOT be interpreted as an empty instruction.

This distinction prevents structural ambiguity.

The semantics of an empty program are defined by Part 2.

---

# 3.38 Representation of Errors

Errors are not represented as ordinary program values unless the language explicitly introduces an error value.

A structural error in a Jaxson document is not equivalent to:

```json
{
  "error": "..."
}
```

unless such an object is the declared output of a successfully executing program.

This distinction prevents failures from being confused with program output.

Diagnostic information MAY be supplied by the runtime outside the program's output document.

---

# 3.39 Representation of Execution Context

Execution context is runtime state and is not necessarily serialised into the program.

For example, the following are generally execution concepts rather than program data:

* program counter;
* current block;
* current iteration;
* call stack, if one exists;
* resource counters;
* runtime cancellation state.

An implementation MAY serialise execution state for debugging or checkpointing, but such a serialisation is not part of the canonical program representation unless explicitly defined.

---

# 3.40 Representation and Security

A Jaxson document is data, but it contains executable semantics.

Implementations MUST therefore treat Jaxson documents as executable input rather than inert configuration.

Parsing a Jaxson document MUST NOT by itself invoke arbitrary external operations.

Execution capabilities are defined by the Jaxson language and runtime.

An implementation MUST NOT infer permission to access external resources merely because a JSON document contains a string representing a URL, filename, command or other external identifier.

---

# 3.41 Representation and Reproducibility

A Jaxson package SHOULD be capable of being retained as a complete reproducibility artefact.

A complete fixture may therefore contain:

```text
input
input schema
program
output schema
expected output
```

Such a fixture can be used to test whether an implementation produces the expected result.

The representation should make it possible to compare:

```text
actual output
```

with:

```text
expected output
```

without requiring knowledge of the runtime's internal state.

This is particularly important for conformance testing.

---

# 3.42 Representation Example

The following is an illustrative representation of a small program.

It is deliberately conceptual; the complete set of canonical operations and fields is established elsewhere in the specification.

```json
{
  "input": {
    "value": 10
  },

  "inputSchema": {
    "...": "queryfy schema"
  },

  "program": [
    {
      "op": "set",
      "target": {
        "$ref": "working.result"
      },
      "value": {
        "$ref": "input.value"
      }
    }
  ],

  "outputSchema": {
    "...": "queryfy schema"
  },

  "output": {
    "result": 10
  }
}
```

The important representation principles illustrated here are:

* input is ordinary JSON;
* schemas are distinct documents;
* the program is an instruction array;
* instructions are objects;
* references are explicit;
* output is ordinary JSON;
* the package distinguishes executable code from data.

---

# 3.43 Representation of a Compute Operation

A computation can be represented explicitly rather than embedded as an arbitrary expression.

Conceptually:

```json
{
  "op": "set",
  "target": {
    "$ref": "working.total"
  },
  "value": {
    "compute": {
      "op": "add",
      "left": {
        "$ref": "working.total"
      },
      "right": {
        "$ref": "current.price"
      }
    }
  }
}
```

The representation expresses three separate concepts:

1. obtain `working.total`;
2. obtain `current.price`;
3. calculate their sum;
4. assign the resulting value to `working.total`.

The computation itself does not perform the assignment.

The surrounding instruction performs the state mutation.

This distinction reflects the language model defined in Part 2.

---

# 3.44 Representation of Structured Control Flow

A structured control-flow instruction may contain nested instruction arrays.

Conceptually:

```json
{
  "op": "if",
  "condition": {
    "$ref": "working.ready"
  },
  "then": [
    {
      "op": "..."
    }
  ],
  "else": [
    {
      "op": "..."
    }
  ]
}
```

The nested arrays are ordinary Jaxson instruction sequences.

They do not introduce a separate syntax for instructions.

This allows the same instruction representation to be used at the top level and inside structured blocks.

---

# 3.45 Representation of Iteration

An iteration instruction may be represented as:

```json
{
  "op": "for",
  "in": {
    "$ref": "input.items"
  },
  "as": "item",
  "do": [
    {
      "op": "..."
    }
  ]
}
```

The representation identifies:

* the collection;
* the iteration variable;
* the body.

The iteration variable is a name within the iteration context, not a reference path itself.

The body may contain references to the current iteration value according to the reference semantics defined by the language.

---

# 3.46 Representation Principles

The Jaxson JSON representation follows several general principles.

### Explicitness

Executable constructs are explicitly marked.

### Structural meaning

A value's role is determined by its position within the program structure.

### No implicit expressions

Ordinary strings and values do not become executable expressions through convention.

### No implicit coercion

JSON types retain their meaning unless an operation explicitly defines conversion.

### Regularity

Instruction bodies, nested blocks and operands use consistent JSON structures.

### Extensibility

The representation can be extended without redefining the fundamental JSON model.

### Inspectability

A human familiar with Jaxson should be able to understand the broad structure of a program by inspecting its JSON.

---

# 3.47 Canonical Form

A Jaxson implementation MAY provide a canonical serialisation format.

Canonical serialisation may define:

* member ordering;
* whitespace;
* number representation;
* escaping;
* metadata ordering;
* normalisation of equivalent structures.

Canonicalisation MUST NOT change program semantics.

If canonicalisation is defined, it may be used to:

* hash programs;
* identify versions;
* compare programs;
* cache compiled representations;
* establish reproducible artefacts.

Canonicalisation is distinct from execution.

---

# 3.48 Program Identity

A Jaxson program MAY be identified by a digest of its canonical representation.

A complete execution contract MAY similarly be identified by a digest of:

```text
input
input schema
program
output schema
```

The expected output need not necessarily be included in the execution identity because it is a result rather than an input to execution.

Any identity mechanism MUST specify precisely which representations are included.

This permits Jaxson programs and execution contracts to be cached, versioned or referenced without depending upon filenames.

---

# 3.49 Separation Between Representation and Runtime

The representation describes **what is to be executed**.

The runtime determines **how execution is performed**.

For example, a program may contain:

```json
{
  "op": "compute"
}
```

without specifying whether the runtime implements computation using:

* an interpreter;
* a compiler;
* bytecode;
* an internal AST;
* native arithmetic;
* an arbitrary-precision numeric library.

Those are runtime implementation details.

The representation MUST remain independent of such implementation choices.

---

# 3.50 Separation Between Representation and Validation

The representation establishes the structure of a Jaxson program.

Validation determines whether that structure is acceptable.

These are related but distinct concerns.

A JSON document may be syntactically valid JSON while being invalid Jaxson.

Likewise, a structurally valid Jaxson program may receive an input which fails its input schema.

The validation sequence is therefore:

```text
JSON syntax
     │
     ▼
Jaxson structure
     │
     ▼
Input contract
     │
     ▼
Execution
     │
     ▼
Output contract
```

Each stage answers a different question.

---

# 3.51 Representation Requirements

A conforming Jaxson representation MUST:

1. represent a program as an ordered collection of instructions;
2. represent each instruction unambiguously;
3. distinguish executable constructs from ordinary JSON data;
4. distinguish references from literals;
5. distinguish computation from ordinary state mutation;
6. represent nested instruction blocks where required;
7. preserve instruction ordering;
8. preserve JSON `null` distinctly from absent members;
9. permit input and output documents to remain ordinary JSON;
10. permit queryfy schemas to be associated with input and output;
11. permit a program to be represented independently of a complete execution package; and
12. provide sufficient structure for a conforming implementation to validate a program before execution.

---

# 3.52 Non-Requirements

The representation does not require:

* a textual programming syntax separate from JSON;
* a compiler;
* a particular internal execution representation;
* a particular programming language implementation;
* a particular storage mechanism;
* a particular file naming convention;
* a particular JSON serialisation library.

A Jaxson implementation may provide any of these facilities, but they are not required by the representation itself.

---

# 3.53 Summary

The Jaxson representation makes the language executable while retaining JSON as its fundamental external form.

The representation distinguishes five logical execution components:

```text
input
input schema
program
output schema
output
```

The program itself is an ordered sequence of instruction objects.

Instructions explicitly distinguish:

* operations;
* operands;
* references;
* destinations;
* computation;
* control-flow blocks.

Ordinary JSON remains ordinary data unless its structural position gives it a defined Jaxson meaning.

The representation therefore follows a central principle:

> **Jaxson code is JSON, but not every JSON value is Jaxson code.**

The representation provides the concrete encoding required to express the abstract machine and language defined in Parts 1 and 2.

Part 4 defines how a Jaxson runtime loads, validates, prepares and executes this representation, including resource management, isolation, cancellation and operational behaviour.

Part 5 defines conformance requirements and the tests by which implementations can demonstrate compatibility with the specification.
