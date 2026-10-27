# accessgraph engines

Copyright (c) 2026 haitch <h@ual.li>. Licensed under Apache-2.0.

The same access-decision rules written for other evaluators, so that the Jaxson
program can be compared with them and with the reference (`../oracle.py`). This is
scaffolding for the comparison reported in `../FINDINGS.md`, not a product. The Go
ports of the reference are in `../agogo/`.

| File | Role |
|---|---|
| `ag.dl` | The rules for Soufflé: nested groups, role inheritance, resource ancestry, grants and delegation chains, support and deny |
| `common.py` | `facts(db)` turns a database into the flat tuples a Datalog engine takes; `assemble` turns an answer back into the reference's output; `compare` runs an engine against the reference on generated databases |
| `run_souffle.py` | Soufflé in interpreter mode |
| `run_cozo.py` | The rules again in CozoDB's dialect, through its embedded Python client |
| `run_starlark.py`, `starlark/` | The rules in Starlark: `starlark/rules.star` is a port of the reference within the language as shipped (no sets, no recursion, no `while`), run by a small Go wrapper over starlark-go that applies and reports the interpreter's step budget |
| `cel/` | A probe, not a port: what CEL's static cost estimate says about a group-membership closure unrolled k rounds, beside the cost CEL counts. `./cel_probe DB.json`; a second argument `parts` shows which pieces of the expression make the estimate unbounded |
| `run_goeng.py`, `goeng/main.go` | The rules for a third-party Go Datalog library. Retired at six-digit ids: four six-digit segments do not fit 64 bits, and the script refuses with that message. Kept as the record of the third encoding |
| `mutants_engines.py` | Plants five faults in each engine's rules (a hop short and extra, denies ignored, group nesting dropped, ancestor walk dropped) and counts the databases where each is caught |
| `sweep.py` | End-to-end timings of every engine on generated databases, the source of the table in `FINDINGS.md` section 4; each engine must equal the reference on every decision |
| `amazon_one.py` | One engine on one of the Amazon variants (`../provision/amazon_to_db.py`), one result line, the source of the table in `FINDINGS.md` section 2 |

## What it needs

Python 3, and for the engines being run:

- Soufflé 2.5 with `souffle` on the path. For compiled mode, build once with
  `souffle -o ag_compiled ag.dl` in this directory (under a minute); the binary is ignored by git.
- The CozoDB Python client: `pip install cozo_embedded`.
- The Go ports: `go build -o agogo .` in `../agogo/`.
- For Starlark and CEL, Go modules fetched from the module proxy on first build: `go build -o starlark_run .` in `starlark/` (starlark-go, small) and `go build -o cel_probe .` in `cel/` (cel-go and its requirements, about 80 MB unpacked in the module cache).
- A built `jaxrun` (see `../README.md`), named by `JAXRUN` or on the path.
- For `goeng`: the Go library `swdunlop.dev/pkg`, which is not vendored here.

Paths can be overridden with `AGOGO`, `AG_COMPILED`, `AG_DL`, `JAXRUN`, `DLENG` and
`AMAZON_DB`; the defaults are relative to this directory.

## Running

    python3 run_souffle.py 20          # 20 generated databases, interpreter
    python3 run_cozo.py 20
    python3 run_starlark.py 24
    ./cel/cel_probe ../data/amazon-db/nested.json   # or any database JSON with users and members
    python3 mutants_engines.py         # five planted faults per engine
    python3 sweep.py tiny,small,medium,large,xlarge ["Souffl,jaxrun"]
    python3 amazon_one.py oracle flat  # then any engine, e.g. go-fast, souffle-compiled, jaxrun, cozo

Run `amazon_one.py oracle VARIANT` first: it writes the reference decisions that the
other engines are compared with, next to the databases. The engines are slow at the
largest sizes (CozoDB took 156 s and 185 s on the two Amazon variants).
