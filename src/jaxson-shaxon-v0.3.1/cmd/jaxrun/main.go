// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// jaxrun is a thin CLI over the jaxson library: it reads a package file,
// runs it, and prints the result. The fixture-suite checker that used to
// live in this binary is now jaxson's own go test (fixtures_test.go);
// this command is a runner, not a conformance check.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	"github.com/ha1tch/jaxson/pkg/jaxtools"
)

func main() {
	numeric := jaxtools.NumericFlags(flag.CommandLine)
	flag.Usage = func() {
		fmt.Println("usage: jaxrun [-strict | -relaxed] <package.json>")
		flag.PrintDefaults()
	}
	flag.Parse()
	mode, err := numeric()
	if err != nil {
		fmt.Println("jaxrun:", err)
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	raw, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	v, perr := jaxson.ParseJSONMode(raw, mode)
	if perr != nil {
		fmt.Println(perr.Error())
		os.Exit(1)
	}
	pkg, ok := v.(map[string]any)
	if !ok {
		fmt.Println("PARSE_ERROR: a package must be a JSON object")
		os.Exit(1)
	}

	prof := jaxson.CoreProfile()
	prof.Numbers = mode
	out, e := prof.Run(pkg)
	if e != nil {
		fmt.Printf("%s\n", e.Error())
		os.Exit(1)
	}
	fmt.Println(jaxson.Show(out))
}
