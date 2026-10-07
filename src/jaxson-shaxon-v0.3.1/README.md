# jaxson

Go implementation of Jaxson, a JSON instruction language for deterministic
stateful computation, and Shaxon, its validation layer, implemented to version 0.3.1 of the Shaxon specification.

Module: `github.com/ha1tch/jaxson`. Version: see `VERSION`.

## Requirements

- Go 1.25 or later

## Layout

| Path | Purpose |
|------|---------|
| `pkg/jaxson` | The core language: values (`Object`, `Number`), paths, operands, compute, schemas, instruction and operator tables, the closure compiler, `Machine`, `Run`, `RunJSON`, `ParseData`, `ParsePackage` |
| `pkg/jaxson/build` | Fluent, type-checked builder for writing packages in Go |
| `pkg/jaxtools` | Host tooling: `Session`, `Carry`, package and turn loading, output rendering |
| `pkg/version` | Implementation version |
| `examples/navywars` | Navy Wars written with the builder; `go run ./examples/navywars/cmd` emits its JSON |
| `cmd/jaxrun` | Run one package and print the result |
| `cmd/jaxplay` | Drive a multi-turn package (REPL or scripted turns) |
| `cmd/shaxonrun` | Run a Shaxon package: `shaxonrun [-pretty] [-steps] package.json [input.json]` |
| `pkg/shaxon` | The Shaxon dialect: shapes, indices, relations, `validate`, `check`, `shaxon.Run`, `shaxon.Validate`; 196 conformance fixtures (`shaxon-v0.3.1-fixtures.json`, see `FIXTURES.md`) |
| `examples/shaxon` | Shaxon packages to try with `shaxonrun`; `examples/shaxon/authz` has five access-by-trail examples |
| `examples/shacl` | The same five examples as SHACL-SPARQL shapes, run through pyshacl and Apache Jena, with a timing harness (`bench.sh`, `scale.py`, `report.py`), set-up scripts for Linux and macOS, and the recorded results; see its `SETUP.md` |

Try it: `go run ./cmd/shaxonrun -pretty examples/shaxon/orders.json`.

See `TRACKER.md` for status and `jaxson-shaxon-implementation-plan.md` for
the plan.

This directory sits inside a larger repository. The specifications this code
implements (`shaxon-v0.3.1-core.md` and its companions, `jaxson-v0.1.0-core-design.md`)
are at the repository root, two levels up, and `docs/IMPLEMENTATION-STATUS.md`
there summarises what runs. The fixture files beside the code are the
conformance suites.

## Build and test

```
go build ./...
go vet ./...
go test -race ./...
```

Copyright (c) 2026 haitch <h@ual.li>
