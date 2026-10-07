// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

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
