# Jaxson

**Version 0.1.0**
Jaxson is a small, closed language for writing **deterministic logic as JSON**: a package bundles an input, a program, and input and output contracts. Any conforming runtime gives it the same result, or the same error, using exact decimals, no coercion, guaranteed termination, and no I/O.

## Documentation

- [An Introduction to Jaxson](jaxson-intro.md) 
- [Jaxson Core Design v0.1.0](jaxson-v0.1.0-core-design.md) - First released version
- [Precedents](docs/PRECEDENTS.md) - Other comparable systems that came before, and how Jaxson differs
- [Showcase with worked examples](examples/showcase/README.md) — A broad array of use cases

## Code
- [Early Jaxson v0.1.0 implementation](src/jaxon-v0.1.0)
- [Most current, ongoing work on Jaxson-Shaxon](src/jaxson-shaxon-v0.3.1) - Latest extensible Jaxson implementation in Go. Includes extensions for mounting a `Dialect` such as Shaxon on top.
- [Early "Navy Wars" game implemented in Jaxson](examples/game) - Includes Go scaffolding+runtime that the earliest Jaxson versions didn't use to specify with enough precision.

## Shaxon

Shaxon is a Jackson extension. It inherits everything Jaxson provides, and extends it with graph handling primitives. As of version v0.3.1 it incorporates a number of features inspired by SHACL 1.0 and 1.1. 

Shaxon **is not** SHACL written verbatim in JSON. Due to Shaxon's inheritance of everything that Jaxson provides –especially its deterministic execution model– many of the tasks that used to demand assistance from external applications in SHACL are rendered unnecessary. That way, Shaxon is afforded the same guarantees that Jaxson has to offer.

Progress is being made to match the expressiveness of the current SHACL 1.2 draft in future Shaxon versions. Work is ongoing to reach –where possible, given the model differences– a great degree of coverage and comparable expressiveness with SHACL 1.2 as of September 2026.

## Current Shaxon Version: v0.3.1
- [Shaxon v0.3.1 Core](shaxon-v0.3.1-core.md) - Latest version
- [Shaxon v0.3.1 Limitations](shaxon-v0.3.1-limitations.md) - What the current version still can't do
- [Shaxon v0.3.1 Editorial](shaxon-v0.3.1-editorial.md) - What changed most recently in v0.3.1, how, and why
- [Shaxon v0.3.1 Proposal](shaxon-v0.3.1-proposal.md) - What was proposed after v0.3.0

---
## Previous Shaxon version: v0.3.0
- [Shaxon v0.3.0 Core](shaxon-v0.3.0-core.md) 
- [Shaxon v0.3.0 Limitations](shaxon-v0.3.0-limitations.md) 
- [Shaxon v0.3.0 Editorial](shaxon-v0.3.0-editorial.md)
- [Shaxon v0.3.0 Orientation](shaxon-v0.3.0-orientation.md) 


### Other work:
- [Earlier development](attic/)

### Research:
- [Shaxon Delta v0.0.1 - Notes 01](shaxon-delta-research/shaxon-delta-v0.0.1-notes-01.md)
- [Shaxon Delta v0.0.1 - Notes 02](shaxon-delta-research/shaxon-delta-v0.0.1-notes-02.md)



