# jaxson

Go implementation of Jaxson, a JSON instruction language for deterministic
stateful computation, and (in progress) Shaxon, its validation layer.

Module: `github.com/ha1tch/jaxson`. Version: see `VERSION`.

## Requirements

- Go 1.22 or later

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

`pkg/shaxon` does not exist yet. See `TRACKER.md` for status and
`jaxson-shaxon-implementation-plan.md` for the plan.

## Build and test

```
go build ./...
go vet ./...
go test -race ./...
```

## Licence

Apache License 2.0. See `LICENSE`.

Copyright (c) 2026 haitch <h@ual.li>
