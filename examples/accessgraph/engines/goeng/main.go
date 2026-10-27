// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"swdunlop.dev/pkg/datalog"
	"swdunlop.dev/pkg/datalog/seminaive"
	"swdunlop.dev/pkg/datalog/syntax"
)

func val(c datalog.Constant) string {
	switch v := c.(type) {
	case datalog.Integer:
		return fmt.Sprintf("%d", int64(v))
	case datalog.String:
		return string(v)
	}
	return fmt.Sprint(c)
}

func main() {
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	tr, err := syntax.Parse(seminaive.New(), string(src))
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(3)
	}
	out, err := tr.Transform(context.Background(), datalog.Empty{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "transform:", err)
		os.Exit(4)
	}
	for _, p := range []struct {
		name  string
		arity int
	}{{"via", 2}, {"den", 2}} {
		for row := range out.Facts(p.name, p.arity) {
			fmt.Printf("%s\t%s\t%s\n", p.name, val(row[0]), val(row[1]))
		}
	}
}
