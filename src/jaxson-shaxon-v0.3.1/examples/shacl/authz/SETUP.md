# Setting up and running the harnesses

This folder runs the five authorisation examples through three engines and
compares them. This guide covers each harness: what it needs, how to set it
up, how to launch it, and what a good run looks like.

```
./setup.sh --check     what is installed and what is missing; changes nothing
./setup.sh             pyshacl, in a local .venv
./setup.sh --jena      also Apache Jena's jars, in .jena/lib
./run.sh help          the targets
./run.sh all           everything that is installed
```

All commands are run from this folder (`examples/shacl/authz`). The scripts
are POSIX `sh`; they were written and run on Linux.

## What each harness needs

| Harness | Launch | Needs | Install |
|---|---|---|---|
| Shaxon cases | `go test ./pkg/shaxon -run TestAuthzExamples` from the module root | Go 1.25 or later | none |
| pyshacl cases | `./run.sh cases` | Python 3 (run with 3.13); pyshacl 0.40 or later (brings rdflib) | `./setup.sh` |
| Fail-open check | `./run.sh no-structure` | as pyshacl cases | `./setup.sh` |
| Mutation check (SHACL rules) | `./run.sh mutants` | as pyshacl cases | `./setup.sh` |
| Mutation check (Shaxon packages) | `python3 ../../shaxon/authz/mutants.py -v` | Go; no Python packages | none |
| Jena cases | `./run.sh jena` | Java 17 or later; Maven, only to fetch the jars | `./setup.sh --jena` |
| Timing | `./run.sh timing` | Go; pyshacl; Jena if you want its column (its engine-only column also needs `javac`, from a JDK) | as above |

Disk and network: `./setup.sh` downloads pyshacl and its dependencies (about
36 MB installed). `./setup.sh --jena` downloads Jena's jars through Maven
(about 17 MB of jars, plus whatever Maven caches under `~/.m2`, which was
about 42 MB when measured). Nothing is installed outside this folder and the
Maven cache. `.venv` and `.jena` are local and can be deleted at any time.

## 1. Shaxon (the reference side)

The Shaxon packages and their cases live in `../../shaxon/authz`. Their test
is part of the Go module:

```
cd ../../..                                  # the module root
go test -race -count=1 ./pkg/shaxon -run TestAuthzExamples
```

Good: `ok`. The test merges each case over its package's default input,
compares the decision and the ordered reasons (or the expected error), and
checks the run is deterministic. The SHACL harnesses below read the very same
case files, so nothing here needs to be kept in step by hand.

To try one package yourself:

```
go run ./cmd/shaxonrun -pretty examples/shaxon/authz/four-eyes-release.json
```

## 2. pyshacl

```
./setup.sh                  # creates .venv and installs requirements.txt
./run.sh cases              # every case; exit 0 if all comparable ones agree
./run.sh cases-verbose      # one line per case, with the time taken
```

Good: the last line reads `agree: 50, differ: 0, n/a: 1`. The `n/a` is the
case whose expectation is `RESOURCE_ERROR/STEPS`; SHACL has no step budget, so
there is nothing to compare. An expected `EXECUTION_ERROR` (a malformed
request) counts as met by any deny, since SHACL reports violations and does
not raise errors.

If the interpreter is not `python3`, set `PYTHON` for `./setup.sh`; once the
venv exists `run.sh` uses it, and `PYTHON` overrides it for `run.sh` too.

## 3. The fail-open check

```
./run.sh no-structure
```

This drops every `ex:RequestStructure` shape and runs the cases again. The
rules alone allow every malformed request, so exactly 6 cases differ, and the
script treats that as the expected result (`differ: 6`). If a different number
differs, the script says so and exits 1.

## 4. The mutation check

```
./run.sh mutants
```

`mutants.py` copies the `examples` tree to a temporary directory, breaks one
rule at a time (a comparison flipped, a join dropped, an aggregate changed),
and requires `run_cases.py` to notice. Good: `tried 19 killed 19 survived []`.
A survivor means a rule can be broken without any case noticing, which points
at a missing case, not at the rule. Each mutant is a line in `mutants.py`: the
file, the exact text, and its replacement; the text must occur exactly once.

## 5. Apache Jena

```
./setup.sh --jena           # needs java and mvn on PATH
./run.sh jena
```

`setup.sh --jena` runs `mvn dependency:copy-dependencies` on `pom.xml`
(`org.apache.jena:jena-cmds:6.2.0`) into `.jena/lib`. Nothing is compiled.
`run.sh jena` then starts `shacl.shacl validate` once per case, so a full run
takes about 40 seconds (about 0.7 s of JVM start per case). Good: the last line
is the same as for pyshacl.

To use jars you already have, set `JENA_LIB` to their directory.

How the harness reads Jena's report differs from pyshacl; see "A second
processor: Apache Jena" in `README.md` for why (`sh:sourceConstraint` names the
shape in Jena, the constraint in pyshacl).

## 6. Timing across the three engines

```
./run.sh timing             # 100, 1000 and 4000 events, both profiles
./run.sh timing-end-to-end  # only the profile that counts the RDF lift
./run.sh timing-engine      # only the profile that leaves the lift out
python scale.py 500 2000    # other sizes, with the venv's python
python scale.py --profile engine 20000
```

There are two profiles. **End-to-end** times what a caller holding the JSON
input pays: the SHACL figures include lifting the input to RDF, and Jena's also
writing Turtle files and starting a JVM. **Engine-only** leaves all of that out
and times the validation alone on data already in each engine's own form
(`shaxon.Run` on a decoded package; `pyshacl.validate` on a lifted graph;
Jena's validator warmed up inside one JVM). It prints the lift as a separate
column, so the difference stays visible. Use the second when the first would be
an unfair comparison, for example when the data already is RDF.

`scale.py` builds `shaxonrun` (end-to-end) and `enginetime` (engine-only) into
a temporary directory with `go build`, generates fixed-seed inputs for all five
examples (`rolling-quota`, `chinese-wall`, `four-eyes-release`,
`delegation-chain`, `break-glass`), and times each engine (best of three
end-to-end; best of ten after a warm-up engine-only). Options:
`--example NAME[,NAME]` (default all), `--profile`, `--budget SECONDS`,
`--json FILE` (append one JSON line per cell, for `report.py`), `--repeat N`
and `--engine-runs N`. The budget (default 120) is the longest any one cell may
take: after each size the next is projected from the growth seen so far (at
least linear), and a size whose projected single run exceeds the budget is
skipped, and so is every larger one, shown as `skipped`. A size that fits gets
as many repeats as the budget allows, up to the usual number. It checks that all
engines return the same verdict on every row and exits 1 if not (end-to-end
compares the reasons too; engine-only compares allow or deny). The Jena column
appears only if its jars are present, and its engine-only column only if
`javac` is also on `PATH` (it compiles `JenaTime.java` on the fly). The `rolling-quota` package it uses is a temporary copy with
`limits.steps` raised, because the shipped package refuses long trails on
purpose.

### The full comparison, in batches (`bench.sh`)

All five examples at 100, 1000, 4000 and 20000 events is long enough that a
single command can hit a session limit, and on a laptop it should survive being
interrupted. `bench.sh` runs it as ten batches (an example times a profile),
one after another, each in its own file under `results/`, and skips the ones
already done:

```
./bench.sh list                       # the batches and which are done
./bench.sh run                        # all ten; run it again to continue after an interruption
./bench.sh run break-glass            # one example, both profiles
./bench.sh run break-glass:engine     # one batch
./bench.sh --sizes "100 1000" --budget 30 run     # a quick pass
./bench.sh report                     # results/REPORT.md: tables with Shaxon's speed relative to each engine
```

Each batch's worst case is bounded by the budget times the number of sizes and
engines; with the default 120 s a batch rarely takes more than a few minutes,
and the whole run is about half an hour on two cores. `results/*.jsonl` also
records the host (CPU, Go, Java, pyshacl and rdflib versions), so a result can
be read later without guessing where it came from.

## macOS (Homebrew)

```
./setup-mac.sh --check      # what is present, what would be installed; changes nothing
./setup-mac.sh              # asks, then: brew install go openjdk maven (whichever are missing), then ./setup.sh --jena
./bench.sh run              # loads env-mac.sh itself
```

`setup-mac.sh` needs Homebrew already (it prints the installer line from
https://brew.sh if it is missing and stops). It installs only what is missing,
after asking (`--yes` skips the question). The JDK stays inside Homebrew's
prefix and `env-mac.sh` sets `JAVA_HOME` and `PATH` for it, so there is no
`sudo` and nothing is linked into `/Library/Java`; source that file
(`. ./env-mac.sh`) in a shell where you want to run `java`, `mvn` or `go`
by hand. The system `python3` of Xcode's command line tools (3.9) is enough for
pyshacl; the script installs `python@3.13` only if there is none. The scripts
are POSIX `sh` and use nothing that differs between GNU and BSD tools. They were
written and tested on Linux; the macOS path (Homebrew detection, `JAVA_HOME`,
the brew formula names) was exercised only against a stub `brew`, not on a Mac,
so report anything that differs.

## Everything at once

```
./run.sh all
```

Runs the pyshacl cases, the fail-open check, the mutants and, if the jars are
present, the Jena cases. It exits 1 if any of them fails; a Jena run is
skipped, with a line saying so, when the jars are missing.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| `run.sh: no Python at .../.venv/bin/python` | `./setup.sh` has not been run, or `PYTHON` points elsewhere |
| `no Jena jars in .../.jena/lib` | `./setup.sh --jena`, or set `JENA_LIB` |
| `Picked up JAVA_TOOL_OPTIONS` on every Java line | Your environment sets it. The harness clears it for the Jena processes it starts; Maven is also started with it cleared |
| `SLF4J(W): No SLF4J providers were found` | Harmless. Jena logs to a no-op logger |
| `jena failed (...)` with a stack trace | Check `java -version` is 17 or later and `.jena/lib` holds `jena-cmds-*.jar` |
| `scale.py: go not found` | Put Go 1.25 or later on `PATH`; only the timing needs it |
| `./run.sh no-structure` says UNEXPECTED | The number of malformed-request cases changed, or a structure shape no longer catches them. Run `python run_cases.py --no-structure --verbose` |
| Disagreements in `scale.py` | An engine disagrees with the others on a generated trail. The row shows each verdict; reproduce with that size |

## Versions this was run with

pyshacl 0.40.1, rdflib 7.6.0, Apache Jena SHACL 6.2.0, Python 3.13, Java 21,
Go 1.27. Other versions are untested.
