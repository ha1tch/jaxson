// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package build_test

import (
	"encoding/json"
	"fmt"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	jb "github.com/ha1tch/jaxson/pkg/jaxson/build"
)

// A complete package in a few lines: sum the prices of an order. Decimal
// sample data is given as json.Number (or with Dec) so it stays exact.
func Example() {
	var (
		item  = jb.Loop("item")
		a, b  = jb.Var("a"), jb.Var("b")
		total = jb.State("total")
	)
	pkg := jb.NewPackage().
		InputSchema(jb.Object().Req("items", jb.Array(jb.Object().Req("price", jb.Number())))).
		OutputSchema(jb.Object().Req("total", jb.Number())).
		Input(map[string]any{"items": []any{
			map[string]any{"price": json.Number("0.10")}, map[string]any{"price": json.Number("0.20")}}}).
		Program(
			jb.Set(total, jb.Int(0)),
			jb.For(jb.Input("items")).As(item).Do(
				jb.Set(total, jb.Calc(jb.Add(a, b)).With(a, total).With(b, item.Key("price"))),
			),
			jb.Set(jb.Output(), jb.Tpl().Set("total", total)),
		)

	out, err := pkg.Run()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(jaxson.Show(out))
	// Output: {"total":0.3}
}
