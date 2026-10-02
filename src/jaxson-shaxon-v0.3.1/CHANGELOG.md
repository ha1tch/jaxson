# Changelog

## 0.3.1 - 2026-09-27

### Added
- `examples/navywars`: Navy Wars written with `jaxson/build`, the Go port of
  `examples/game/build_navywars.py`, plus `examples/navywars/cmd` to write
  it out as JSON (`go run ./examples/navywars/cmd navywars.json`).

### Verified
- The Go-built package is JSON-equal to `examples/game/navywars.json`
  (the acceptance test for the builder, `TestPortMatchesPythonGeneratedPackage`).
- `jaxplay -carry game -turns turns-seed0.json -format json` on the Go-built
  package and on the original produces byte-identical transcripts
  (1480 lines, reaching `status: "won"` on turn 8).
- The first attempt differed in one place only: a compute variable named
  `l` where the original used `L`. Fixed.
- `go build`, `go vet`, `gofmt -l`, `go test -race`: clean.

### Notes
- `build_navywars.py` is kept as the independent oracle the port is checked
  against; it is no longer the source of truth for new work.

## 0.3.0 - 2026-09-27

### Added
- `pkg/jaxson/build`: a fluent, chained, type-checked builder for Jaxson
  packages (plan 7.3), replacing the Python script as the way to write
  programs. Import as `jb`. Operands, expressions, instructions and schemas
  are separate sealed types; `If` without `Then`, `For` without `As`/`Do`,
  an operator below its minimum arity, a path inside an expression, and an
  unknown operator or rounding mode are compile errors. An unbound `Var`,
  a write to `input`, a shadowed loop name, a `$`-form inside `Data`, and a
  sample input that violates its schema are reported on assembly
  (`Package.Check`/`Map`/`JSON`/`Run`) by the runtime's own static checker.
  Builder values are immutable. `Template.Opt` takes a `Path`, matching the
  runtime's `$opt`.
- Tests: the `sum-of-prices` fixture rebuilt with the builder and compared to
  the fixture document; every operator constructor accepted by the runtime;
  assembly rejections; immutability; a runnable `Example`.

### Not yet done
- Port of Navy Wars to the builder, as the acceptance test against
  `examples/game/navywars.json`. `build_navywars.py` remains the source of
  truth until then.

## 0.2.0 - 2026-09-27

Phase 1 closed and verified. Implementation version; not the language
version (jaxson "1.0"; Shaxon v0.3.1 is specified but not yet implemented).

### Added
- `jaxson.CheckSchema` and `jaxson.Validate`: exported wrappers for the
  contract-schema check and value validation stages, which a host package's
  own `Run` must repeat.
- `jaxson.CheckProgramHost`: `CheckProgram` with the `Checker.Host` slot
  populated. Previously no exported path could set `Host`.
- `pkg/jaxson/extension_test.go`: nine tests, written from outside the
  package, covering host instruction registration, `Checker.Host`
  delivery, shared step accounting, `OnMutate` for all four mutation
  instructions, fresh-per-call tables, and concurrent `Run` under `-race`.
- `LICENSE`, `README.md`, `VERSION`, `pkg/version`.

### Changed
- `Run` moved from `machine.go` to its own `run.go`. No behaviour change.
- `TRACKER.md` brought up to date; implementation plan filename corrected
  in comments and the tracker.

### Verified
- `go build`, `go vet`, `gofmt -l`, `go test -race`: clean on Go 1.22.2.
- 28/28 Jaxson fixtures; 8/8 `jaxtools` tests; 9/9 extension tests.
- Navy Wars, seed 0, all 8 turns through `jaxplay`, reaches `won`.
