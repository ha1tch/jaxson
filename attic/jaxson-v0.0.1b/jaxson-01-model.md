# 1. Model

## 1.1 Overview

Jaxson is a declarative instruction language for deterministic, stateful computation over JSON documents.

A Jaxson program describes a sequence of execution steps which operate on an input document, maintain execution state, and produce an output document.

Jaxson is designed for situations in which computation must be described as data rather than embedded in a conventional programming language. A Jaxson program is therefore itself a JSON document and can be stored, transmitted, validated, inspected, generated, and executed independently of the implementation language of the Jaxson runtime.

Jaxson does not treat JSON merely as a serialisation format for program source. JSON is the fundamental data model upon which the execution environment operates.

The central Jaxson model is:

```text
Input
  │
  ▼
Input validation
  │
  ▼
Jaxson program
  │
  ▼
Stateful execution
  │
  ▼
Output
  │
  ▼
Output validation
```

A successful execution therefore establishes a complete transformation from one valid JSON document to another under a specified program and pair of validation contracts.

---

## 1.2 Execution Contract

A Jaxson program does not constitute a complete executable unit by itself.

A Jaxson execution is defined by five associated JSON documents:

1. **Input** — the JSON document supplied to the program.
2. **Input schema** — the queryfy schema against which the input is validated.
3. **Program** — the Jaxson instruction sequence to be executed.
4. **Output schema** — the queryfy schema against which the resulting output is validated.
5. **Output** — the JSON document produced by execution.

These documents may be maintained as separate files or represented together as a single JSON package.

Conceptually:

```text
┌───────────────────────┐
│       Input           │
└───────────┬───────────┘
            │
            ▼
┌───────────────────────┐
│   Input Schema        │
│      queryfy          │
└───────────┬───────────┘
            │
            ▼
┌───────────────────────┐
│     Jaxson Program    │
└───────────┬───────────┘
            │
            ▼
┌───────────────────────┐
│      Execution        │
│   Stateful Machine    │
└───────────┬───────────┘
            │
            ▼
┌───────────────────────┐
│       Output          │
└───────────┬───────────┘
            │
            ▼
┌───────────────────────┐
│   Output Schema       │
│      queryfy          │
└───────────────────────┘
```

The input and output documents are data. The schemas are contracts. The Jaxson program is the executable description connecting them.

---

## 1.3 Input

The input is a single valid JSON document.

The input may be any JSON value permitted by the Jaxson implementation, including an object, array, string, number, boolean, or `null`, subject to the requirements imposed by the input schema.

The input is the externally supplied data for an execution.

The Jaxson program may inspect the input but does not thereby imply that the input itself is mutable. Input mutability is a separate semantic property of the execution state and MUST NOT be inferred merely from the existence of an input document.

The input schema establishes the conditions under which the input is considered valid for the program.

---

## 1.4 Input Validation

Before normal execution begins, the input MUST be validated against the associated input schema.

Input validation is performed using queryfy.

The purpose of input validation is to establish the precondition under which the Jaxson program is permitted to execute.

An input which does not satisfy its associated schema does not constitute a valid execution input.

Input validation and program execution are distinct operations. A failure of input validation is therefore not a Jaxson instruction failure.

The Jaxson program SHOULD be designed on the assumption that a successfully validated input satisfies the structural and value constraints established by its input schema.

Jaxson does not require the program itself to duplicate those constraints.

---

## 1.5 Program

A Jaxson program is a JSON representation of an ordered collection of instructions.

The program defines the operations performed by the Jaxson execution machine.

The program is declarative in representation but procedural in execution: its instructions establish an ordered sequence of state transitions.

The conceptual execution of a program is:

```text
initial state
     │
     ▼
instruction 1
     │
     ▼
state 1
     │
     ▼
instruction 2
     │
     ▼
state 2
     │
     ▼
...
     │
     ▼
final state
```

The precise JSON representation of instructions is defined in Part 3, Representation.

The semantics of individual instructions are defined in Part 2, Language.

---

## 1.6 Execution State

Execution proceeds against a mutable machine state.

The state represents everything that may change during execution and everything required to determine the next execution step.

At the conceptual level, execution state consists of:

* externally supplied input;
* working data;
* variables or other mutable values;
* control-flow position;
* execution context; and
* termination state.

The precise representation of these components is an implementation and language concern defined elsewhere in the specification.

The important property is that execution is stateful.

An instruction does not merely calculate an isolated result. It may transform the state in which subsequent instructions execute.

Thus, two identical instructions may produce different results when executed against different states.

---

## 1.7 State Transitions

Jaxson execution is modelled as a sequence of state transitions.

If the machine is in state `S` and executes instruction `I`, the instruction produces a subsequent state `S'`:

```text
S ──I──▶ S'
```

A complete execution can therefore be represented as:

```text
S₀ ──I₀──▶ S₁
S₁ ──I₁──▶ S₂
S₂ ──I₂──▶ S₃
...
Sₙ₋₁ ──Iₙ₋₁──▶ Sₙ
```

Execution terminates when the program reaches a defined termination condition.

The output document is derived from the final execution state according to the Jaxson program's output semantics.

This model makes state mutation a first-class property of Jaxson rather than an incidental consequence of implementation.

---

## 1.8 Data Model

Jaxson operates on JSON values.

The fundamental values are:

* object;
* array;
* string;
* number;
* boolean; and
* null.

Jaxson does not require an independent primitive data model merely to perform execution.

Where an operation requires a value of a particular kind, the operation's semantics define the permitted JSON types and the behaviour when those requirements are not satisfied.

Jaxson programs SHOULD NOT depend on values which cannot be represented consistently in JSON unless such values are explicitly defined by the language.

The output of a successful Jaxson execution MUST be a valid JSON document.

---

## 1.9 References

A Jaxson program must be able to obtain values from its execution state.

A reference identifies a value within a defined JSON structure or execution context.

Queryfy provides the path and validation model used for accessing JSON data where Jaxson delegates path resolution to queryfy.

A reference and a computation are conceptually different.

A reference answers:

> Where is the value?

A computation answers:

> What value should be produced from these values?

Jaxson maintains this distinction so that accessing data does not implicitly become an expression language.

The precise reference syntax and semantics are defined in Part 2.

---

## 1.10 Mutation

Jaxson permits execution state to be mutated by instructions.

Mutation may include operations such as:

* assigning a value;
* replacing a value;
* creating a value;
* removing a value;
* modifying an array;
* modifying an object; or
* transferring a value between locations.

Mutation is explicit.

The act of obtaining a value does not, by itself, modify state.

Similarly, the act of computing a value does not, by itself, imply mutation of the state containing the operands.

This distinction permits a Jaxson program to separate:

```text
obtain
  ↓
compute
  ↓
mutate
```

rather than treating every expression as an implicit state-changing operation.

---

## 1.11 Computation

Jaxson distinguishes **execution** from **computation**.

Execution determines the operations performed by the machine, including state mutation, control flow and iteration.

Computation determines values.

Arithmetic and related value-producing operations are restricted to `compute` blocks.

A `compute` block therefore constitutes a defined computational boundary within the otherwise instruction-oriented Jaxson model.

Conceptually:

```text
Jaxson execution
       │
       ├── state
       ├── control flow
       ├── iteration
       └── mutation
                │
                ▼
          compute block
                │
                ├── operands
                ├── arithmetic
                └── result
                │
                ▼
          Jaxson execution
```

This separation prevents arithmetic expressions from becoming a general-purpose expression syntax available throughout the language.

The semantics of `compute` blocks are defined in Part 2.

The conceptual distinction is fundamental:

> **Jaxson orchestrates execution; compute blocks calculate values.**

This model is related to the separation between orchestration and computation established by UAL.

---

## 1.12 Control Flow

Jaxson programs are ordered, but execution need not proceed through their instructions exactly once or strictly from beginning to end.

The language provides control-flow mechanisms which allow execution to:

* conditionally execute instructions;
* repeat instructions;
* iterate over collections;
* select between alternative execution paths;
* terminate execution; and
* resume execution at another defined point.

Control flow changes **which instructions execute and when**. It does not itself define the values produced by computation.

This distinction permits the language to remain an instruction-oriented execution model rather than becoming a general expression language.

The complete control-flow model is defined in Part 2.

---

## 1.13 Iteration

Iteration is a form of controlled repeated execution.

An iteration operates over a defined collection or sequence and executes a defined body for each applicable element.

Iteration may modify execution state, including state used by subsequent iterations.

Iteration therefore differs from a pure mapping operation: its body executes within the same stateful execution environment unless the language explicitly establishes a separate scope.

The semantics of:

* iteration order;
* iteration variables;
* nested iteration;
* empty collections;
* collection mutation during iteration; and
* termination

are defined by the language specification.

Where JSON does not establish an ordering that is sufficient for deterministic execution, Jaxson MUST define an explicit ordering or prohibit dependence upon that ordering.

---

## 1.14 Output

A successful Jaxson execution produces a single output JSON document.

The output is the externally observable result of the program.

The output may be:

* constructed progressively during execution;
* derived from working state;
* copied from input;
* or otherwise produced according to the program's instructions.

The mechanism by which the program establishes its output is a language concern.

The output itself is subject to the associated output schema.

---

## 1.15 Output Validation

After successful execution, the resulting output MUST be validated against the associated output schema.

Output validation is performed using queryfy.

The output schema establishes the postcondition of the Jaxson execution.

Consequently, the complete contract is:

```text
valid input
    +
valid program
    +
successful execution
    =
output satisfying the output contract
```

An execution which produces JSON but fails output validation has not successfully satisfied its Jaxson execution contract.

Output validation is distinct from execution.

A program may therefore execute without an execution error while still failing its output contract.

---

## 1.16 Jaxson and Queryfy

Queryfy provides the validation and data-querying capabilities on which Jaxson relies.

Jaxson does not need to reproduce an independent schema language merely to define the structure of its inputs and outputs.

The relationship is:

```text
             queryfy
          ┌─────┴─────┐
          │           │
       validate     query
          │           │
          ▼           ▼
       Jaxson execution
```

Queryfy is used to establish whether documents satisfy the declared contracts and to provide a defined mechanism for accessing JSON data.

Jaxson provides the execution model around those capabilities.

This separation is intentional.

Queryfy answers questions such as:

> Does this document satisfy this schema?

and:

> What value exists at this path?

Jaxson answers questions such as:

> What should happen to the state?

and:

> In what order should these operations occur?

The two systems therefore have complementary responsibilities.

---

## 1.17 Jaxson and jsonplate

Jaxson and jsonplate address different levels of JSON transformation.

jsonplate describes a fixed JSON structure into which values can be substituted.

Jaxson describes an executable stateful transformation.

Conceptually:

```text
jsonplate

fixed JSON structure
        +
value references
        │
        ▼
     JSON output
```

versus:

```text
Jaxson

input
  +
instructions
  +
state
  +
control flow
  +
computation
        │
        ▼
     JSON output
```

jsonplate is therefore a structural projection mechanism.

Jaxson is an execution mechanism.

Jaxson does not require jsonplate, and jsonplate does not require Jaxson.

The two may nevertheless be used together where a fixed output structure is required as part of a larger executable transformation.

The important design boundary is that Jaxson does not attempt to turn jsonplate into a general-purpose programming language.

---

## 1.18 Jaxson and UAL

UAL provides a related conceptual distinction between orchestration and computation.

Jaxson adopts a similar principle by restricting arithmetic and related computation to explicit `compute` blocks.

The purpose is not to make Jaxson a second implementation of UAL.

Instead, Jaxson establishes a clear boundary between:

* execution;
* state management;
* control flow; and
* computation.

This allows the Jaxson instruction model to remain small and explicit while providing a defined place for value calculation.

The exact relationship between Jaxson computation and UAL computation is an implementation and language-design concern and is specified outside the general model.

---

## 1.19 Determinism

Jaxson is intended to provide reproducible execution.

For a given:

* input;
* input schema;
* program;
* output schema; and
* defined execution environment,

a conforming implementation MUST produce the same observable result.

Jaxson programs MUST NOT implicitly depend upon uncontrolled external state.

Examples of external state which MUST NOT affect execution unless explicitly incorporated into the language include:

* current wall-clock time;
* random values;
* process-specific state;
* network responses;
* unspecified object ordering; and
* implementation-specific behaviour.

If a future Jaxson extension introduces access to an external source of nondeterminism, that behaviour must be explicitly defined by the language and reflected in the execution contract.

Determinism is a property of the execution model, not merely a desirable characteristic of a particular implementation.

---

## 1.20 Execution Boundaries

Jaxson separates four conceptually distinct stages:

```text
1. Definition
       │
       ▼
2. Input validation
       │
       ▼
3. Program execution
       │
       ▼
4. Output validation
```

Each stage has a different responsibility.

### Definition

Establishes whether the supplied Jaxson package and program are structurally valid.

### Input validation

Establishes whether the supplied input satisfies the program's declared precondition.

### Program execution

Performs the state transitions described by the Jaxson program.

### Output validation

Establishes whether execution produced a result satisfying the declared postcondition.

These boundaries MUST remain distinguishable in a conforming implementation, even where an implementation combines them internally.

---

## 1.21 Program Identity and Reproducibility

A Jaxson program is data.

It may therefore be identified, versioned, stored and transmitted independently of the runtime which executes it.

A complete reproducible execution can be represented by preserving:

```text
input
input schema
program
output schema
output
```

The resulting collection forms a complete execution record.

This makes it possible to retain not only the result of a computation but also the inputs, program and contracts necessary to explain how that result was obtained.

A Jaxson implementation MAY provide additional metadata concerning execution, such as:

* execution duration;
* instruction count;
* resource consumption;
* runtime version; or
* diagnostic information.

Such metadata is not itself part of the program's JSON output unless explicitly defined by the program.

---

## 1.22 Separation of Concerns

The Jaxson model is deliberately divided into distinct concerns:

| Concern   | Responsibility                |
| --------- | ----------------------------- |
| JSON      | Data representation           |
| queryfy   | Validation and JSON querying  |
| Jaxson    | Stateful execution            |
| `compute` | Value calculation             |
| jsonplate | Fixed structural substitution |
| Runtime   | Actual execution environment  |

No single component is required to provide all of these capabilities.

This separation is central to the design of Jaxson.

In particular, Jaxson does not attempt to become:

* a schema language;
* a JSON template language;
* a general-purpose expression language;
* a conventional general-purpose programming language; or
* an arbitrary code execution environment.

Its purpose is narrower:

> **to provide a portable, declarative description of deterministic stateful computation over JSON.**

---

## 1.23 Abstract Execution Model

The complete abstract model can be summarised as follows.

A Jaxson execution begins with a validated input document and an initial execution state.

The program is then executed as an ordered sequence of instructions.

Each instruction observes the current state and produces a subsequent state.

Control-flow instructions determine which instruction executes next.

Mutation instructions modify state.

References obtain values from defined locations.

`compute` blocks calculate values from supplied operands.

Execution continues until a defined termination condition is reached.

The resulting output document is then validated against the output schema.

Formally:

```text
Input
  │
  │ validate
  ▼
Valid Input
  │
  │ initialise
  ▼
Initial State S₀
  │
  │ execute instruction
  ▼
State S₁
  │
  │ execute instruction
  ▼
State S₂
  │
  │ ...
  ▼
Final State Sₙ
  │
  │ produce output
  ▼
Output
  │
  │ validate
  ▼
Valid Output
```

The five principal components of the Jaxson model are therefore:

```text
             ┌─────────────┐
             │    Input    │
             └──────┬──────┘
                    │
             ┌──────▼──────┐
             │ Input Schema │
             └──────┬──────┘
                    │
                    ▼
             ┌─────────────┐
             │   Jaxson    │
             │   Program   │
             └──────┬──────┘
                    │
                    ▼
             ┌─────────────┐
             │   Runtime   │
             │    State    │
             └──────┬──────┘
                    │
                    ▼
             ┌─────────────┐
             │    Output   │
             └──────┬──────┘
                    │
             ┌──────▼───────┐
             │Output Schema │
             └──────────────┘
```

This model provides the foundation for the remaining specification.

**Part 2 — Language** defines the instructions and their semantics.

**Part 3 — Representation** defines their JSON encoding.

**Part 4 — Runtime** defines the execution environment and operational requirements.

**Part 5 — Conformance** defines the requirements an implementation must satisfy.
