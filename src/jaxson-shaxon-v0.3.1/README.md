# jaxson

Go implementation of Jaxson, a JSON instruction language for deterministic
stateful computation, and (in progress) Shaxon, its validation layer.

Module: `github.com/ha1tch/jaxson`. Version: see `VERSION`.

## Requirements

- Go 1.25 or later

## Layout

| Path | Purpose |
|------|---------|
| `pkg/jaxson` | The core language: values, paths, operands, compute, schemas, instruction and operator tables, `Machine`, `Run` |
| `pkg/jaxson/build` | Fluent, type-checked builder for writing packages in Go |
| `pkg/jaxtools` | Host tooling: `Session`, `Carry`, package and turn loading, output rendering |
| `pkg/version` | Implementation version |
| `examples/navywars` | Navy Wars written with the builder; `go run ./examples/navywars/cmd` emits its JSON |
| `cmd/jaxrun` | Run one package and print the result |
| `cmd/jaxplay` | Drive a multi-turn package (REPL or scripted turns) |
| `cmd/shaxonrun` | Run a Shaxon package: `shaxonrun [-pretty] [-steps] package.json [input.json]` |
| `pkg/shaxon` | The Shaxon dialect: shapes, indices, relations, `validate`, `check`, `shaxon.Run`; 161 conformance fixtures (`shaxon-v0.3.1-fixtures.json`, see `FIXTURES.md`) |
| `examples/shaxon` | Shaxon packages to try with `shaxonrun`; `examples/shaxon/authz` has five access-by-trail examples |
| `examples/shacl` | The same five examples as SHACL-SPARQL shapes, run through pyshacl (Python), for comparison |

Try it: `go run ./cmd/shaxonrun -pretty examples/shaxon/orders.json`.

See `TRACKER.md` for status and `jaxson-shaxon-implementation-plan.md` for
the plan.

## Build and test

```
go build ./...
go vet ./...
go test -race ./...
```

## Licence

Apache License 2.0. See `LICENSE`.

Copyright (c) 2026 haitch <h@ual.li>
