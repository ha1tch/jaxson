// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0

// Command navywars writes the Navy Wars package as JSON to standard output
// (or to the file named by its only argument).
package main

import (
	"fmt"
	"os"

	"github.com/ha1tch/jaxson/examples/navywars"
)

func main() {
	out, err := navywars.Package().JSON()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		if err := os.WriteFile(os.Args[1], out, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Stdout.Write(out)
}
