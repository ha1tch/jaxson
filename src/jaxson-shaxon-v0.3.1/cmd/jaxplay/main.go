// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// jaxplay is a general-purpose, package-agnostic developer tool for
// exercising a multi-turn Jaxson package. Flags come before the
// package path — Go's flag package stops parsing at the first
// non-flag argument, so this order is required, not stylistic:
//
//	jaxplay [-format text|markdown|json] [-carry key] [-carry-whole] [-turns turns.json] <package.json>
//
// It knows nothing about any one package's own field conventions.
// Earlier, this command was a direct port of the Navy-Wars-specific
// src/jaxson-v0.1.0/play.go — hardcoding "action"/"x"/"y"/"seed" on the
// way in and "view"/"messages"/"game" on the way out. That original
// remains untouched, still run against the old single-file jaxrun.go:
// it is the project's one complete worked example, so it stays working
// on purpose, as a verification oracle. This command replaces its
// in-tree successor with something that plays *any* Jaxson package,
// not just that one, by reading and printing whatever input/output
// shape a package's own schemas declare instead of assuming Navy
// Wars'. Game logic still lives entirely in the package; this host
// loop only reads a turn, calls jaxtools.Session.Step, and renders
// the result.
//
// Modes:
//
//   - Interactive (default): each line of stdin is parsed as one
//     turn's JSON input object; "q" or "quit" exits.
//   - Scripted (-turns file.json): file.json is a JSON array of
//     per-turn input objects, run in order, non-interactively — the
//     generalisation of the old play.go's "-seed N" convenience: a
//     fixed sequence of turns is exactly as reproducible as a fixed
//     seed, for any package, not just one that happens to accept a
//     "seed" field.
//
// -carry key: before every turn after the first, splice carried state
// into this turn's input under key. Which field, if any, a package
// uses to carry state forward is that package's own convention, not
// part of the Jaxson spec (whose only fixed roots are
// input/state/output/local), so a generic tool cannot infer it on its
// own — key names it. Two shapes of that convention exist, and
// -carry-whole picks between them:
//
//   - Default (-carry-whole not given): the previous turn's
//     output[key] becomes this turn's input[key] — a same-named
//     sub-field carried forward as-is. This is the common shape: a
//     package's outputSchema separates its carried state (Navy Wars'
//     "game") from other output-only fields (its "view", its
//     "messages"), and inputSchema.key expects exactly the shape of
//     that sub-field, not a wrapper around it.
//   - -carry-whole: the previous turn's *entire* output becomes this
//     turn's input[key], unchanged. For a package whose relevant
//     state genuinely is its whole output (no separate view/messages
//     fields to leave out), or whose output isn't an object at all,
//     ask for this explicitly rather than getting it as an unstated
//     default.
//
// Navy Wars carries its state under "game", separately from "view" and
// "messages", so the default shape is what it needs:
//
//	jaxplay -carry game -format json examples/game/navywars.json
//
// plays it, in raw input/output form, through the new library — the
// intended way to verify the new pkg/jaxson against the old jaxrun.go
// on the same package and the same turns, now that both are "general
// purpose": diff this command's output against the old play.go's (or
// against a -turns run compared turn by turn) rather than eyeballing
// two differently-formatted transcripts.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	"github.com/ha1tch/jaxson/pkg/jaxtools"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("jaxplay", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text, markdown, or json")
	carryKey := fs.String("carry", "", "input key to splice carried state under (package-specific; empty disables carrying)")
	carryWhole := fs.Bool("carry-whole", false, "with -carry, splice the previous turn's whole output under key, instead of just output[key] (the default)")
	turnsPath := fs.String("turns", "", "JSON file of per-turn input objects to run non-interactively, instead of reading turns from stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Println("usage: jaxplay [-format text|markdown|json] [-carry key] [-carry-whole] [-turns turns.json] <package.json>")
		return 2
	}
	switch *format {
	case "text", "markdown", "json":
	default:
		fmt.Printf("unknown format %q (want text, markdown, or json)\n", *format)
		return 2
	}
	f := jaxtools.Format(*format)

	pkg, err := jaxtools.LoadPackage(fs.Arg(0))
	if err != nil {
		fmt.Println(err)
		return 2
	}
	sess := jaxtools.NewSession(pkg)

	if *turnsPath != "" {
		return runScripted(sess, *turnsPath, f, *carryKey, *carryWhole)
	}
	return runInteractive(sess, f, *carryKey, *carryWhole)
}

// step applies -carry (if set) and runs one turn. When whole is false
// (the default), the value spliced under carryKey is the previous
// output's own carryKey field — output[carryKey] -> input[carryKey] —
// which is what a package whose output separates carried state from
// other output-only fields (Navy Wars' "game" vs. "view"/"messages")
// needs. When whole is true, the previous turn's entire output is
// spliced under carryKey unchanged, matching this command's original
// behaviour, for a package that wants that instead. If the previous
// output isn't a map[string]any (so there's no field to pick out of
// it), whole is treated as true regardless — there is nothing else to
// carry.
func step(sess *jaxtools.Session, input map[string]any, carryKey string, whole bool) (any, *jaxson.Err) {
	if carryKey != "" {
		prev := sess.Last()
		if !whole {
			if m, ok := prev.(map[string]any); ok {
				prev = m[carryKey]
			}
		}
		input = jaxtools.Carry(input, carryKey, prev)
	}
	return sess.Step(input)
}

func printResult(turn int, out any, f jaxtools.Format) {
	rendered, err := jaxtools.Render(out, f)
	if err != nil {
		// run() already validated f against the same three formats
		// Render accepts; this is not expected to happen.
		fmt.Println(err)
		return
	}
	fmt.Printf("--- turn %d ---\n%s", turn, rendered)
}

func runScripted(sess *jaxtools.Session, path string, f jaxtools.Format, carryKey string, carryWhole bool) int {
	turns, err := jaxtools.LoadTurns(path)
	if err != nil {
		fmt.Println(err)
		return 2
	}
	for i, input := range turns {
		out, e := step(sess, input, carryKey, carryWhole)
		if e != nil {
			fmt.Printf("turn %d: %s\n", i+1, e.Error())
			return 1
		}
		printResult(i+1, out, f)
	}
	return 0
}

func runInteractive(sess *jaxtools.Session, f jaxtools.Format, carryKey string, carryWhole bool) int {
	fmt.Println("jaxplay -- type each turn's input as a JSON object, or q to quit.")
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !sc.Scan() {
			fmt.Println()
			return 0
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if line == "q" || line == "quit" {
			return 0
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(line)))
		dec.UseNumber()
		var input map[string]any
		if err := dec.Decode(&input); err != nil {
			fmt.Println("not a JSON object:", err)
			continue
		}
		input = jaxson.Normalize(input).(map[string]any)
		out, e := step(sess, input, carryKey, carryWhole)
		if e != nil {
			fmt.Println("error:", e.Error())
			continue
		}
		printResult(sess.Turn(), out, f)
	}
}
