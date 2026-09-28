# 4. Runtime

## 4.1 Overview

The Jaxson runtime is the execution environment responsible for loading, validating and executing Jaxson programs.

The runtime provides the operational boundary between a Jaxson execution contract and the underlying host system.

The runtime is responsible for:

* loading Jaxson representations;
* validating program structure;
* validating input;
* establishing execution state;
* executing instructions;
* enforcing runtime limits;
* detecting execution failure;
* producing output;
* validating output;
* reporting execution results.

The runtime MUST implement the semantics defined by Parts 1 and 2 and MUST interpret the representation defined by Part 3.

The runtime MAY use any internal architecture capable of satisfying those requirements.

For example, a runtime MAY be:

* an interpreter;
* a compiler;
* a bytecode virtual machine;
* a tree-walking evaluator;
* a hybrid interpreter/compiler;
* an embedded library;
* a standalone executable;
* a cloud service.

The Jaxson specification does not prescribe an implementation architecture.

The fundamental runtime model is:

```text
                 Jaxson Package
                       │
                       ▼
                 Load / Parse
                       │
                       ▼
              Program Validation
                       │
                       ▼
                Input Validation
                       │
                       ▼
                State Creation
                       │
                       ▼
                  Execution
                       │
          ┌────────────┴────────────┐
          │                         │
       failure                   success
          │                         │
          ▼                         ▼
      Execution                 Produce
       Error                    Output
                                    │
                                    ▼
                             Output Validation
                                    │
                          ┌─────────┴─────────┐
                          │                   │
                       failure             success
                          │                   │
                          ▼                   ▼
                    Output Error          Result
```

---

# 4.2 Runtime Responsibilities

A conforming runtime has five principal responsibilities.

### 4.2.1 Preparation

The runtime establishes that the supplied program is structurally executable.

### 4.2.2 Validation

The runtime validates the input against its declared input schema and the output against its declared output schema.

### 4.2.3 Execution

The runtime performs the state transitions described by the program.

### 4.2.4 Resource Management

The runtime prevents a Jaxson execution from consuming unbounded host resources.

### 4.2.5 Result Reporting

The runtime communicates whether execution succeeded or failed and, on success, provides the resulting output document.

These responsibilities MUST remain semantically distinguishable even if a particular implementation combines several of them internally.

---

# 4.3 Runtime Lifecycle

A complete execution proceeds through a defined lifecycle.

The conceptual lifecycle is:

```text
LOAD
  │
  ▼
PARSE
  │
  ▼
PROGRAM VALIDATION
  │
  ▼
INPUT VALIDATION
  │
  ▼
INITIALISE
  │
  ▼
EXECUTE
  │
  ▼
FINALISE
  │
  ▼
OUTPUT VALIDATION
  │
  ▼
COMPLETE
```

An execution may terminate unsuccessfully at any stage.

A runtime MUST NOT report successful completion if any required preceding stage has failed.

---

# 4.4 Loading

Loading obtains the Jaxson representation from its source.

The source may be:

* a file;
* a byte stream;
* an in-memory JSON value;
* a database record;
* a network response;
* another application-defined source.

The source mechanism is outside the core Jaxson language.

Once loaded, the runtime receives a JSON representation to parse.

The runtime MUST NOT execute instructions merely as a consequence of loading a document.

Loading and execution are distinct operations.

---

# 4.5 Parsing

The runtime MUST parse the supplied representation as JSON.

Invalid JSON is a representation error.

A runtime MUST reject invalid JSON before attempting normal Jaxson execution.

JSON parsing errors are distinct from Jaxson structural errors.

The distinction is:

```text
invalid JSON
    ≠
valid JSON containing invalid Jaxson
```

A runtime SHOULD report these as separate error categories.

---

# 4.6 Program Validation

After parsing, the runtime validates the Jaxson program.

Program validation establishes that the representation satisfies the structural requirements of the Jaxson language.

Validation includes, where applicable:

* valid package structure;
* valid instruction objects;
* recognised operations;
* required instruction fields;
* valid operand structures;
* valid references;
* valid nested blocks;
* valid computation structures;
* valid control-flow targets;
* valid labels;
* valid instruction nesting.

Program validation SHOULD occur before input execution begins.

A runtime SHOULD detect all statically determinable program errors before executing the first instruction.

---

# 4.7 Static and Dynamic Validation

Not every error can necessarily be detected before execution.

Jaxson therefore distinguishes between:

### Static validation

Properties which can be established from the program representation without executing it.

Examples:

* unknown operation;
* malformed instruction;
* missing required field;
* malformed reference;
* duplicate label;
* unresolved static control target.

### Dynamic validation

Properties which depend upon execution state.

Examples:

* a reference to a missing value;
* an invalid runtime type;
* an invalid array index;
* division by zero;
* a computation failure;
* a runtime-generated invalid destination.

A runtime SHOULD perform as much validation as possible before execution.

It MUST nevertheless enforce dynamic requirements while executing.

---

# 4.8 Input Validation

Once the program is structurally valid, the runtime validates the input document against the associated queryfy input schema.

The runtime MUST NOT begin normal program execution if the input fails its declared schema.

Input validation therefore establishes the execution precondition.

Conceptually:

```text
Program valid
      +
Input valid
      │
      ▼
Execution may begin
```

Input validation errors are not Jaxson instruction errors.

They indicate that the execution contract was not satisfied before execution began.

---

# 4.9 Queryfy Integration

The runtime uses queryfy for the validation and querying responsibilities assigned to it by the Jaxson language.

The runtime MUST interpret the associated input and output schemas according to queryfy semantics.

The runtime SHOULD use the same query semantics for references where Jaxson delegates reference resolution to queryfy.

The runtime MUST NOT silently substitute an incompatible path or validation system while claiming conformance to the corresponding Jaxson semantics.

The relationship is:

```text
             Jaxson Runtime
                   │
          ┌────────┴────────┐
          │                 │
       execution         queryfy
          │                 │
          │          ┌──────┴──────┐
          │          │             │
          │       validation     querying
          │          │             │
          └──────────┴─────────────┘
```

---

# 4.10 Initialisation

After program and input validation, the runtime creates the initial execution state.

Initialisation establishes:

* input state;
* working state;
* output state, where applicable;
* execution context;
* control state;
* runtime counters;
* resource accounting.

The initial state MUST be deterministic for a given execution contract and runtime configuration.

The runtime MUST NOT introduce uncontrolled values into initial state.

For example, an implementation MUST NOT silently insert:

* current time;
* random values;
* process identifiers;
* host-specific values;

unless those values are explicitly part of the defined execution environment.

---

# 4.11 Initial State

The initial state contains the validated input and the initial values required by the program.

Any default values defined by the language MUST be applied consistently.

The runtime MUST distinguish between:

* a value explicitly supplied by the program;
* a value supplied by the input;
* a language-defined default;
* an implementation-defined runtime value.

Only the first three may participate in core deterministic semantics.

Implementation-defined values MUST NOT affect portable program behaviour.

---

# 4.12 Execution Context

The execution context contains runtime information required to execute the current instruction.

Conceptually:

```text
Execution Context
├── current instruction
├── program position
├── current block
├── iteration context
├── working state
└── resource state
```

The execution context is not ordinary program data.

A program cannot access arbitrary runtime internals merely because they exist.

Only context explicitly exposed by the language is available through Jaxson references or instructions.

---

# 4.13 Program Counter

The runtime maintains a logical program position.

For sequential execution, the position advances to the next instruction.

For structured control flow, the runtime enters and exits blocks according to their semantics.

For explicit branching, the runtime changes the next execution position according to the branch target.

The runtime MAY implement control flow without maintaining a literal integer program counter.

For example, a runtime may use a stack of execution frames.

Such an implementation remains conforming if its observable execution is equivalent to the abstract Jaxson machine.

---

# 4.14 Instruction Dispatch

The runtime selects the next instruction and executes it according to its defined semantics.

Conceptually:

```text
current state
     │
     ▼
select instruction
     │
     ▼
resolve operands
     │
     ▼
execute operation
     │
     ▼
produce next state
     │
     ▼
select next instruction
```

Instruction dispatch MUST preserve the semantic ordering defined by the language.

A runtime MAY optimise dispatch through:

* opcode tables;
* compiled functions;
* bytecode;
* jump tables;
* pre-resolved references;
* specialised execution paths.

Such optimisations MUST NOT change observable behaviour.

---

# 4.15 Operand Resolution

Before an instruction performs an operation, its operands are resolved according to the semantics defined in Part 2.

Resolution may involve:

* reading a literal;
* resolving a reference;
* evaluating a computation;
* obtaining an iteration value.

The runtime MUST resolve operands according to the defined evaluation order.

If operand resolution fails, the instruction MUST fail.

The runtime MUST NOT continue execution as if the operand had produced an arbitrary value.

---

# 4.16 Instruction Execution

An instruction constitutes one logical state transition.

The runtime executes the instruction against the current state and obtains either:

```text
success → next state
```

or:

```text
failure → execution error
```

The runtime MUST NOT expose partially completed state transitions as successfully committed state.

If an instruction consists internally of several implementation operations, those operations MUST collectively produce the atomic semantic result defined for that instruction.

---

# 4.17 State Mutation

The runtime applies state mutations according to the language semantics.

Mutation operations MUST respect:

* destination validity;
* value validity;
* type requirements;
* scope;
* immutability rules;
* array semantics;
* object semantics.

The runtime MUST NOT mutate state merely because a value was read.

Likewise, evaluating a computation MUST NOT implicitly mutate unrelated state.

---

# 4.18 Computation Runtime

The runtime executes `compute` blocks within the computational boundary established by Part 2.

A compute operation proceeds conceptually as:

```text
resolve operands
       │
       ▼
validate operand types
       │
       ▼
perform computation
       │
       ▼
produce result
```

The computation result is then returned to the enclosing instruction.

The runtime MUST NOT treat arbitrary JSON objects as executable computations merely because they contain operation-like members.

Only valid computation structures have computational semantics.

---

# 4.19 Numeric Semantics

The runtime MUST implement the numeric semantics defined by the Jaxson language.

The host language's numeric types MUST NOT silently determine Jaxson semantics.

For example, if Jaxson defines exact integer arithmetic, a runtime cannot substitute floating-point arithmetic merely because its implementation language makes floating-point operations convenient.

Conversely, if Jaxson defines bounded numeric behaviour, a runtime MUST enforce those bounds.

The numeric model therefore belongs to the language contract, while the internal numeric representation belongs to the runtime.

---

# 4.20 Runtime Numeric Errors

The runtime MUST detect computation errors defined by the language.

Examples include:

* division by zero;
* invalid numeric operands;
* unsupported numeric operation;
* numeric overflow where the Jaxson numeric model defines overflow;
* invalid numeric conversion.

A numeric error MUST terminate normal execution unless the language explicitly defines a recovery mechanism.

The runtime MUST NOT silently substitute:

```text
0
null
NaN
infinity
```

or another implementation-specific value unless that behaviour is explicitly part of the Jaxson numeric semantics.

---

# 4.21 Control-Flow Execution

The runtime executes control-flow instructions by selecting the next applicable instruction sequence.

For structured control flow, this may involve entering a nested block.

For explicit branching, this may involve changing the program position.

The runtime MUST preserve the semantics of:

* conditional execution;
* branching;
* iteration;
* termination.

A runtime MAY compile structured control flow into jumps internally.

This is an implementation detail.

---

# 4.22 Iteration Runtime

When an iteration begins, the runtime establishes an iteration context.

The runtime MUST:

1. resolve the collection;
2. verify that it is a valid iteration target;
3. establish the iteration order;
4. establish the iteration context;
5. execute the body;
6. advance to the next element;
7. terminate when the collection is exhausted.

The runtime MUST preserve the iteration semantics defined in Part 2.

Iteration over an array MUST be deterministic.

---

# 4.23 Iteration Frames

A runtime SHOULD represent each active iteration using an execution frame or equivalent internal structure.

A conceptual frame may contain:

```text
Iteration Frame
├── collection
├── current index
├── current value
├── iteration variable
└── body position
```

The actual representation is implementation-specific.

Nested iterations require independent frames.

When an inner iteration terminates, the runtime resumes the outer iteration according to its saved context.

---

# 4.24 Structured Execution

A runtime MAY execute nested blocks using an execution stack.

For example:

```text
Program
  │
  └── FOR
       │
       └── IF
            │
            └── COMPUTE
```

may produce an internal execution structure such as:

```text
Program Frame
  └── Iteration Frame
       └── Conditional Frame
```

This is not required by the specification.

It illustrates that nested Jaxson structures may be executed without flattening the program into a conventional source language.

---

# 4.25 Runtime Stack

If the language provides nested blocks, iteration or subprogram execution, the runtime MAY maintain a stack of execution frames.

The runtime MUST impose a limit where uncontrolled nesting could consume unbounded resources.

Exceeding the configured nesting limit is an execution failure.

A runtime MUST NOT allow a malicious or accidentally recursive program to consume unbounded stack resources merely because the host language permits it.

---

# 4.26 Program Termination

Normal execution terminates when the program reaches a valid termination condition.

At termination, the runtime establishes the output document according to the language semantics.

The runtime MUST distinguish:

```text
normal termination
```

from:

```text
execution failure
```

and:

```text
runtime cancellation
```

These conditions have different meanings and SHOULD be distinguishable to callers.

---

# 4.27 Output Construction

The runtime obtains the final output from the state established by the program.

The output MUST be a valid JSON document.

The runtime MUST NOT automatically add diagnostic information to the output merely because execution generated diagnostics.

Diagnostic information belongs to the runtime result or execution metadata unless the program explicitly constructs it as output.

This preserves the distinction between:

```text
program result
```

and:

```text
runtime information about the program
```

---

# 4.28 Output Validation

After output construction, the runtime validates the output against the associated queryfy output schema.

Output validation is mandatory for a complete execution contract.

The runtime MUST NOT report the execution as contractually successful if output validation fails.

Conceptually:

```text
program terminates
       │
       ▼
produce output
       │
       ▼
validate output
       │
   ┌───┴───┐
   │       │
 valid   invalid
   │       │
   ▼       ▼
success   failure
```

An output validation failure does not necessarily mean that the program itself was structurally invalid.

It means that the program did not satisfy its declared output contract for that execution.

---

# 4.29 Runtime Result

A runtime SHOULD expose a result structure which distinguishes successful execution from failure.

Conceptually:

```text
Execution Result
├── status
├── output
├── error
└── metadata
```

A successful result contains the output document.

A failed result contains diagnostic information sufficient to identify the failure.

The exact API returned by a particular runtime is implementation-specific.

The semantic categories of failure are defined by this specification.

---

# 4.30 Error Categories

A runtime SHOULD distinguish at least the following categories:

```text
PARSE_ERROR
PROGRAM_ERROR
INPUT_ERROR
EXECUTION_ERROR
RESOURCE_ERROR
CANCELLATION
OUTPUT_ERROR
RUNTIME_ERROR
```

### PARSE_ERROR

The supplied representation is not valid JSON.

### PROGRAM_ERROR

The JSON is valid but does not represent a valid Jaxson program.

### INPUT_ERROR

The supplied input does not satisfy its declared input schema.

### EXECUTION_ERROR

A valid program encountered a semantic failure during execution.

### RESOURCE_ERROR

Execution exceeded an applicable resource limit.

### CANCELLATION

Execution was explicitly cancelled.

### OUTPUT_ERROR

The program produced an output which failed its declared output schema.

### RUNTIME_ERROR

The host runtime itself encountered an unexpected failure.

The exact error taxonomy exposed by an implementation MAY contain additional information.

---

# 4.31 Error Localisation

Where possible, runtime errors SHOULD identify:

* the program;
* instruction position;
* operation;
* relevant block;
* iteration context;
* reference;
* computation;
* error category.

For example, an error might conceptually identify:

```text
program: calculate-total
instruction: 17
operation: compute
context: items[4]
error: division by zero
```

The exact diagnostic format is implementation-specific.

Diagnostics MUST NOT alter program semantics.

---

# 4.32 Error Propagation

By default, an execution error terminates the current execution.

The runtime MUST NOT silently continue from a failed instruction.

For example:

```text
compute
  ↓
error
  ↓
execution terminates
```

rather than:

```text
compute
  ↓
error
  ↓
use null
  ↓
continue
```

unless the language explicitly defines error recovery.

This provides predictable failure behaviour.

---

# 4.33 Cancellation

A runtime MAY permit external cancellation of an execution.

Cancellation may originate from:

* caller request;
* request timeout;
* service shutdown;
* worker termination;
* resource policy.

Cancellation is not equivalent to successful termination.

A cancelled execution MUST NOT produce a successful Jaxson result.

The runtime SHOULD make cancellation distinguishable from ordinary instruction failure.

---

# 4.34 Time Limits

A runtime MAY impose a maximum execution duration.

A time limit exists to protect the host environment from programs which execute indefinitely or consume excessive resources.

If the time limit is exceeded, execution MUST terminate with a resource or cancellation condition.

The time limit MUST NOT silently become part of program semantics.

Two executions with different runtime time limits may therefore have different operational outcomes even when the program itself is deterministic.

The runtime configuration SHOULD be observable where reproducibility requires it.

---

# 4.35 Instruction Limits

A runtime MAY impose a maximum number of executed instructions.

This limit protects against:

* infinite loops;
* unexpectedly large iterations;
* pathological control flow;
* resource exhaustion.

The instruction count SHOULD count semantic instruction executions rather than host-language operations.

For example, one Jaxson instruction which internally performs ten host-language operations remains one Jaxson instruction.

The exact accounting mechanism is runtime-defined.

---

# 4.36 Memory Limits

A runtime MAY impose memory limits.

Memory limits may apply to:

* working state;
* intermediate values;
* computation;
* execution frames;
* loaded programs;
* runtime metadata.

Exceeding the configured memory limit MUST terminate execution.

The runtime SHOULD distinguish resource exhaustion from ordinary program errors.

---

# 4.37 Output Size Limits

A runtime MAY impose a maximum output size.

This protects systems which transport or persist Jaxson results.

If output exceeds the configured limit, the execution MUST fail.

The runtime MUST NOT silently truncate output and report successful execution.

Truncation, if ever supported, would constitute a different output semantics and must therefore be explicitly defined.

---

# 4.38 Nesting Limits

A runtime MAY impose limits on:

* nested blocks;
* nested iterations;
* reference depth;
* computation depth.

These limits protect the runtime from pathological or malicious inputs.

Exceeding such a limit is a resource failure.

The runtime SHOULD make applicable limits configurable.

---

# 4.39 Resource Accounting

Resource limits SHOULD be measured consistently.

At minimum, a runtime SHOULD be able to account for:

* instruction executions;
* execution duration;
* memory usage;
* nesting depth.

The runtime MAY provide additional metrics.

Resource accounting MUST NOT alter ordinary program semantics except where the program exceeds a defined limit.

---

# 4.40 Isolation

A Jaxson runtime SHOULD execute programs within a controlled environment.

A program MUST NOT automatically inherit arbitrary capabilities from the host process.

In particular, core Jaxson execution does not grant access to:

* filesystem contents;
* operating-system processes;
* network sockets;
* environment variables;
* host credentials;
* unrelated application state.

If a runtime provides such facilities, they MUST be explicit capabilities.

This is particularly important when Jaxson programs are obtained from external or untrusted sources.

---

# 4.41 Capability Model

Future Jaxson extensions MAY define capabilities.

A capability grants a program access to a specific external facility.

For example, a hypothetical runtime might explicitly grant:

```text
network.read
```

or:

```text
clock.read
```

The capability itself becomes part of the execution environment.

A program which requires a capability not granted by the runtime MUST fail rather than silently receiving an alternative capability.

Capabilities should be explicit because they affect both security and reproducibility.

---

# 4.42 Deterministic Runtime Environment

The core runtime environment SHOULD be deterministic.

The runtime MUST NOT inject uncontrolled values into execution state.

If runtime metadata such as timestamps or identifiers is made available to programs, that access must be explicitly defined.

This preserves the core relationship:

```text
same contract + same environment
             =
        same result
```

The environment therefore becomes part of reproducibility whenever it exposes observable state.

---

# 4.43 External Services

The core runtime does not require external services.

A runtime MAY integrate Jaxson with:

* databases;
* APIs;
* message queues;
* object stores;
* cloud services.

Such integration is outside the core language unless explicitly defined as a Jaxson capability.

External operations introduce additional concerns:

* latency;
* failure;
* retries;
* timeouts;
* authentication;
* nondeterminism;
* side effects.

Consequently, external services MUST NOT become implicit runtime behaviour.

---

# 4.44 Runtime Concurrency

The core Jaxson execution model is sequential.

A runtime MAY execute independent Jaxson executions concurrently.

For example:

```text
Execution A ───────────────▶
Execution B ───────────────▶
Execution C ───────────────▶
```

Each execution MUST have isolated execution state.

Concurrent executions MUST NOT implicitly share mutable Jaxson state.

A runtime MAY share immutable program representations between executions for efficiency.

---

# 4.45 Reentrancy

A Jaxson program SHOULD be safely executable multiple times by the same runtime.

Executing:

```text
Program + Input A
```

must not modify the program such that a later execution:

```text
Program + Input B
```

observes state left behind by the first execution.

The runtime MUST therefore distinguish immutable program representation from per-execution mutable state.

This is particularly important for long-lived cloud runtimes.

---

# 4.46 Program Caching

A runtime MAY cache validated or compiled Jaxson programs.

Caching MUST NOT introduce observable state between executions.

A cached program representation may be reused:

```text
program
   │
   ├── execution A → state A
   ├── execution B → state B
   └── execution C → state C
```

but the execution states MUST remain independent.

A runtime MUST NOT cache mutable execution state as though it were immutable program state.

---

# 4.47 Program Preparation

A runtime MAY perform a preparation phase after program validation.

Preparation may include:

* resolving labels;
* constructing instruction tables;
* compiling computations;
* pre-validating references;
* preparing control-flow structures;
* allocating immutable program metadata.

Preparation MUST NOT mutate execution input or working state.

Preparation SHOULD be deterministic.

---

# 4.48 Runtime Optimisation

Implementations MAY optimise execution.

Possible optimisations include:

* constant folding;
* instruction dispatch optimisation;
* reference pre-resolution;
* control-flow compilation;
* memory reuse;
* program caching;
* computation specialisation.

An optimisation is conforming only if it preserves observable Jaxson semantics.

In particular, an implementation MUST NOT perform an optimisation which changes:

* output;
* error behaviour;
* instruction ordering where observable;
* reference semantics;
* numeric semantics;
* iteration ordering.

---

# 4.49 Constant Computation

A runtime MAY precompute a computation whose operands are entirely literal and whose semantics are deterministic.

For example:

```text
compute 10 + 20
```

may be evaluated before execution begins.

The result MUST be indistinguishable from evaluating it during normal execution.

If a computation depends on runtime state, it MUST be evaluated according to normal execution semantics.

---

# 4.50 Reference Optimisation

A runtime MAY precompile or otherwise optimise reference resolution.

For example, a path may be parsed once and represented internally as a sequence of access operations.

This is an implementation detail.

The resulting behaviour MUST remain equivalent to resolving the reference according to the specified query semantics at execution time.

If the referenced state is mutable, the runtime MUST NOT incorrectly cache the value itself across state changes.

---

# 4.51 Runtime Observability

A runtime SHOULD provide sufficient information to diagnose execution without modifying the program's output.

Useful runtime information includes:

* execution identifier;
* program identifier;
* runtime version;
* instruction count;
* execution duration;
* memory usage;
* failure location;
* failure category.

Observability information is runtime metadata.

It MUST NOT be silently inserted into program output.

---

# 4.52 Execution Tracing

A runtime MAY support instruction tracing.

A trace may record:

```text
instruction
state before
operation
state change
state after
```

or a more compact representation.

Tracing MUST NOT alter the semantics of execution.

A runtime SHOULD make tracing optional because detailed state capture may consume substantial resources.

Tracing untrusted programs SHOULD itself be subject to resource limits.

---

# 4.53 Debugging

A runtime MAY provide debugging facilities such as:

* breakpoints;
* instruction stepping;
* state inspection;
* reference inspection;
* execution traces.

These are runtime facilities rather than language features unless explicitly incorporated into the language.

Debugging operations MUST NOT alter the semantics of the program.

---

# 4.54 Runtime Version

A runtime SHOULD identify its implementation version.

The runtime version is useful for diagnostics and reproducibility.

A program's semantics MUST NOT silently depend on a runtime version unless the relevant specification explicitly defines version-dependent behaviour.

If incompatible runtime behaviour exists between versions, the implementation SHOULD expose an explicit compatibility or specification version.

---

# 4.55 Specification Version

A Jaxson representation MAY identify the specification version against which it was created.

This allows a runtime to determine whether it supports the program's language features.

A program requiring unsupported language semantics MUST be rejected rather than partially executed.

The specification version and runtime implementation version are distinct concepts.

For example:

```text
Jaxson specification: 1.0
Runtime implementation: 2.3.1
```

The runtime may support specification version 1.0 while having an implementation version 2.3.1.

---

# 4.56 Compatibility

A runtime is compatible with a Jaxson specification version if it implements the required semantics of that version.

Compatibility MUST NOT mean merely that the runtime can parse the program.

A runtime which can parse a program but does not implement one of its required instructions is not compatible with that program.

Unsupported features MUST result in a clear program or compatibility error.

---

# 4.57 Host Language Independence

The Jaxson runtime may be implemented in any language.

The host implementation language MUST NOT determine Jaxson semantics.

For example, a runtime written in Go, Rust, Java, Python or another language must implement the same:

* instruction semantics;
* reference semantics;
* numeric semantics;
* control-flow semantics;
* iteration semantics;
* validation boundaries.

This allows Jaxson programs to remain portable between implementations.

---

# 4.58 Runtime API

A runtime implementation will typically expose an API conceptually equivalent to:

```text
load
validate
execute
result
```

A minimal embedding API may therefore look conceptually like:

```text
Execute(program, input) → result
```

where the program carries or is associated with its schemas.

A richer API may expose:

```text
Prepare(program)
Validate(program)
Execute(program, input)
```

The Jaxson specification does not require a particular programming-language API.

The observable semantic lifecycle remains normative.

---

# 4.59 Batch Execution

A runtime MAY support executing multiple independent inputs against one program.

For example:

```text
             Program
                │
       ┌────────┼────────┐
       ▼        ▼        ▼
    Input A   Input B   Input C
       │        │        │
       ▼        ▼        ▼
    Output A Output B Output C
```

Each execution MUST have independent mutable state.

The program itself MAY be shared.

Batch execution MUST therefore behave equivalently to executing each input independently.

---

# 4.60 Streaming Execution

A runtime MAY support streaming inputs or outputs.

Streaming is not part of the core Jaxson language unless explicitly specified.

A streaming runtime MUST preserve the semantics of the underlying Jaxson execution.

In particular, streaming MUST NOT silently change:

* ordering;
* mutation semantics;
* iteration semantics;
* output validity.

If the core program requires the entire input document before execution, a streaming interface may still buffer the input internally.

---

# 4.61 Failure Isolation

A runtime executing one Jaxson program MUST prevent its failure from corrupting independent executions.

For example:

```text
Execution A → failure
Execution B → successful execution
```

Execution A MUST NOT modify the program, runtime state or execution context used by B.

This is particularly important for:

* worker pools;
* server processes;
* cloud functions;
* multi-tenant services.

---

# 4.62 Security Boundary

The runtime constitutes the primary security boundary between a Jaxson program and its host environment.

A program should be assumed to be potentially untrusted unless the deployment environment establishes otherwise.

The runtime SHOULD therefore provide:

* bounded execution;
* bounded memory;
* controlled capabilities;
* isolated mutable state;
* explicit external access;
* safe failure handling.

The core language does not itself provide operating-system security.

The runtime environment remains responsible for enforcing host-level security boundaries.

---

# 4.63 Resource Exhaustion

Resource exhaustion is a valid runtime failure condition.

A program MUST NOT be permitted to consume unlimited:

* CPU;
* memory;
* stack;
* output space;
* execution time.

A runtime SHOULD expose configurable limits suitable for its deployment environment.

For example, a cloud service may impose significantly lower limits than an offline development tool.

The language semantics remain the same; only the permitted execution envelope changes.

---

# 4.64 Runtime Determinism

The runtime MUST preserve the deterministic semantics of the language.

For executions which remain within their resource limits:

```text
same program
+
same input
+
same schemas
+
same defined environment
=
same observable result
```

Differences in:

* CPU architecture;
* host operating system;
* implementation language;
* internal data structures;
* execution optimisation

MUST NOT produce different Jaxson results where the specification defines a deterministic operation.

This requirement is especially important for numeric computation and collection ordering.

---

# 4.65 Runtime and Numeric Portability

Numeric operations require particular care.

A runtime MUST implement the Jaxson numeric model independently of the host architecture where the language requires architecture-independent results.

For example, if Jaxson specifies exact integer arithmetic, an implementation MUST NOT permit machine word size to alter the result.

Likewise, if Jaxson specifies decimal semantics, a runtime MUST implement those semantics rather than relying on whatever floating-point behaviour happens to be provided by the host language.

Numeric portability is therefore a runtime conformance requirement.

---

# 4.66 Runtime and JSON Fidelity

The runtime MUST preserve JSON values accurately.

In particular:

* object values;
* array ordering;
* strings;
* booleans;
* null;
* numbers

must not be changed merely as a side effect of parsing or internal representation.

Where the language requires distinctions which JSON itself does not encode, those distinctions must be represented internally without changing the externally visible JSON semantics.

---

# 4.67 Runtime Shutdown

A runtime MAY be shut down while executions are active.

Shutdown behaviour MUST be defined by the hosting environment.

An implementation SHOULD provide a mechanism to:

* reject new executions;
* allow active executions to complete; or
* cancel active executions.

The runtime MUST NOT report cancelled executions as successful merely because the host process terminated.

---

# 4.68 Runtime Recovery

A runtime MAY recover from failures affecting an individual execution.

For example:

```text
Execution A
    ↓
runtime-level failure
    ↓
discard execution state
    ↓
Execution B
    ↓
normal execution
```

Recovery mechanisms MUST preserve execution isolation.

A runtime MUST NOT reuse partially corrupted state from a failed execution.

---

# 4.69 Runtime State Persistence

The core runtime does not require persistent execution state.

A runtime MAY provide:

* checkpoints;
* suspension;
* resumption;
* durable execution;
* execution replay.

Such facilities are extensions to the basic runtime model.

If execution can be resumed, the implementation MUST preserve enough state to reproduce the semantics of uninterrupted execution.

---

# 4.70 Replay

A runtime MAY support replaying a previous execution.

Replay requires preserving all externally observable inputs to the execution.

At minimum this includes:

```text
program
input
input schema
output schema
runtime-relevant environment
```

If external capabilities are involved, the relevant external results must also be captured or deterministically reproduced.

A replay which changes observable program behaviour is not a faithful replay.

---

# 4.71 Runtime Profiles

A deployment may define a runtime profile.

A profile may specify:

* supported specification version;
* resource limits;
* available extensions;
* available capabilities;
* numeric implementation;
* maximum program size;
* maximum output size.

Profiles allow the same core language to operate in different environments without changing its fundamental semantics.

For example:

```text
Development Profile
Cloud Worker Profile
Embedded Profile
Restricted Profile
```

A profile MUST NOT silently change core language semantics.

---

# 4.72 Core Runtime

The core Jaxson runtime should require no external side effects.

Its responsibilities are limited to:

```text
JSON
  +
Jaxson program
  +
queryfy validation/querying
  ↓
deterministic execution
  ↓
JSON
```

This defines the smallest useful runtime.

Additional capabilities can then be layered on top without making them implicit dependencies of the language.

---

# 4.73 Runtime Conformance

A runtime conforms to this specification only if it:

1. correctly parses valid Jaxson representations;
2. rejects invalid representations;
3. validates programs before execution where statically possible;
4. validates input against the declared input schema;
5. establishes execution state correctly;
6. executes instructions according to Part 2;
7. enforces computation semantics;
8. preserves deterministic behaviour;
9. enforces applicable resource limits;
10. distinguishes execution failures from successful output;
11. produces valid JSON output on successful execution;
12. validates output against the declared output schema; and
13. does not expose unintended host capabilities to core Jaxson programs.

---

# 4.74 Runtime Architecture

The specification does not require a particular internal architecture.

A conceptual implementation may nevertheless be represented as:

```text
┌──────────────────────────────────────────────┐
│                 Jaxson Runtime              │
│                                              │
│  ┌─────────┐   ┌──────────┐   ┌──────────┐ │
│  │ Loader  │ → │ Validator│ → │ Preparer │ │
│  └─────────┘   └──────────┘   └────┬─────┘ │
│                                     │       │
│                                     ▼       │
│                              ┌────────────┐ │
│                              │  Executor  │ │
│                              └─────┬──────┘ │
│                                    │        │
│                  ┌─────────────────┼──────┐ │
│                  ▼                 ▼      ▼ │
│               State           Compute  Control│
│                  │                 │      │ │
│                  └─────────────────┼──────┘ │
│                                    ▼        │
│                              ┌───────────┐  │
│                              │ Finaliser │  │
│                              └─────┬─────┘  │
│                                    ▼        │
│                             Output Validator│
└──────────────────────────────────────────────┘
```

This is an architectural illustration only.

An implementation may combine or divide these components differently.

---

# 4.75 Runtime Design Principle

The runtime should remain deliberately boring.

The language defines the interesting behaviour.

The runtime's job is to execute that behaviour reliably, deterministically and safely.

In particular, runtime implementation should not introduce hidden language features merely because they are convenient to implement.

The boundary is:

```text
Part 2
"What does this instruction mean?"

Part 3
"How is that instruction represented?"

Part 4
"How does a runtime execute it safely and consistently?"
```

Keeping those questions separate allows multiple runtime implementations to exist without fragmenting the language.

---

# 4.76 Summary

The Jaxson runtime is the execution environment for the language defined by Parts 1–3.

Its fundamental lifecycle is:

```text
load
  ↓
parse
  ↓
validate program
  ↓
validate input
  ↓
initialise state
  ↓
execute
  ↓
produce output
  ↓
validate output
  ↓
return result
```

During execution, the runtime:

* maintains mutable state;
* resolves references;
* executes instructions;
* performs computation;
* manages control flow;
* performs iteration;
* enforces resource limits;
* detects failures;
* maintains execution isolation.

The runtime MUST remain independent of the implementation language and MAY use any internal execution architecture.

Its central contract is:

> **Given a valid Jaxson execution contract and a defined runtime environment, execute the program according to the Jaxson language semantics, within the permitted resource envelope, and produce either a valid result or an explicit failure.**

Part 5 defines how implementations demonstrate that they satisfy this contract.
