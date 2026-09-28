# 2. Language

## 2.1 Overview

The Jaxson language defines a small set of instructions for manipulating execution state, controlling execution flow, accessing JSON values, iterating over collections, and performing computation.

A Jaxson program is an ordered sequence of instructions.

The language is intentionally divided into two domains:

1. **execution**, which controls state, sequencing, branching, iteration and mutation; and
2. **computation**, which produces values within explicit `compute` blocks.

This distinction is fundamental to Jaxson.

Jaxson is not an expression-oriented language in which arbitrary expressions may appear wherever a value is required. Values are obtained from state, supplied as literals, or produced by computation. Instructions determine how those values are used.

The language therefore has the following conceptual structure:

```text
                 Jaxson Program
                       │
              ┌────────┴────────┐
              │                 │
          Execution         Computation
              │                 │
      ┌───────┼────────┐        │
      │       │        │        │
    State   Control  Iteration  │
      │       │        │        │
      └───────┴────────┘        │
              │                 │
              └───────┬─────────┘
                      │
                      ▼
                    Output
```

The language does not prescribe how these concepts are represented in JSON. Their JSON representation is defined in Part 3.

---

# 2.2 Instruction Model

An instruction is an atomic unit of Jaxson execution.

An instruction has an operation and, where required, operands or parameters.

Conceptually:

```text
instruction
    │
    ├── operation
    ├── operands
    └── optional parameters
```

The operation determines the semantic effect of the instruction.

An instruction MAY:

* read execution state;
* read one or more values;
* produce a value;
* modify execution state;
* alter control flow;
* invoke a computation;
* establish an iteration context;
* terminate execution.

An instruction MUST have deterministic semantics.

The meaning of an instruction MUST NOT depend upon the implementation language used by the runtime.

---

# 2.3 Instruction Categories

Jaxson instructions are divided into functional categories.

### 2.3.1 Data instructions

Data instructions obtain, assign, copy, create or remove values.

They operate on JSON values and execution state.

### 2.3.2 Control instructions

Control instructions determine which instructions execute and in what order.

They provide conditional execution, branching and termination.

### 2.3.3 Iteration instructions

Iteration instructions execute a body repeatedly against the members of a collection.

### 2.3.4 Computation instructions

Computation instructions invoke a `compute` block to produce a value.

Arithmetic belongs to this category.

### 2.3.5 Output instructions

Output instructions establish or modify the document that will ultimately be returned by the execution.

These categories describe semantics rather than requiring a particular opcode naming scheme.

---

# 2.4 Values

A Jaxson value is a JSON value:

* object;
* array;
* string;
* number;
* boolean; or
* null.

Values may originate from several sources.

### Literal values

A literal value is supplied directly by the program.

### Referenced values

A referenced value is obtained from the current execution state or another defined data context.

### Computed values

A computed value is produced by a `compute` block.

### Instruction results

Some instructions may produce values as part of their normal operation.

Jaxson does not require all instructions to produce values.

An instruction which changes state without producing a value is nevertheless a complete instruction.

---

# 2.5 Operands

An operand identifies a value used by an instruction.

An operand may conceptually be:

```text
literal
reference
computed value
```

The exact representation is defined in Part 3.

The language MUST distinguish between a literal value and a reference to a value.

For example, an instruction receiving:

```text
5
```

does not implicitly mean:

> obtain the value located at path `5`.

Likewise, a string does not implicitly become a path merely because it resembles one.

References must have an explicit representation.

This prevents accidental interpretation of ordinary JSON values as executable expressions.

---

# 2.6 References

A reference identifies a location from which a value can be obtained.

References may address:

* input data;
* working state;
* variables;
* objects;
* arrays;
* nested JSON structures;
* iteration context; or
* other locations explicitly defined by the language.

Jaxson uses queryfy's data-querying model where appropriate rather than defining an unrelated path language.

A reference is an access operation, not a computation.

For example:

```text
reference → state.a
```

means:

> obtain the value currently located at `state.a`.

It does not mean:

> evaluate an expression involving `state.a`.

This distinction is maintained throughout the language.

---

# 2.7 Reference Resolution

Reference resolution occurs when an instruction requiring a value is executed.

The value obtained by a reference is the value present at the referenced location at that point in execution.

Consequently, references are evaluated against the **current state**, not against the state which existed when the program was initially loaded.

For example:

```text
SET a = 10
SET b = reference(a)
SET a = 20
```

The second instruction obtains `10`.

The later modification of `a` does not retroactively alter the value already assigned to `b`.

A reference therefore resolves to a value when the instruction using it executes.

---

# 2.8 Missing References

A reference may identify a location which does not exist.

The language MUST define missing-reference behaviour explicitly.

Jaxson distinguishes between:

* a location containing JSON `null`; and
* a location which does not exist.

These are not inherently equivalent.

Where an instruction requires an existing value, attempting to use a missing reference MAY constitute an execution error.

Where an instruction explicitly permits absent values, the instruction MAY define an alternative behaviour.

The behaviour is instruction-specific and MUST NOT be inferred from JSON `null`.

This distinction is important because `null` is a legitimate JSON value.

---

# 2.9 State

Jaxson execution operates on mutable state.

State is conceptually divided into:

```text
Input
Working State
Output State
Execution Context
Control State
```

The exact representation of these components is defined by the runtime and language specifications.

The language does not require that these components appear as literal top-level JSON properties.

The important semantic property is that instructions can observe changes made by previous instructions.

---

# 2.10 Assignment

Assignment establishes a value at a destination in mutable state.

Conceptually:

```text
SET destination = value
```

The value is resolved before the destination is modified.

For example:

```text
SET b = a
```

means:

1. resolve `a`;
2. obtain its current value;
3. assign that value to `b`.

It does not create a persistent alias between `a` and `b`.

Subsequent changes to `a` therefore do not implicitly change `b`.

Unless otherwise specified, assignment copies the JSON value into the destination according to the value semantics defined by the runtime.

---

# 2.11 State Mutation

Jaxson provides explicit state mutation operations.

Mutation MAY include:

* setting a value;
* replacing a value;
* deleting a value;
* appending to an array;
* inserting into an array;
* creating an object member;
* replacing an array member.

Mutation instructions operate on the current state.

The language MUST define the behaviour of invalid destinations, including:

* nonexistent parent paths;
* incompatible destination types;
* invalid array indexes;
* attempts to modify immutable state.

A runtime MUST NOT silently invent incompatible structure unless that behaviour is explicitly defined by the relevant instruction.

---

# 2.12 Input State

The externally supplied input is available to the program through a defined input context.

The input is logically distinct from mutable working state.

Unless a language instruction explicitly permits mutation of input, instructions MUST NOT modify the original input document.

This provides a stable source from which a program may repeatedly obtain values.

If a program requires mutable copies of input values, it MUST explicitly place those values into mutable state.

This distinction permits a program to retain the original input while progressively transforming derived state.

---

# 2.13 Working State

Working state contains values created or modified during execution.

It is the principal mutable domain of Jaxson.

Working state may contain:

* intermediate values;
* counters;
* accumulators;
* temporary values;
* constructed objects;
* constructed arrays;
* iteration results;
* computation results.

Working state exists to allow a Jaxson program to perform multi-step transformations without requiring every operation to be expressed as a single computation.

---

# 2.14 Output State

The output state is the state from which the final output document is established.

A Jaxson implementation MAY represent output as an explicit mutable object throughout execution or establish it as a final projection of working state.

The language semantics MUST nevertheless make the output location unambiguous.

A program MUST NOT rely on implementation-specific details of how output is internally represented.

---

# 2.15 Sequencing

Instructions execute in program order unless control flow changes the execution position.

If instructions `A`, `B` and `C` occur sequentially:

```text
A
B
C
```

then, absent a control-flow operation:

```text
A → B → C
```

A state mutation performed by `A` is therefore visible to `B`, and a mutation performed by `B` is visible to `C`.

Instruction ordering is consequently semantically significant.

A conforming implementation MUST preserve the observable ordering required by the language.

---

# 2.16 Conditional Execution

Jaxson provides conditional execution.

A conditional evaluates a condition and executes one of two possible instruction sequences according to the result.

Conceptually:

```text
IF condition
    THEN instructions
    ELSE instructions
```

The condition MUST resolve to a boolean value.

Jaxson does not implicitly define general truthiness rules unless explicitly specified by the language.

In particular, values such as:

```text
0
""
null
[]
{}
```

MUST NOT automatically be treated as boolean values merely because they occur in a conditional context.

A condition either produces a boolean or is invalid for the conditional operation.

---

# 2.17 Branching

Jaxson control flow MAY permit execution to branch to another instruction or block.

A branch changes the next instruction executed by the machine.

Branching does not itself modify ordinary data state.

This distinction allows a program to alter execution order without introducing hidden data mutation.

If Jaxson exposes labels or equivalent control-flow targets, those targets are part of the program's control structure and MUST resolve unambiguously.

---

# 2.18 Blocks

A block is an ordered sequence of Jaxson instructions treated as a single structural unit.

Blocks may be used for:

* conditional branches;
* iteration bodies;
* computation;
* other structured control-flow constructs.

A block has no inherent independent data state unless the language explicitly defines a scope for it.

Consequently, entering a block does not automatically create a new variable environment.

Where scope is required, the relevant language construct MUST define it explicitly.

---

# 2.19 Iteration

Jaxson supports iteration over JSON collections.

The primary iteration target is an array.

An iteration executes a body once for each element of the collection.

Conceptually:

```text
FOR element IN collection
    body
```

For an array:

```text
[
    A,
    B,
    C
]
```

the body executes three times:

```text
element = A
element = B
element = C
```

Iteration order MUST be deterministic.

For arrays, this means ascending index order unless the language explicitly provides another mode.

---

# 2.20 Iteration Context

Each iteration establishes an iteration context.

The context provides access to the current element and, where defined, associated information such as:

* index;
* collection;
* iteration count.

Iteration context is temporary.

A value exposed as the current element MUST refer to the element associated with the current iteration and MUST change as execution advances to the next iteration.

Nested iteration creates nested iteration contexts.

The language MUST define which context is visible when an inner iteration completes.

---

# 2.21 Nested Iteration

An iteration body MAY itself contain an iteration.

For example:

```text
FOR row IN rows
    FOR item IN row.items
        ...
```

Each nested iteration has its own current element.

The outer context remains available unless shadowed by an explicitly defined scope rule.

The language MUST define how iteration variables are resolved when names overlap.

Nested iteration MUST preserve deterministic ordering at every level.

---

# 2.22 Mutation During Iteration

A program MAY modify state while iterating.

However, modifying the collection currently being iterated introduces potentially ambiguous behaviour.

Jaxson therefore MUST define whether the iteration operates over:

1. the live collection; or
2. a stable iteration view established when iteration begins.

A stable iteration view is generally preferable for deterministic semantics because modifications made by the loop body cannot silently alter which elements will subsequently be visited.

If the language permits live-collection semantics, those semantics must be explicitly defined.

A conforming implementation MUST NOT leave this behaviour implementation-dependent.

---

# 2.23 Computation

Computation occurs exclusively within `compute` blocks.

A compute block receives one or more values and produces a result according to the computational operations supported by Jaxson.

Conceptually:

```text
compute
    operands
        │
        ▼
    calculation
        │
        ▼
     result
```

The result may subsequently be assigned to state or supplied to another instruction.

Computation does not implicitly modify unrelated state.

For example:

```text
compute a + b
```

produces a value.

It does not by itself modify either `a` or `b`.

---

# 2.24 Arithmetic

Arithmetic operations are not general Jaxson instructions.

Arithmetic MUST occur within a `compute` block.

This restriction is intentional.

It prevents arithmetic syntax from spreading throughout the instruction language and establishes a clear boundary between:

```text
what the machine does
```

and:

```text
what value a calculation produces
```

Arithmetic operations include, subject to the supported computational model:

* addition;
* subtraction;
* multiplication;
* division;
* remainder;
* comparison;
* and other explicitly defined numeric operations.

The exact set of supported operations is defined as part of the computation semantics.

---

# 2.25 Computation Purity

A compute block is conceptually value-producing rather than state-mutating.

A compute block:

1. obtains its operands;
2. evaluates its computation;
3. produces a result.

It does not implicitly assign that result to arbitrary state.

State mutation occurs through the surrounding Jaxson instruction model.

This establishes:

```text
compute → value
instruction → state transition
```

rather than:

```text
expression → arbitrary side effects
```

A future extension MAY define explicitly stateful computational operations, but such operations must not be introduced implicitly through ordinary arithmetic.

---

# 2.26 Computation Errors

A compute block MAY fail.

Examples include:

* invalid operand types;
* division by zero;
* numeric overflow where applicable;
* unsupported numeric operations;
* invalid numeric representations.

A computation failure is an execution error.

The runtime MUST NOT silently convert an invalid computation into an unrelated JSON value unless the relevant operation explicitly defines such behaviour.

---

# 2.27 Comparison

Comparison is a computational operation.

Comparisons produce boolean values.

The language MUST define comparison semantics separately for:

* numbers;
* strings;
* booleans;
* null;
* arrays; and
* objects.

Jaxson MUST NOT assume that arbitrary JSON values can be meaningfully ordered merely because they can be represented as JSON.

Where an operation requires ordering, the permitted operand types must be explicit.

Equality and ordering are distinct concepts.

Two values may be equal without one being ordered before the other.

---

# 2.28 Type Requirements

Jaxson is JSON-oriented but does not require every instruction to accept every JSON value.

Each instruction defines its permitted operand types.

For example:

```text
array append → array destination
numeric addition → numeric operands
conditional → boolean condition
```

When an instruction receives an incompatible value, the operation fails according to its defined error semantics.

Jaxson does not require implicit coercion between unrelated JSON types unless an operation explicitly defines such coercion.

In particular, a numeric string SHOULD NOT automatically become a number merely because it appears in a numeric computation.

Explicit conversion, where supported, is preferable to implicit conversion.

---

# 2.29 Null

`null` is a legitimate JSON value.

It is distinct from:

* a missing value;
* an undefined reference;
* an execution error.

Instructions MUST define how they handle `null`.

A reference resolving to a location containing `null` has successfully resolved a value.

A reference resolving to no location has not.

This distinction is particularly important for conditional logic, assignment and output construction.

---

# 2.30 Deletion

Deletion removes a value from a mutable JSON structure.

Deletion is distinct from assigning `null`.

For example:

```text
delete object.field
```

results in the member no longer existing.

Whereas:

```text
set object.field = null
```

leaves the member present with a `null` value.

Jaxson MUST preserve this distinction.

---

# 2.31 Object Mutation

Objects may be modified through explicit state mutation instructions.

An object member may be:

* created;
* replaced;
* read;
* deleted.

Object member names are strings according to JSON semantics.

The language MUST define the behaviour of assigning a member which already exists.

Unless otherwise specified, assignment replaces the previous value.

---

# 2.32 Array Mutation

Arrays may be modified through explicit mutation instructions.

Operations may include:

* append;
* insert;
* replace;
* delete.

Array indexes are zero-based unless explicitly specified otherwise.

An invalid array index is an execution error unless the operation explicitly defines another behaviour.

Deleting an array element MUST have defined semantics regarding subsequent indexes. Jaxson MUST NOT leave this behaviour to the implementation.

---

# 2.33 Temporary Values

A program may require intermediate values which are not part of its final output.

Such values may be held in working state.

Temporary state is ordinary mutable state unless the language explicitly defines a separate temporary scope.

A runtime MUST NOT assume that a value is disposable merely because the program does not ultimately reference it.

Resource-management rules are defined in Part 4.

---

# 2.34 Scope

Jaxson may establish local execution contexts for blocks and iterations.

Scope controls visibility of names or references.

The language MUST define whether a value created inside a block remains accessible after the block terminates.

The initial language should favour explicit semantics over implicit lexical behaviour.

In particular, entering a block MUST NOT automatically imply a new scope unless the block construct explicitly defines one.

This avoids introducing conventional programming-language semantics where they are not necessary.

---

# 2.35 Program Counter

The executing Jaxson machine maintains a logical position within the instruction sequence.

This position identifies the next instruction to execute.

The program counter is part of execution state but is not ordinary JSON data.

Sequential execution advances the program counter.

Control-flow operations may change it.

A Jaxson implementation may use any internal representation for the program counter provided that its externally observable behaviour conforms to the language semantics.

The JSON representation of a program therefore does not need to expose a numeric program counter merely because the abstract machine has one.

---

# 2.36 Labels and Targets

If a Jaxson implementation exposes explicit branching targets, each target MUST identify exactly one valid destination within the program.

A target MUST NOT be ambiguous.

An unresolved target is a program-definition error rather than an ordinary runtime data error.

Where possible, such errors SHOULD be detected before execution begins.

Part 3 defines how targets are represented.

---

# 2.37 Termination

A Jaxson program terminates when it reaches an explicit or implicit termination condition defined by the language.

Normal termination produces an output document.

A program MUST NOT be considered successfully terminated merely because execution reaches an arbitrary instruction boundary.

The language MUST define what happens when:

* execution reaches the end of the instruction sequence;
* an explicit return/termination operation occurs;
* a loop terminates;
* a branch has no valid continuation.

An implementation MUST distinguish normal termination from execution failure.

---

# 2.38 Infinite Execution

A Jaxson program may theoretically contain control flow which never reaches a termination condition.

The language therefore permits a runtime to impose execution limits.

Such limits may include:

* maximum instruction count;
* maximum execution duration;
* maximum nesting depth;
* maximum memory consumption.

Exceeding a runtime limit is an execution failure.

Runtime limits are operational concerns and are specified more fully in Part 4.

A program which exceeds a runtime limit MUST NOT be reported as having successfully produced its declared output.

---

# 2.39 Side Effects

Jaxson instructions operate on execution state.

The language does not inherently grant a program access to external side effects.

In particular, a Jaxson program does not implicitly:

* access a network;
* read a filesystem;
* invoke an operating-system process;
* access environment variables;
* access a clock;
* generate random values.

Such capabilities, if ever introduced, must be explicit language features and part of the execution contract.

This restriction is important to deterministic execution.

---

# 2.40 External Data

The core Jaxson language operates on its declared input and execution state.

External data sources are not implicitly available.

A runtime MAY eventually provide explicit mechanisms for obtaining external data, but such mechanisms must define:

* the source;
* the operation;
* the returned value;
* failure behaviour;
* timeout behaviour;
* determinism;
* and the effect on reproducibility.

External access is therefore outside the core execution model unless explicitly incorporated into the language.

---

# 2.41 Instruction Atomicity

An instruction represents one logical state transition.

Where an instruction performs several internal operations, the observable result MUST conform to the instruction's defined atomic semantics.

A partially applied instruction MUST NOT expose an intermediate state to subsequent instructions unless the instruction explicitly defines such behaviour.

For example, an assignment which fails validation of its destination MUST NOT leave the destination partially modified.

This permits implementations to optimise execution without changing observable semantics.

---

# 2.42 Evaluation Order

Where an instruction has multiple operands, the language MUST define the order in which those operands are resolved whenever that order can affect observable behaviour.

In the normal case, operand resolution SHOULD be free of side effects, making evaluation order irrelevant.

Because Jaxson separates computation from mutation, ordinary operand evaluation should not itself modify state.

This provides a strong basis for deterministic implementations.

---

# 2.43 Instruction Failure

An instruction may fail when its preconditions are not satisfied.

Examples include:

* invalid reference;
* invalid destination;
* invalid operand type;
* invalid array index;
* failed computation;
* invalid control-flow target.

An instruction failure terminates normal execution unless the language explicitly provides an error-handling mechanism.

The failure is not equivalent to producing a JSON value.

In particular, an error MUST NOT silently become `null`.

---

# 2.44 Error Handling

The initial Jaxson language should treat execution errors as terminal.

A failed instruction causes the current execution to fail.

Error recovery, exception handling and conditional execution based on errors are not implicit language features.

This keeps the initial machine small and deterministic.

A future version may introduce explicit error-handling instructions if there is a demonstrated need.

Such facilities would need to define whether execution can resume and what state is visible after the failure.

---

# 2.45 Language Invariants

A conforming Jaxson language implementation MUST preserve the following invariants.

### State invariant

Instructions observe the state produced by preceding instructions.

### Reference invariant

References obtain values from defined execution contexts and do not implicitly perform computation.

### Mutation invariant

State changes occur through explicit state-changing operations.

### Computation invariant

Arithmetic and computational expressions occur within defined `compute` blocks.

### Control invariant

Control-flow operations affect execution order but do not implicitly modify ordinary data state.

### JSON invariant

Program data and produced output remain representable according to the JSON model, subject to explicitly defined language extensions.

### Determinism invariant

The same defined execution inputs and environment produce the same observable result.

### Error invariant

Execution failures are distinct from ordinary JSON values.

---

# 2.46 Minimal Language Philosophy

The Jaxson language is intentionally small.

A new operation should not be introduced merely because the operation can be expressed conveniently as a dedicated instruction.

An operation belongs in the core language when it establishes a useful primitive of the execution model.

For example, the distinction between:

```text
SET
```

and:

```text
compute
```

is fundamental because it distinguishes state mutation from calculation.

By contrast, a specialised operation which merely combines several existing primitives may not belong in the core language.

The goal is not to maximise the number of operations available to a program.

The goal is to provide a sufficiently expressive machine with a small and comprehensible instruction set.

---

# 2.47 Relationship Between Instructions and Computation

The central language distinction can be expressed as:

```text
             INSTRUCTION
                  │
        ┌─────────┼──────────┐
        │         │          │
       read     mutate     control
        │         │          │
        └─────────┼──────────┘
                  │
                  ▼
             state transition


             COMPUTE
                │
                ▼
              value
```

A Jaxson instruction determines what the machine does.

A compute block determines what value is produced.

A typical operation therefore has the conceptual form:

```text
read values
    ↓
compute result
    ↓
mutate state
    ↓
continue execution
```

This pattern is not merely an implementation convenience. It is a defining characteristic of the language.

---

# 2.48 Example of the Abstract Model

Consider an input containing:

```text
{
    "items": [
        { "price": 10 },
        { "price": 20 },
        { "price": 30 }
    ]
}
```

A Jaxson program might conceptually:

1. obtain the `items` array;
2. initialise an accumulator;
3. iterate over the items;
4. obtain the price of the current item;
5. compute the new accumulator value inside a `compute` block;
6. store the result;
7. produce an output document.

The important distinction is that the computation:

```text
accumulator + price
```

belongs to the computational domain, while:

```text
iterate
obtain
assign
produce output
```

belongs to the instruction domain.

The example does not prescribe a particular JSON syntax. It illustrates the language model that Part 3 must encode.

---

# 2.49 Language Boundaries

The core Jaxson language deliberately excludes several classes of behaviour.

Jaxson does not inherently provide:

* arbitrary native code execution;
* unrestricted expression evaluation;
* implicit network access;
* implicit filesystem access;
* implicit randomness;
* implicit time access;
* general-purpose exception handling;
* unrestricted mutation through expressions.

These exclusions are part of the language design rather than limitations of a particular implementation.

They preserve the distinction between a declarative execution language and a general-purpose programming environment.

---

# 2.50 Summary

The Jaxson language can be reduced to a small number of fundamental concepts:

```text
VALUES
  │
  ├── literals
  ├── references
  └── computed values
          │
          ▼
INSTRUCTIONS
  │
  ├── data
  ├── mutation
  ├── control
  ├── iteration
  └── computation
          │
          ▼
STATE TRANSITIONS
          │
          ▼
      TERMINATION
          │
          ▼
        OUTPUT
```

The essential language model is therefore:

> **Jaxson is a state machine whose instructions manipulate JSON-oriented state, control execution, iterate over data, and invoke explicit computation blocks to produce values.**

The language deliberately separates **state mutation**, **control flow**, and **computation**.

References obtain values.
Instructions change state or execution.
Compute blocks calculate values.
Control flow determines what executes next.
Termination produces the result.

Part 3 defines how these language constructs are represented as JSON.
