// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0

// jaxrun is a thin CLI over the jaxson library: it reads a package file,
// runs it, and prints the result. The fixture-suite checker that used to
// live in this binary is now jaxson's own go test (fixtures_test.go);
// this command is a runner, not a conformance check.
package main

import (
	"fmt"
	"os"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: jaxrun <package.json>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	v, perr := jaxson.ParseJSON(raw)
	if perr != nil {
		fmt.Println(perr.Error())
		os.Exit(1)
	}
	pkg, ok := v.(map[string]any)
	if !ok {
		fmt.Println("PARSE_ERROR: a package must be a JSON object")
		os.Exit(1)
	}

	out, e := jaxson.Run(pkg)
	if e != nil {
		fmt.Printf("%s\n", e.Error())
		os.Exit(1)
	}
	fmt.Println(jaxson.Show(out))
}
