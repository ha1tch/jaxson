# agogo: the accessgraph pipeline in Go

Copyright (c) 2026 haitch <h@ual.li>. Licensed under Apache-2.0.

A Go-only version of the accessgraph reference side: generate a synthetic
access-graph database, decide every request, check two implementations against
each other, and time them. No Python is needed to run it.

| File | Role |
|---|---|
| `synth.go` | Port of `synth.py`: the generator, with five presets (`tiny`, `small`, `medium`, `large` = 1,000 users, `xlarge` = 5,000 users) |
| `ref.go` | Line-for-line port of `oracle.py`: top-down, memoized, string-keyed maps. The readable reference. |
| `fast.go` | The optimized oracle: same algorithm and results, different representation |
| `main.go` | `gen`, `eval`, `check` and `bench` commands |
| `agogo_test.go` | Tests and benchmarks |
| `xcheck.py` | Cross-check against `oracle.py` (needs the Python example) |

## What the optimized oracle changes

The algorithm is unchanged: for each request, enumerate the derivations that
could support it, memoized per user and delegation budget. The representation
is what differs.

- Every id is interned to an integer once, so the search never hashes or
  compares a string.
- A role's permissions are a bitmask, closed over inheritance.
- "Scope is an ancestor of resource" is two integer comparisons, using an Euler
  tour of the resource tree. This requires the resource parents to form a
  forest; the reference tolerates a parent cycle through a length cap, this
  version rejects it.
- Adjacency lists are flat arrays built in one counting pass.
- A delegation chain is a link in an arena. Its string is built only to break a
  tie between equal-hop derivations, or to report the winner, so the result is
  byte-for-byte what the string comparison in the reference gives.
- Scratch buffers are reused across users.

## Usage

```
go build -o agogo .
./agogo gen -seed 7 -size medium > db.json
./agogo eval -impl fast db.json
./agogo check -n 60        # reference vs optimized, hop limits 0 to 4
./agogo bench              # all sizes, with time and allocation
go test ./...
python3 xcheck.py                  # needs the Python example one directory up
```

## Results

Single machine, one CPU, best of several runs, each time covering index
building and evaluation of every request.

| Size | Users | Reference | Optimized | Speedup |
|---|---|---|---|---|
| tiny | 12 | 118 µs | 42 µs | 2.8x |
| small | 60 | 817 µs | 174 µs | 4.7x |
| medium | 200 | 6.6 ms | 0.65 ms | 10x |
| large | 1,000 | 67 ms | 5.8 ms | 11x |
| xlarge | 5,000 | 379 ms | 36 ms | 10x |

Agreement: the two Go oracles agree on 63,000 decisions over 60 databases at
five hop limits, and with the Python oracle on every decision of 39
Go-generated databases. The Go generator does not reproduce Python's random
stream, so the same seed gives a different database; the mix of allow, deny,
delegated and hop-bound-dependent decisions is close to the Python
generator's.

Grant and delegation ids are policy: six digits after the letter (G000001), so up to 999,999 of each. A database with another width is refused by both oracles.
