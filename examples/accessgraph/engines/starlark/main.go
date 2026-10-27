// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

// starlark_run runs the access-decision rules (rules.star) on a database and prints
// the decisions as JSON. The step count goes to stderr; an optional third argument
// is a step budget, which the interpreter enforces.
//
//	starlark_run RULES.star DB.json [MAX_STEPS]
package main

import (
	"fmt"
	"os"
	"strconv"

	"go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: starlark_run RULES.star DB.json [MAX_STEPS]")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[1])
	check(err)
	input, err := os.ReadFile(os.Args[2])
	check(err)
	thread := &starlark.Thread{Name: "rules"}
	if len(os.Args) > 3 {
		n, err := strconv.ParseUint(os.Args[3], 10, 64)
		check(err)
		thread.SetMaxExecutionSteps(n)
	}
	pre := starlark.StringDict{"json": json.Module, "INPUT": starlark.String(input)}
	// FileOptions left at their defaults: no sets, no while, no recursion, no top-level loops
	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, os.Args[1], src, pre)
	fmt.Fprintln(os.Stderr, "steps:", thread.ExecutionSteps())
	check(err)
	out, ok := globals["OUT"].(starlark.String)
	if !ok {
		check(fmt.Errorf("rules did not set OUT"))
	}
	fmt.Println(string(out))
}
