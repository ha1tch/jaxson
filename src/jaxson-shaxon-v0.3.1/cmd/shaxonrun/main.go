// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// shaxonrun runs a Shaxon package: it reads the package file, optionally
// replaces its "input" with another JSON file, runs the whole pipeline of
// core section 9, and prints the output and any combined validation report
// as one JSON object:
//
//	{"output": ..., "report": {"conforms": true, "violations": []}}
//
// "report" is present only when a report-mode entry without `into` ran. On
// a failure it prints the error and exits 1; on a usage or read error, 2.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	"github.com/ha1tch/jaxson/pkg/shaxon"
)

func main() {
	pretty := flag.Bool("pretty", false, "indent the result")
	steps := flag.Bool("steps", false, "print the step count to standard error")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: shaxonrun [-pretty] [-steps] <package.json> [input.json]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 || flag.NArg() > 2 {
		flag.Usage()
		os.Exit(2)
	}
	raw, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	v, perr := jaxson.ParsePackage(raw)
	if perr != nil {
		fmt.Fprintln(os.Stderr, perr.Error())
		os.Exit(1)
	}
	pkg, ok := v.(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, "PARSE_ERROR: a package must be a JSON object")
		os.Exit(1)
	}
	if flag.NArg() == 2 {
		in, err := os.ReadFile(flag.Arg(1))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		iv, perr := jaxson.ParseData(in)
		if perr != nil {
			fmt.Fprintln(os.Stderr, "input:", perr.Error())
			os.Exit(1)
		}
		pkg["input"] = iv
	}

	res, rerr := shaxon.Run(pkg)
	if rerr != nil {
		fmt.Fprintln(os.Stderr, rerr.Error())
		os.Exit(1)
	}
	out := map[string]any{"output": res.Output}
	if res.Report != nil {
		out["report"] = res.Report.Value()
	}
	text := jaxson.Show(out)
	if *pretty {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(text), "", "  "); err == nil {
			text = buf.String()
		}
	}
	fmt.Println(text)
	if *steps {
		fmt.Fprintf(os.Stderr, "steps: %d\n", res.Steps)
	}
}
