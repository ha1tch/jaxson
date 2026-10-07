// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// enginetime times the Shaxon engine alone, for scale.py's engine-only
// profile. It reads a package and an input, parses both once, then runs the
// package in-process N+1 times and prints the best run as one JSON object.
// The first run is a warm-up: it also compiles the package's programs to
// closures (the engine's own "lift"), which is therefore not in the figure:
//
//	{"ms": 1.2345, "steps": 473, "output": {...}}
//
// What is inside the timer is shaxon.Run on an already-decoded package. What
// is outside it: starting the process, reading the files and decoding the
// JSON (the input into jaxson.Object form, as shaxon.RunJSON does). Shaxon
// has no counterpart of the RDF lift, because it reads the JSON tree as it is.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	"github.com/ha1tch/jaxson/pkg/shaxon"
)

func load(path string, parse func([]byte) (any, *jaxson.Err)) any {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "enginetime:", err)
		os.Exit(2)
	}
	v, perr := parse(raw)
	if perr != nil {
		fmt.Fprintln(os.Stderr, "enginetime:", path, perr)
		os.Exit(2)
	}
	return v
}

func main() {
	reps := flag.Int("n", 10, "timed runs after one warm-up run")
	prof := flag.String("cpuprofile", "", "write a CPU profile of the timed runs to this file")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: enginetime [-n N] <package.json> <input.json>")
		os.Exit(2)
	}
	pkg, ok := load(flag.Arg(0), jaxson.ParseJSON).(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, "enginetime: a package must be a JSON object")
		os.Exit(2)
	}
	pkg["input"] = load(flag.Arg(1), jaxson.ParseData) // decoded into the engine's own form

	var best time.Duration
	var res shaxon.Result
	for i := 0; i <= *reps; i++ {
		if i == 1 && *prof != "" { // after the warm-up run
			f, err := os.Create(*prof)
			if err != nil {
				fmt.Fprintln(os.Stderr, "enginetime:", err)
				os.Exit(2)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		t0 := time.Now()
		r, err := shaxon.Run(pkg)
		d := time.Since(t0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "enginetime:", err)
			os.Exit(1)
		}
		res = r
		if i > 0 && (best == 0 || d < best) {
			best = d
		}
	}
	fmt.Printf("{\"ms\":%.4f,\"steps\":%d,\"output\":%s}\n", float64(best.Nanoseconds())/1e6, res.Steps, jaxson.Show(res.Output))
}
