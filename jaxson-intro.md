## Introduction

### Motivation

Applications keep needing small pieces of logic that should change without a redeploy: a pricing rule, a payload reshaping, a validation step, a routing decision. The usual answers each fail somewhere. Embedding a general-purpose scripting language brings ambient state, I/O, unbounded running time and behaviour that differs from host to host. Ad hoc JSON "rule formats" are typically under-specified, so two implementations disagree at the edges: floating-point drift, missing versus `null`, member ordering, what a loop does when its collection changes underneath it.

Jaxson exists to close that gap. It is logic expressed as data, small enough to specify completely, that any conforming runtime can execute and that gives the same answer, or the same error, on every one of them.

### What Jaxson is

A Jaxson program is a JSON document. A complete package carries five things: the input, a contract the input must satisfy, the program itself, a contract the output must satisfy, and optionally a declared resource limit. A runtime validates the input, runs the program, then validates the output. Each failure has a category and, where relevant, a stable code, so a failure can be classified without parsing prose.

The program is a list of instructions drawn from a closed set of eight, reading and writing through four named roots (`input`, `state`, `output`, `local`). Calculation happens in **compute islands**: small pure expressions whose operands are bound up front, so they have no way to read state. Around that core sit a handful of deliberate choices:

- Numbers are exact decimals. `0.1 + 0.2` is `0.3`, and nothing is ever rounded silently.
- There is no coercion, no truthiness and no autovivification. Present-`null` and absent are always distinct.
- Contracts are closed by default: anything the schema does not name is rejected.
- Every loop iterates a finite snapshot, and a declared step limit is counted semantically. Every program therefore terminates, and exceeds its limit at exactly the same point on every runtime.
- There is no clock, no randomness and no I/O. A cutoff on time or memory can never change a result.

Jaxson descends from queryfy (contracts at the boundary), jsonplate (a document that is its own output shape) and ual's compute islands (declared inputs, no outside access), but defines its own semantics so that this specification stands on its own (section 0).

### What it is good for

- **Business rules and calculations shipped as data.** Totals, pricing, eligibility checks and similar logic, particularly where money makes exact decimal arithmetic matter.
- **Payload transformation with contracts at both ends.** Reshape one JSON document into another and be certain of what went in and what may come out.
- **Running logic you did not write.** The language cannot reach anything outside its declared input, and it cannot run forever, which makes sandboxing a property of the language rather than of the host.
- **Several runtimes that must agree.** A Go runtime and a Rust runtime, say, can differ internally and still produce identical results. The golden fixtures double as a conformance suite.
- **Auditable logic.** Paths are arrays of segments and compute islands cannot read state, so a validator can list the paths a program reads.

### What it is not for

Jaxson is deliberately not a general-purpose language. It has no jumps or `while`, no regular expressions, no floating point or transcendental functions, no locale-dependent string handling and no I/O. Text processing, numerical science and anything needing unbounded iteration belong elsewhere. Where more is needed, the intended route is a later profile behind explicit extension names, so that the core stays total and portable.
