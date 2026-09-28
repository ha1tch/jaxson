# 5. Conformance

## 5.1 Overview

Conformance defines the requirements that an implementation MUST satisfy in order to claim compatibility with the Jaxson specification.

The purpose of conformance is to ensure that:

* a valid Jaxson program has consistent meaning;
* different implementations execute the same program equivalently;
* valid representations can be exchanged between implementations;
* invalid programs are rejected consistently;
* input and output contracts are enforced;
* runtime failures remain distinguishable from successful execution;
* deterministic programs remain deterministic;
* implementation-specific details do not become accidental language semantics.

Conformance applies to the complete Jaxson system:

```text
Part 1 — Model
Part 2 — Language
Part 3 — Representation
Part 4 — Runtime
Part 5 — Conformance
```

A conforming implementation MUST satisfy all mandatory requirements applicable to the features it claims to support.

Conformance is therefore behavioural rather than architectural.

A Jaxson implementation does not need to use a particular:

* programming language;
* parser;
* interpreter;
* compiler;
* virtual machine;
* data structure;
* storage system;
* execution engine.

It must, however, produce behaviour consistent with the semantics defined by the specification.

---

# 5.2 Conformance Principles

Jaxson conformance is based on five principles.

### 5.2.1 Semantic conformance

The implementation MUST assign the same meaning to Jaxson programs as defined by the language specification.

### 5.2.2 Representational conformance

The implementation MUST correctly read and produce representations defined by the representation specification.

### 5.2.3 Contract conformance

The implementation MUST enforce input and output schemas according to the execution contract.

### 5.2.4 Runtime conformance

The implementation MUST execute programs according to the runtime requirements.

### 5.2.5 Observable equivalence

Different conforming implementations SHOULD produce indistinguishable results for the same deterministic execution contract.

---

# 5.3 Scope of Conformance

Conformance applies at several levels.

A Jaxson implementation may conform to:

1. the JSON representation;
2. the language;
3. the execution model;
4. the runtime;
5. the complete specification.

These levels should not be confused.

For example, an implementation may be able to parse Jaxson JSON without being capable of executing it.

Likewise, an implementation may execute a subset of the language without being a complete conforming implementation.

A claim of full Jaxson conformance requires satisfying all mandatory requirements of the relevant specification version.

---

# 5.4 Conformance Terminology

The following terms have normative meaning.

### MUST

The requirement is mandatory.

An implementation which violates it is not conforming.

### MUST NOT

The prohibited behaviour is mandatory to avoid.

### SHOULD

The behaviour is strongly recommended but may be omitted when there is a documented reason.

### SHOULD NOT

The behaviour is discouraged but may be permitted under documented circumstances.

### MAY

The behaviour is optional.

Optional behaviour does not become part of core Jaxson semantics unless explicitly defined by the specification.

---

# 5.5 Specification Version

Conformance is always relative to a particular Jaxson specification version.

An implementation MUST identify the specification version against which it claims conformance.

For example:

```text
Jaxson specification: 1.0
Implementation: 1.0.3
```

The implementation version and specification version are separate concepts.

A runtime implementation may have many implementation releases while remaining conformant to one specification version.

---

# 5.6 Feature Support

An implementation MAY support only a subset of optional Jaxson facilities.

If so, it MUST clearly identify the supported feature set.

An implementation MUST NOT claim complete conformance while silently omitting a mandatory language feature.

Unsupported optional features SHOULD produce an explicit unsupported-feature result rather than being silently ignored.

---

# 5.7 Conformance Classes

A future Jaxson specification MAY define formal conformance classes.

A useful conceptual division is:

```text
Representation Conformance
        │
        ▼
Language Conformance
        │
        ▼
Runtime Conformance
        │
        ▼
Full Conformance
```

The initial specification should avoid creating unnecessary conformance classes unless they provide practical interoperability benefits.

The default interpretation of “Jaxson conforming implementation” should be full conformance.

---

# 5.8 Representation Conformance

A conforming implementation MUST correctly interpret valid Jaxson JSON representations.

This includes:

* valid JSON syntax;
* program structure;
* instruction structure;
* operands;
* references;
* blocks;
* computation structures;
* schemas;
* bundled execution contracts.

The implementation MUST reject malformed representations rather than attempting to reinterpret them according to implementation-specific rules.

---

# 5.9 JSON Conformance

The representation layer is JSON.

A conforming implementation MUST accept valid JSON wherever the Jaxson representation requires JSON.

It MUST reject syntactically invalid JSON.

The implementation MUST preserve the JSON value model:

```text
object
array
string
number
boolean
null
```

An implementation MUST NOT introduce host-language values into Jaxson documents as though they were standard JSON values.

---

# 5.10 Structural Conformance

The implementation MUST validate the structural requirements of Jaxson documents.

For example, where a field is required to contain:

* an instruction;
* an array of instructions;
* a reference;
* a computation;
* a schema;

the implementation MUST reject a structurally incompatible value.

Structural validation MUST occur before normal execution of the affected structure.

---

# 5.11 Instruction Conformance

Every supported Jaxson instruction MUST have the semantics defined by Part 2.

An implementation MUST NOT redefine an instruction according to a host-language convention.

For example, if a Jaxson instruction means:

```text
read value
→ compute result
→ assign result
```

an implementation cannot omit the computation or assignment merely because its internal representation differs.

---

# 5.12 Unknown Instructions

An implementation encountering an unknown mandatory instruction MUST reject the program.

It MUST NOT:

* silently ignore the instruction;
* treat it as a no-op;
* execute an approximate equivalent;
* continue execution as if it were absent.

An extension mechanism MAY define how unknown extension instructions are handled.

Such behaviour must be explicitly identifiable as extension behaviour.

---

# 5.13 Reference Conformance

Reference semantics are a central interoperability requirement.

A conforming implementation MUST resolve references according to the reference semantics defined by Part 2 and the associated queryfy semantics.

This includes:

* path interpretation;
* object member access;
* array access;
* nested access;
* missing values;
* reference context.

Two implementations MUST NOT interpret the same valid reference differently.

---

# 5.14 Missing Values

The distinction between missing and `null` MUST be preserved.

For example:

```text
missing
```

and:

```json
null
```

are not automatically equivalent.

A conforming implementation MUST follow the language's defined behaviour for missing references.

It MUST NOT substitute `null` merely because its host representation makes absence convenient.

Where Jaxson explicitly defines a missing reference as producing `null`, that behaviour is part of conformance.

Where Jaxson defines missingness as an error, the implementation MUST report an error.

---

# 5.15 Mutation Conformance

Mutation semantics MUST be preserved.

An implementation MUST:

* modify the intended state;
* preserve unrelated state;
* apply assignments to the correct destination;
* preserve mutation ordering;
* maintain the distinction between input and mutable working state where defined.

An implementation MUST NOT perform hidden mutations which affect later instructions.

---

# 5.16 State Conformance

A conforming implementation MUST preserve the abstract state-transition model.

For a program:

```text
I1
I2
I3
```

the implementation must behave as though it performs:

```text
S0
 ↓ I1
S1
 ↓ I2
S2
 ↓ I3
S3
```

The internal implementation may use another representation.

Observable behaviour must nevertheless be equivalent.

---

# 5.17 Sequencing Conformance

Instruction ordering is semantically significant unless the language explicitly defines otherwise.

A conforming implementation MUST execute instructions in the order defined by the language.

It MUST NOT reorder instructions merely because an alternative order appears computationally equivalent.

This is particularly important where instructions:

* mutate state;
* depend on previous results;
* change control flow;
* affect iteration;
* invoke computation.

---

# 5.18 Control-Flow Conformance

Control-flow semantics MUST be implemented exactly.

This includes:

* conditional execution;
* branching;
* nested blocks;
* termination;
* control targets;
* execution order.

A runtime may compile structured control flow into lower-level jumps, but the resulting behaviour must be equivalent to the abstract Jaxson execution model.

---

# 5.19 Iteration Conformance

Iteration is a semantic feature and therefore requires explicit conformance testing.

A conforming implementation MUST:

* identify the correct collection;
* establish the defined iteration order;
* expose the correct iteration value;
* maintain the correct iteration context;
* execute the body the correct number of times;
* handle nested iteration correctly;
* terminate correctly.

For arrays, iteration order MUST be deterministic.

---

# 5.20 Computation Conformance

Computation is confined to `compute` blocks.

A conforming implementation MUST enforce the distinction between:

```text
execution semantics
```

and:

```text
computation semantics
```

Arithmetic MUST NOT be silently introduced as an ordinary instruction capability outside the defined computation mechanism.

A runtime may optimise computation internally, but its observable results MUST remain equivalent to the specified computation semantics.

---

# 5.21 Numeric Conformance

Numeric semantics MUST be tested independently of the host implementation language.

Conformance tests SHOULD include:

* positive integers;
* negative integers;
* zero;
* large values;
* arithmetic boundaries;
* division;
* comparison;
* invalid operations;
* values near defined limits.

Where Jaxson requires architecture-independent numeric behaviour, the same program and input MUST produce the same result on different host architectures.

---

# 5.22 Comparison Conformance

Comparison operations MUST follow the defined type and value semantics.

Tests SHOULD cover:

* equal values;
* unequal values;
* numeric comparisons;
* string comparisons where supported;
* boolean comparisons where supported;
* null;
* missing values;
* incompatible types.

The implementation MUST NOT allow host-language coercion rules to silently become Jaxson semantics.

For example, a host language may consider:

```text
0 == false
```

while Jaxson may not.

The Jaxson specification determines the result.

---

# 5.23 Null Conformance

`null` is a JSON value and MUST be treated according to Jaxson semantics.

A conforming implementation MUST distinguish:

```text
null
```

from:

```text
missing
```

where the language makes that distinction observable.

Null handling MUST remain consistent across:

* references;
* assignments;
* comparisons;
* computation;
* output;
* schema validation.

---

# 5.24 Input Contract Conformance

A conforming implementation MUST enforce the declared input schema before normal execution.

A valid program with invalid input MUST NOT produce a successful result.

Conceptually:

```text
program valid
input invalid
     ↓
execution rejected
```

The runtime MUST NOT bypass input validation merely because execution would otherwise be possible.

---

# 5.25 Output Contract Conformance

A conforming implementation MUST validate the final output against the declared output schema.

A program that produces structurally valid JSON which fails its output schema MUST NOT be reported as a successful contract execution.

This establishes the distinction between:

```text
valid JSON
```

and:

```text
valid Jaxson output
```

The latter requires satisfying the declared output contract.

---

# 5.26 Contract Independence

Input and output schemas are independently significant.

A conforming implementation MUST NOT assume that:

```text
input schema = output schema
```

or that output necessarily has the same structure as input.

The entire purpose of the execution contract is to permit transformation between different document structures.

---

# 5.27 Program Independence

The program is independent of any particular input instance.

A conforming runtime SHOULD be able to execute the same program against multiple valid inputs.

Execution state MUST be isolated between invocations.

For example:

```text
Program P

P + Input A → Output A
P + Input B → Output B
```

must not result in state from A affecting B.

---

# 5.28 Determinism

Deterministic behaviour is a fundamental conformance property.

Where the language defines an operation deterministically, a conforming implementation MUST produce the same observable result for the same:

* program;
* input;
* schemas;
* defined execution environment.

This requirement applies across:

* repeated execution;
* separate processes;
* separate machines;
* separate runtime implementations.

---

# 5.29 Determinism Test

A basic determinism test is:

```text
execute(P, I) → R1
execute(P, I) → R2
```

The implementation conforms if:

```text
R1 = R2
```

for all deterministic programs within the defined resource envelope.

A stronger interoperability test is:

```text
implementation A:
execute(P, I) → R1

implementation B:
execute(P, I) → R2
```

and:

```text
R1 = R2
```

for the same defined environment.

---

# 5.30 Environmental Determinism

Determinism applies to the defined execution environment.

If a program is explicitly granted access to an external capability such as a clock, external service or random source, the result may depend upon that capability.

Such dependence does not constitute a violation of deterministic core semantics if the capability is explicitly part of the execution contract.

Implicit environmental dependence is not permitted.

---

# 5.31 Error Conformance

Errors are part of observable runtime behaviour.

A conforming implementation MUST reject invalid programs and invalid executions rather than silently producing arbitrary output.

Error conditions SHOULD be classified consistently according to the runtime model.

At minimum, implementations should distinguish:

* representation failure;
* program validation failure;
* input validation failure;
* execution failure;
* resource failure;
* cancellation;
* output validation failure.

---

# 5.32 Error Equivalence

Two implementations do not need to produce byte-for-byte identical diagnostic messages.

They MUST, however, agree on the semantic outcome.

For example:

```text
Implementation A:
"division by zero"

Implementation B:
"numeric operation failed: divisor = 0"
```

may both be conforming if both identify the same semantic failure.

The exact human-readable diagnostic is implementation-defined.

---

# 5.33 Error Location

Where the specification identifies an instruction or execution context as the source of failure, a conforming implementation SHOULD provide enough information to identify it.

This may include:

* instruction index;
* instruction identifier;
* operation;
* reference;
* iteration position;
* nested block.

Diagnostic richness is not itself part of core semantics, but it is important for practical interoperability and debugging.

---

# 5.34 Resource-Limit Conformance

A runtime may impose limits.

Such limits MUST be enforced consistently.

A runtime MUST NOT silently exceed a declared hard limit and then report successful execution.

Relevant limits may include:

* instruction count;
* execution time;
* memory;
* output size;
* nesting depth.

Resource limits are environmental constraints rather than language semantics.

---

# 5.35 Resource Failure

A program exceeding a runtime resource limit MUST terminate without being reported as successfully completed.

For example:

```text
program
  ↓
instruction limit exceeded
  ↓
resource failure
```

The runtime MUST NOT silently truncate execution and return the partially constructed state as normal output.

---

# 5.36 Cancellation Conformance

If execution is cancelled, the runtime MUST report cancellation rather than successful completion.

A partially generated output MUST NOT be returned as a successful result unless the language explicitly defines partial-result semantics.

The default Jaxson execution contract has no such partial-success semantics.

---

# 5.37 Isolation Conformance

Independent executions MUST have independent mutable state.

The following must be safe:

```text
P + A
P + B
P + C
```

executing concurrently.

The result of one execution MUST NOT alter another execution's:

* variables;
* working state;
* iteration state;
* output;
* control state.

Immutable program data MAY be shared.

---

# 5.38 Reentrancy Conformance

A runtime MUST be capable of executing a valid program more than once without retaining unintended execution state.

A useful conformance test is:

```text
execute(P, A)
execute(P, B)
execute(P, A)
```

The first and third executions must produce equivalent results.

This catches accidental retention of:

* counters;
* variables;
* iteration frames;
* output state;
* cached mutable values.

---

# 5.39 Parallel Conformance

If a runtime claims support for concurrent execution, concurrency MUST NOT alter program semantics.

For independent executions:

```text
serial:
A → B → C

parallel:
A ──────▶
B ──────▶
C ──────▶
```

the individual results must remain equivalent.

The runtime MUST NOT expose scheduling order as program state unless the language explicitly defines such behaviour.

---

# 5.40 Representation Round-Trip

A representation-capable implementation SHOULD support a round-trip property.

Conceptually:

```text
Jaxson representation
        ↓
      parse
        ↓
 internal representation
        ↓
    serialise
        ↓
Jaxson representation
```

The resulting representation need not be byte-for-byte identical.

It MUST remain semantically equivalent.

Differences such as:

* whitespace;
* member ordering where JSON semantics permit it;
* formatting;
* numeric presentation where semantically equivalent

must not change program meaning.

---

# 5.41 Canonical Representation

If Jaxson defines a canonical serialisation in Part 3, a conforming serializer MUST produce that canonical form.

If canonical serialisation is not required, implementations MAY choose their own formatting.

The semantic identity of a Jaxson program MUST NOT depend upon irrelevant JSON formatting.

---

# 5.42 Schema Conformance

Input and output schemas MUST be interpreted according to queryfy semantics.

A conforming implementation MUST NOT reinterpret a queryfy schema using an unrelated validation language.

Schema failures MUST prevent successful contract execution.

If queryfy itself has versioned semantics, the Jaxson implementation MUST identify the compatible queryfy version or semantic level where this affects interoperability.

---

# 5.43 Query Semantics

Reference and query semantics must remain consistent with the queryfy integration defined by the specification.

Tests SHOULD cover:

* direct properties;
* nested properties;
* arrays;
* indexed arrays;
* nested arrays and objects;
* missing paths;
* null values;
* invalid paths.

The purpose is to ensure that Jaxson does not acquire implementation-specific path semantics.

---

# 5.44 Extension Conformance

Jaxson implementations MAY provide extensions.

Extensions MUST be distinguishable from core Jaxson.

An extension MUST NOT redefine the semantics of an existing core instruction.

For example, an implementation MUST NOT make:

```text
set
```

mean something different merely because it provides an extension package.

Extensions should instead introduce separately identifiable facilities.

---

# 5.45 Extension Isolation

A program using no extensions MUST execute according to core Jaxson semantics.

Installing or enabling an extension MUST NOT change the meaning of unrelated core programs.

This permits runtimes to provide deployment-specific capabilities without fragmenting the language.

---

# 5.46 Capability Conformance

If a runtime exposes external capabilities, the availability of those capabilities MUST be explicit.

A program which requires an unavailable capability MUST fail explicitly.

The runtime MUST NOT silently substitute:

* fake data;
* null;
* empty results;
* host-local data

unless the relevant extension specification defines such behaviour.

---

# 5.47 Security Conformance

A conforming runtime MUST NOT grant core Jaxson programs arbitrary access to the host environment.

The core language must remain sandboxable.

In particular, merely executing a Jaxson program MUST NOT implicitly grant access to:

* files;
* processes;
* network resources;
* credentials;
* operating-system APIs.

Where external access is supported, it must be explicitly provided by the runtime or an extension.

---

# 5.48 Host Independence

A conforming implementation MUST NOT rely upon accidental properties of its host language.

For example, host-language behaviour relating to:

* type coercion;
* integer overflow;
* map ordering;
* floating-point representation;
* null handling;
* exceptions;
* references

must not silently become Jaxson semantics.

The Jaxson specification has priority over host-language defaults.

---

# 5.49 Architecture Independence

Where the Jaxson specification defines architecture-independent behaviour, the implementation MUST preserve that behaviour across architectures.

For example:

```text
x86-64
ARM64
RISC-V
```

must produce equivalent Jaxson results for the same deterministic execution.

This is particularly important for numeric operations.

---

# 5.50 Conformance Test Suite

A formal Jaxson distribution SHOULD include a conformance test suite.

The suite should consist of executable test cases representing language and runtime requirements.

Each test should define, as applicable:

```text
program
input
input schema
expected result
output schema
expected error
```

A useful test case can therefore be represented as:

```text
Test Case
├── input
├── inputSchema
├── program
├── outputSchema
└── expected
```

The test format should itself be deterministic and machine-readable where practical.

---

# 5.51 Positive Tests

Positive tests verify programs which MUST succeed.

Examples include:

* literal assignment;
* reference resolution;
* nested property access;
* array access;
* conditional execution;
* iteration;
* accumulation;
* computation;
* output construction;
* nested control flow.

A positive test passes when the resulting output is semantically equivalent to the expected output.

---

# 5.52 Negative Tests

Negative tests verify programs or execution contracts which MUST fail.

Examples include:

* invalid JSON;
* malformed Jaxson;
* unknown operation;
* malformed reference;
* invalid input;
* invalid computation;
* division by zero;
* invalid array access;
* output schema failure;
* resource exhaustion.

Negative tests are important because a runtime that produces the correct result for valid programs but accepts invalid programs is not fully conforming.

---

# 5.53 Boundary Tests

Conformance testing SHOULD include boundary conditions.

Examples include:

* empty object;
* empty array;
* empty string;
* null;
* first array element;
* last array element;
* single-element arrays;
* zero iterations;
* deeply nested structures;
* maximum supported numeric values;
* minimum supported numeric values;
* resource limits.

Boundary tests frequently reveal differences between implementations which ordinary examples do not expose.

---

# 5.54 State-Transition Tests

Tests SHOULD verify intermediate state transitions where those transitions affect later observable behaviour.

For example:

```text
set x = 1
set x = 2
output x
```

must produce:

```json
2
```

rather than:

```json
1
```

Such tests establish that instruction ordering and mutation semantics are correct.

---

# 5.55 Iteration Tests

Iteration tests SHOULD cover:

```text
empty collection
single element
multiple elements
nested iteration
mutation during iteration
accumulation
conditional execution inside iteration
```

For example, an accumulator test may establish that:

```text
total = 0

for each item:
    total = total + item.value
```

produces the same total regardless of runtime implementation.

---

# 5.56 Control-Flow Tests

Control-flow tests SHOULD verify:

* true conditions;
* false conditions;
* nested conditions;
* skipped blocks;
* repeated blocks;
* branching;
* termination;
* unreachable instructions.

A test should verify both the result and the absence of unintended execution.

---

# 5.57 Reference Tests

Reference tests SHOULD establish:

```text
object property
nested property
array element
nested array/object
missing property
null property
```

For example:

```text
input:
{
    "customer": {
        "name": "Ada"
    }
}
```

A reference to:

```text
customer.name
```

must resolve consistently across conforming implementations.

---

# 5.58 Computation Tests

Computation tests SHOULD be separated from control-flow tests where possible.

This makes it possible to identify whether a failure originates in:

```text
reference resolution
```

or:

```text
computation
```

or:

```text
state mutation
```

A useful test suite should therefore include both simple isolated computations and computations embedded inside larger programs.

---

# 5.59 Contract Tests

Tests SHOULD verify the complete execution contract:

```text
input
  ↓
input schema
  ↓
program
  ↓
output
  ↓
output schema
```

Examples should include:

```text
valid input → valid output
invalid input → rejected
valid input → invalid output → rejected
```

This ensures that an implementation does not accidentally treat schemas as documentation rather than executable contracts.

---

# 5.60 Cross-Implementation Testing

Where multiple Jaxson implementations exist, they SHOULD be tested against the same conformance suite.

For example:

```text
Reference Runtime
       │
       ├──── Test 1 → expected
       ├──── Test 2 → expected
       ├──── Test 3 → expected
       └──── Test N → expected
                 ▲
                 │
          Implementation B
```

This provides practical evidence that Jaxson semantics are implementation-independent.

---

# 5.61 Differential Testing

Implementations MAY be tested against one another.

Given:

```text
Program P
Input I
```

the results from two implementations can be compared:

```text
Runtime A → Result A
Runtime B → Result B
```

For deterministic programs:

```text
Result A ≡ Result B
```

where `≡` means semantic equivalence rather than necessarily identical diagnostic formatting or serialisation.

---

# 5.62 Property-Based Testing

A conformance suite MAY use generated programs and inputs.

Property-based testing is particularly useful for:

* reference resolution;
* arrays;
* nested structures;
* arithmetic;
* iteration;
* mutation;
* deterministic execution.

Generated cases MUST themselves obey the relevant Jaxson representation rules when testing valid-program properties.

Invalid-program generation may intentionally violate those rules when testing rejection behaviour.

---

# 5.63 Fuzz Testing

Jaxson runtimes SHOULD be fuzz-tested at the representation boundary.

Useful fuzz targets include:

* JSON parsing;
* instruction parsing;
* reference parsing;
* computation parsing;
* nested blocks;
* deeply nested documents;
* malformed schemas.

A fuzzing failure which causes:

* arbitrary code execution;
* host crash;
* memory corruption;
* uncontrolled resource consumption;
* cross-execution state leakage

should be treated as a runtime defect.

---

# 5.64 Resource-Conformance Testing

A runtime should be tested against programs designed to exceed each configured resource limit.

For example:

```text
very large iteration
deep nesting
large output
large computation
non-terminating control flow
```

The expected result should be a controlled resource failure rather than:

* process crash;
* silent truncation;
* host exhaustion;
* successful partial result.

---

# 5.65 Security Testing

A runtime SHOULD be tested with deliberately hostile programs and representations.

Tests should verify that a Jaxson program cannot escape its intended execution environment.

Examples include attempts to:

* access arbitrary files;
* execute host processes;
* access environment variables;
* consume unlimited memory;
* execute indefinitely;
* interfere with another execution.

Security testing belongs primarily to the runtime rather than the abstract language.

---

# 5.66 Conformance and Optimisation

An optimised runtime is conforming only if its optimisations preserve observable semantics.

For every optimisation:

```text
unoptimised(P, I)
```

and:

```text
optimised(P, I)
```

must produce equivalent results.

This applies to:

* constant folding;
* dead-code elimination;
* instruction fusion;
* reference caching;
* computation compilation;
* control-flow compilation.

An optimisation which changes a program's error behaviour is not valid unless that error behaviour is explicitly outside the observable contract.

---

# 5.67 Conformance and Caching

Caching requires particular care.

A runtime MAY cache:

* parsed programs;
* validated programs;
* compiled instructions;
* immutable computation structures.

It MUST NOT incorrectly cache:

* mutable execution state;
* input-dependent values;
* output state;
* iteration state.

A cached execution artefact must remain independent of the state of previous executions.

---

# 5.68 Conformance and Serialization

If an implementation serialises a Jaxson program, it MUST preserve its semantics.

The following are generally not semantically significant unless explicitly specified:

* whitespace;
* indentation;
* line endings;
* object member ordering;
* formatting.

The following may be semantically significant and therefore require preservation:

* array ordering;
* instruction ordering;
* numeric values;
* strings;
* references;
* control targets;
* computation structure.

---

# 5.69 Conformance and Diagnostics

Diagnostics are important but are not generally part of program output.

Two conforming implementations MAY report different textual errors.

They SHOULD nevertheless expose equivalent semantic information.

For example:

```text
Runtime A:
"input schema validation failed"

Runtime B:
"INPUT_ERROR: schema mismatch"
```

may both be conforming.

A runtime MUST NOT convert an error into successful output merely because its diagnostic subsystem is unable to represent the failure.

---

# 5.70 Conformance and Logging

Logging is implementation-defined.

Logs MUST NOT be required for program correctness.

A program must behave equivalently whether diagnostic logging is:

* enabled;
* disabled;
* redirected;
* sampled.

Logging SHOULD therefore be treated as an observational facility outside the language semantics.

---

# 5.71 Conformance and Tracing

Instruction tracing is similarly optional.

Enabling tracing MUST NOT alter:

* instruction ordering;
* computation;
* state;
* output;
* error semantics.

If enabling tracing changes the result, the implementation is not conforming.

---

# 5.72 Conformance and Debugging

Debugging facilities may expose internal state.

Such facilities MUST NOT accidentally create additional program-visible state.

For example, inspecting a variable in a debugger must not modify it.

A debugger may pause execution, but resuming execution must produce the same semantic result as uninterrupted execution, subject to externally imposed cancellation or resource limits.

---

# 5.73 Conformance and External Capabilities

Programs using external capabilities require an explicitly defined environment.

For example:

```text
Program
+
Input
+
Clock capability
```

is a different execution environment from:

```text
Program
+
Input
```

A conformance test involving external capabilities MUST define the capability's behaviour.

Otherwise, the test cannot establish deterministic equivalence.

---

# 5.74 Conformance and Versioning

A newer runtime MAY support older Jaxson specifications.

It MUST preserve the semantics of the older version when executing a program under that version.

A runtime MUST NOT silently reinterpret an old program according to incompatible newer semantics.

Where semantics have intentionally changed, the implementation must use explicit versioning or compatibility rules.

---

# 5.75 Backward Compatibility

Backward compatibility should preserve the meaning of existing valid programs.

A new Jaxson version SHOULD therefore avoid changing the semantics of existing instructions unless there is a compelling specification-level reason.

If a semantic change is necessary, it should be associated with an explicit specification version.

---

# 5.76 Forward Compatibility

A runtime is not required to execute programs containing features it does not understand.

However, it SHOULD fail cleanly.

For example:

```text
Jaxson 2.0 program
        ↓
Jaxson 1.0 runtime
        ↓
unsupported specification/version
```

is preferable to attempting partial execution.

Partial interpretation can produce outputs which appear valid while being semantically incorrect.

---

# 5.77 Conformance Manifest

A runtime implementation SHOULD be able to identify its conformance characteristics.

A conceptual manifest may contain:

```json
{
  "implementation": "example-runtime",
  "implementationVersion": "1.2.0",
  "jaxsonVersion": "1.0",
  "queryfyVersion": "1.x",
  "features": [],
  "extensions": []
}
```

The exact manifest format is implementation-defined unless formally standardised.

Its purpose is to make interoperability characteristics explicit.

---

# 5.78 Reference Implementation

The Jaxson project SHOULD provide a reference implementation.

The reference implementation serves several purposes:

* executable interpretation of the specification;
* demonstration of semantics;
* development tool;
* conformance oracle;
* test-suite host;
* interoperability reference.

The reference implementation is not inherently authoritative merely because it is first.

If the implementation and specification disagree, the specification defines the required behaviour.

The reference implementation should therefore itself be tested against the conformance suite.

---

# 5.79 Conformance Suite as Executable Specification

The conformance suite should eventually become an executable complement to the written specification.

The relationship is:

```text
written specification
        │
        ▼
semantic requirements
        │
        ▼
conformance tests
        │
        ▼
runtime implementations
```

This makes the specification less dependent upon prose alone.

A good conformance suite should make ambiguous or underspecified behaviour visible.

Where a test cannot be written unambiguously, the corresponding language rule may require clarification.

---

# 5.80 Test Case Independence

Each conformance test SHOULD be independently executable.

A test MUST NOT rely upon mutable state left behind by another test.

For example:

```text
Test A
Test B
Test C
```

must produce the same individual results whether executed:

```text
A → B → C
```

or:

```text
C → A → B
```

This prevents accidental dependence upon runtime state.

---

# 5.81 Test Determinism

Conformance tests themselves MUST be deterministic unless they explicitly test nondeterministic or external capabilities.

The same test executed repeatedly should produce the same pass/fail result.

Tests should therefore avoid unspecified dependencies on:

* current time;
* host filesystem;
* host environment;
* network availability;
* execution scheduling.

---

# 5.82 Minimal Conformance Set

A minimal initial conformance suite should cover at least:

### Representation

* valid package;
* valid instruction;
* malformed instruction;
* malformed reference;
* malformed computation.

### References

* object access;
* nested access;
* array indexing;
* missing path;
* null.

### State

* assignment;
* mutation;
* multiple sequential mutations.

### Control

* conditional;
* branch;
* nested block;
* termination.

### Iteration

* empty collection;
* single element;
* multiple elements;
* nested iteration.

### Computation

* basic arithmetic;
* comparison;
* invalid computation;
* numeric boundary conditions.

### Contracts

* valid input;
* invalid input;
* valid output;
* invalid output.

### Runtime

* instruction limit;
* nesting limit;
* cancellation;
* isolated concurrent executions.

---

# 5.83 Full Conformance Matrix

A mature Jaxson project SHOULD maintain a conformance matrix mapping specification requirements to tests.

Conceptually:

```text
Requirement
    │
    ├── specification section
    ├── test identifier
    ├── expected behaviour
    └── implementation result
```

For example:

```text
JAX-REF-001
  Part 2: Reference Resolution
  Test: reference.object.nested
  Expected: nested value resolved
```

This makes it possible to determine whether every normative requirement has executable coverage.

---

# 5.84 Conformance Failures

A conformance failure occurs when an implementation violates a mandatory requirement.

Examples include:

* accepting malformed Jaxson as valid;
* executing an unknown mandatory operation;
* resolving a reference incorrectly;
* changing instruction order;
* producing architecture-dependent results where semantics are architecture-independent;
* accepting invalid input;
* returning output which violates the output contract;
* silently swallowing execution errors;
* allowing execution state to leak between invocations.

A conformance failure should be reported against the relevant specification requirement.

---

# 5.85 Partial Conformance

An implementation MAY describe itself as supporting a subset of Jaxson during development.

It SHOULD use explicit terminology such as:

```text
experimental
partial
subset
development
```

rather than simply claiming conformance.

The supported subset SHOULD be documented.

For example:

```text
Supported:
references
assignment
conditional execution
compute

Not yet supported:
iteration
branching
extensions
```

This avoids ambiguity for users and other implementations.

---

# 5.86 Conformance Claims

A full conformance claim SHOULD identify:

* Jaxson specification version;
* implementation name;
* implementation version;
* supported mandatory features;
* supported optional features;
* supported extensions;
* applicable runtime limits;
* queryfy compatibility.

A claim should be reproducible through the published conformance suite.

---

# 5.87 Interoperability

The ultimate purpose of conformance is interoperability.

A Jaxson program should be transferable between conforming implementations without requiring its author to rewrite the program for each runtime.

Conceptually:

```text
             Jaxson Program
                    │
          ┌─────────┼─────────┐
          ▼         ▼         ▼
       Runtime A Runtime B Runtime C
          │         │         │
          ▼         ▼         ▼
       Output A  Output B  Output C
```

For deterministic programs:

```text
Output A ≡ Output B ≡ Output C
```

subject to explicitly defined differences such as diagnostic representation.

---

# 5.88 Semantic Equivalence

Conformance compares semantics, not implementation details.

Two outputs are semantically equivalent when they represent the same JSON value according to the Jaxson and JSON data models.

Two error results are semantically equivalent when they represent the same defined failure condition, even if their human-readable diagnostics differ.

Two execution traces need not be identical unless tracing itself is standardised.

---

# 5.89 Implementation Freedom

Conformance deliberately leaves substantial implementation freedom.

A runtime may choose:

```text
tree interpreter
bytecode VM
compiled machine code
JIT
AOT compiler
stack machine
register machine
```

provided that all approaches produce equivalent observable semantics.

Likewise, the implementation may represent state internally using:

```text
maps
structs
arrays
persistent trees
mutable trees
specialised values
```

provided that those choices do not alter Jaxson behaviour.

---

# 5.90 Specification Authority

When an implementation detail conflicts with the defined Jaxson semantics, the semantics take precedence.

The following hierarchy applies conceptually:

```text
Jaxson specification
        ↓
language semantics
        ↓
representation semantics
        ↓
runtime behaviour
        ↓
implementation details
```

An implementation detail cannot redefine the language.

---

# 5.91 Conformance and Undefined Behaviour

The specification SHOULD minimise undefined behaviour.

Where behaviour is not defined, implementations SHOULD NOT independently invent incompatible semantics and treat them as portable Jaxson.

If an unspecified area becomes important for interoperability, it should be promoted into an explicit specification rule.

This is particularly important for:

* numeric edge cases;
* missing values;
* iteration;
* evaluation order;
* error handling;
* resource limits.

---

# 5.92 Conformance and Implementation-Defined Behaviour

Some behaviour may legitimately remain implementation-defined.

Examples include:

* diagnostic wording;
* internal execution architecture;
* tracing format;
* logging format;
* maximum resource limits;
* optional debugging facilities.

Implementation-defined behaviour should be documented where it can affect users or interoperability.

---

# 5.93 Conformance and Optional Features

Optional features must remain optional.

An implementation which does not support an optional feature may still be conforming if the specification permits its omission.

However, a program requiring that feature cannot be executed successfully by such an implementation.

The runtime should therefore provide an explicit compatibility result.

---

# 5.94 Conformance and Future Extensions

The conformance model should permit Jaxson to evolve without compromising existing implementations.

New features should preferably be introduced through:

* new specification versions;
* explicit extensions;
* additional operations;
* explicit capability declarations.

Existing semantics should remain stable wherever possible.

---

# 5.95 Conformance Philosophy

Jaxson is intended to be small enough that its semantics can be tested exhaustively at the language boundary.

The conformance model therefore favours:

* explicit semantics;
* deterministic execution;
* machine-readable tests;
* small primitives;
* clear error boundaries;
* implementation independence;
* explicit extensions.

The goal is not to certify a particular architecture.

The goal is to establish that:

> **a Jaxson program means the same thing wherever a conforming runtime executes it.**

---

# 5.96 Complete Conformance Model

The five parts of the Jaxson specification can now be viewed as one system:

```text
┌─────────────────────────────────────────────────────┐
│                     JAXSON                          │
├─────────────────────────────────────────────────────┤
│                                                     │
│  1. MODEL                                           │
│     Defines what Jaxson is                          │
│                                                     │
│  2. LANGUAGE                                        │
│     Defines what programs mean                      │
│                                                     │
│  3. REPRESENTATION                                  │
│     Defines how programs are encoded                │
│                                                     │
│  4. RUNTIME                                         │
│     Defines how programs are executed               │
│                                                     │
│  5. CONFORMANCE                                     │
│     Defines how implementations prove equivalence   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

The execution contract is:

```text
        INPUT
          │
          ▼
   INPUT SCHEMA
          │
          ▼
       PROGRAM
          │
          ▼
       RUNTIME
          │
          ▼
        STATE
          │
          ▼
      EXECUTION
          │
          ▼
        OUTPUT
          │
          ▼
  OUTPUT SCHEMA
          │
          ▼
        RESULT
```

The conformance contract is:

```text
same program
+
same input
+
same schemas
+
same defined environment
        │
        ▼
conforming runtime
        │
        ▼
same observable semantics
```

---

# 5.97 Final Conformance Requirement

A Jaxson implementation conforms to this specification when, for every mandatory language and runtime requirement applicable to its claimed specification version, its observable behaviour is equivalent to the behaviour defined by the specification.

Conformance does not require identical source code, architecture, internal representation, performance characteristics, diagnostics or implementation language.

It requires semantic equivalence.

The essential test is therefore:

```text
Can another conforming implementation execute
the same Jaxson contract and obtain the same
defined result?
```

If the answer is yes, Jaxson has achieved its principal interoperability objective.

---

# 5.98 Completion of the Specification

With the completion of Part 5, the Jaxson specification consists of five complementary documents:

1. **Model** — defines the conceptual machine and execution contract.
2. **Language** — defines instructions, state, references, computation, control flow and iteration.
3. **Representation** — defines how those concepts are encoded as JSON.
4. **Runtime** — defines how an implementation loads, validates, executes and contains a Jaxson program.
5. **Conformance** — defines the requirements and tests by which implementations can demonstrate semantic compatibility.

Together they define Jaxson as a complete language rather than merely a JSON format.

The resulting architecture can be summarised as:

```text
JSON
 │
 ├── input
 ├── input schema
 ├── program
 └── output schema
        │
        ▼
     JAXSON
        │
        ├── declarative instruction sequence
        ├── mutable execution state
        ├── references
        ├── control flow
        ├── iteration
        └── compute blocks
        │
        ▼
     RUNTIME
        │
        ├── validation
        ├── execution
        ├── resource control
        ├── isolation
        └── diagnostics
        │
        ▼
      OUTPUT
        │
        ▼
   QUERYFY VALIDATION
        │
        ▼
      RESULT
```

The central property of the system is that the program remains a declarative description of an executable state machine, while the surrounding schemas establish the contract within which that machine operates.

That separation permits Jaxson to remain small, portable and implementation-independent while still being capable of expressing stateful transformations, control flow and iteration over JSON data.
