// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

// agogo is the Go-only accessgraph pipeline: generate a database, evaluate it
// with the reference and the optimized oracle, check that they agree, and time
// them. Nothing here needs Python.
//
//	agogo gen   [-seed N] [-size S] [-nocycles]      database JSON to stdout
//	agogo eval  [-impl fast|ref] [-hops N] db.json   decisions JSON to stdout
//	agogo check [-n N]                               reference vs optimized
//	agogo bench [-sizes a,b,c]                       generate, evaluate, time
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

func die(code int, args ...any) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(code)
}

func loadDB(path string) *DB {
	raw, err := os.ReadFile(path)
	if err != nil {
		die(2, err)
	}
	var db DB
	if err := json.Unmarshal(raw, &db); err != nil {
		die(3, "parse:", err)
	}
	return &db
}

// sameResults reports whether the optimized results equal the reference's.
func sameResults(db *DB, ref map[string]Result, fast []Result) bool {
	if len(ref) != len(fast) {
		return false
	}
	for i, q := range db.Requests {
		if !reflect.DeepEqual(ref[q.ID], fast[i]) {
			return false
		}
	}
	return true
}

func main() {
	if len(os.Args) < 2 {
		die(2, "usage: agogo gen|eval|check|bench [flags]")
	}
	cmd, args := os.Args[1], os.Args[2:]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	switch cmd {
	case "gen":
		seed := fs.Uint64("seed", 0, "random seed")
		size := fs.String("size", "small", strings.Join(PresetOrder, "|"))
		nocycles := fs.Bool("nocycles", false, "omit the planted membership cycles")
		fs.Parse(args)
		db, err := Generate(*seed, *size, !*nocycles)
		if err != nil {
			die(2, err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(db); err != nil {
			die(5, err)
		}

	case "eval":
		impl := fs.String("impl", "fast", "fast|ref")
		hops := fs.Int("hops", MaxHops, "maximum delegation chain length")
		repeat := fs.Int("repeat", 0, "if > 0, time this many runs and print the best to stderr")
		fs.Parse(args)
		if fs.NArg() != 1 {
			die(2, "usage: agogo eval [-impl fast|ref] [-hops N] [-repeat N] db.json")
		}
		db := loadDB(fs.Arg(0))
		if *repeat > 0 {
			best := time.Duration(1 << 62)
			for i := 0; i < *repeat; i++ {
				t := time.Now()
				if *impl == "ref" {
					EvalRef(db, *hops)
				} else {
					EvalFast(db, *hops)
				}
				if d := time.Since(t); d < best {
					best = d
				}
			}
			fmt.Fprintf(os.Stderr, "evaluate: %.3f ms (best of %d)\n", float64(best.Microseconds())/1000, *repeat)
		}
		var out map[string]Result
		if *impl == "ref" {
			var err error
			if out, err = EvalRef(db, *hops); err != nil {
				die(4, err)
			}
		} else {
			res, err := EvalFast(db, *hops)
			if err != nil {
				die(4, err)
			}
			out = make(map[string]Result, len(res))
			for i, q := range db.Requests {
				out[q.ID] = res[i]
			}
		}
		if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
			die(5, err)
		}

	case "check":
		n := fs.Int("n", 60, "number of databases")
		fs.Parse(args)
		sizes := []string{"tiny", "small", "small", "medium"}
		bad, decisions := 0, 0
		for seed := 0; seed < *n; seed++ {
			db, err := Generate(uint64(seed), sizes[seed%len(sizes)], seed%7 != 0)
			if err != nil {
				die(2, err)
			}
			for hops := 0; hops <= MaxHops+1; hops++ {
				ref, err1 := EvalRef(db, hops)
				fast, err2 := EvalFast(db, hops)
				if err1 != nil || err2 != nil {
					die(4, "evaluate:", err1, err2)
				}
				decisions += len(fast)
				if !sameResults(db, ref, fast) {
					bad++
					fmt.Printf("MISMATCH seed=%d size=%s hops=%d\n", seed, db.Meta.Size, hops)
				}
			}
		}
		fmt.Printf("check: %d databases x %d hop limits, %d decisions, %d mismatches\n",
			*n, MaxHops+2, decisions, bad)
		if bad > 0 {
			os.Exit(1)
		}

	case "bench":
		sizes := fs.String("sizes", strings.Join(PresetOrder, ","), "comma-separated presets")
		fs.Parse(args)
		fmt.Printf("%-8s %7s %9s %12s %12s %9s %12s %12s\n", "size", "users", "gen", "ref", "fast", "speedup", "ref alloc", "fast alloc")
		for _, size := range strings.Split(*sizes, ",") {
			t := time.Now()
			db, err := Generate(0, size, true)
			if err != nil {
				die(2, err)
			}
			gen := time.Since(t)
			ref, refT, refA := measure(func() any { r, _ := EvalRef(db, MaxHops); return r })
			fast, fastT, fastA := measure(func() any { r, _ := EvalFast(db, MaxHops); return r })
			if !sameResults(db, ref.(map[string]Result), fast.([]Result)) {
				die(1, "MISMATCH at size", size)
			}
			fmt.Printf("%-8s %7d %9s %12s %12s %8.1fx %9.1f MB %9.1f MB\n", size, len(db.Users),
				gen.Round(time.Microsecond), refT.Round(time.Microsecond), fastT.Round(time.Microsecond),
				float64(refT)/float64(fastT), float64(refA)/1e6, float64(fastA)/1e6)
		}

	default:
		die(2, "unknown command", cmd)
	}
}

// measure runs fn enough times for a stable figure and returns its last
// result, the best time of a run, and the bytes allocated by one run.
func measure(fn func() any) (any, time.Duration, uint64) {
	var res any
	best := time.Duration(1 << 62)
	var alloc uint64
	deadline := time.Now().Add(400 * time.Millisecond)
	for i := 0; i < 3 || time.Now().Before(deadline); i++ {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		t := time.Now()
		res = fn()
		d := time.Since(t)
		runtime.ReadMemStats(&after)
		if d < best {
			best, alloc = d, after.TotalAlloc-before.TotalAlloc
		}
		if i >= 40 {
			break
		}
	}
	return res, best, alloc
}
