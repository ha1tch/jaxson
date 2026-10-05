// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Benchmarks for the engine alone (the package and input are decoded before
// the timer starts). They use two of the authorisation packages on long
// trails: rolling-quota (a fold over every event) and chinese-wall (indices
// with computed keys over every event). The inputs are generated here with
// a fixed pseudo-random sequence, so a run can be repeated; they are not the
// same events as examples/shacl/authz/scale.py makes.
//
//	go test -run '^$' -bench Engine -benchmem ./pkg/shaxon/
//	SHAXON_BENCH_TREE=1 go test ...   (the same on the tree-walking oracle)

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func benchPackage(b *testing.B, name string, inputJSON string, raiseSteps bool) map[string]any {
	b.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "shaxon", "authz", name+".json"))
	if err != nil {
		b.Fatal(err)
	}
	v, perr := jaxson.ParseJSON(raw)
	if perr != nil {
		b.Fatal(perr)
	}
	pkg := v.(map[string]any)
	// The input is parsed into the machine's own representation, as RunJSON
	// does; SHAXON_BENCH_MAPINPUT=1 keeps it as maps, so Run converts it
	// every time (the cost of handing Run a pre-parsed map).
	parseInput := jaxson.ParseData
	if os.Getenv("SHAXON_BENCH_MAPINPUT") != "" {
		parseInput = jaxson.ParseJSON
	}
	in, perr := parseInput([]byte(inputJSON))
	if perr != nil {
		b.Fatal(perr)
	}
	pkg["input"] = in
	if raiseSteps {
		pkg["limits"] = map[string]any{"steps": big.NewRat(1000000000000, 1)}
	}
	return pkg
}

func quotaInputJSON(n int) string {
	actors := n / 6
	if actors < 1 {
		actors = 1
	}
	var sb strings.Builder
	sb.WriteString(`{"policy":{"windowHours":24,"limitMb":0.9,"maxExports":5},"events":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"t":%.4f,"actor":"u%d","mb":0.0001}`, float64(i)*20/float64(n), i%actors)
	}
	sb.WriteString(`],"request":{"actor":"u1","t":20.5,"mb":0.1}}`)
	return sb.String()
}

func wallInputJSON(n int) string {
	var sb strings.Builder
	sb.WriteString(`{"classOf":{`)
	for i := 0; i < 200; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"c%d":"k%d"`, i, i%40)
	}
	sb.WriteString(`},"events":[`)
	seed := uint32(7)
	next := func(m uint32) uint32 { seed = seed*1664525 + 1013904223; return (seed >> 8) % m }
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"actor":"u%d","company":"c%d"}`, next(100), next(200))
	}
	sb.WriteString(`],"request":{"actor":"u1","company":"c1"}}`)
	return sb.String()
}

func runBench(b *testing.B, pkg map[string]any) {
	b.Helper()
	// SHAXON_BENCH_TREE=1 runs the same benchmark on the tree-walking oracle
	// (scheduled for deletion), for a same-binary comparison with the compiler.
	if os.Getenv("SHAXON_BENCH_TREE") != "" {
		jaxson.UseTreeInterpreter(true)
		defer jaxson.UseTreeInterpreter(false)
	}
	// The first run also compiles the package's programs to closures (the
	// "lift"); it is timed on its own and left out of the measured loop.
	t0 := time.Now()
	if _, err := Run(pkg); err != nil {
		b.Fatal(err)
	}
	first := time.Since(t0)
	b.ResetTimer()
	b.ReportAllocs()
	var steps int64
	for i := 0; i < b.N; i++ {
		res, err := Run(pkg)
		if err != nil {
			b.Fatal(err)
		}
		steps = res.Steps
	}
	b.ReportMetric(float64(steps), "steps")
	b.ReportMetric(float64(first.Microseconds())/1000, "first-ms")
}

func BenchmarkEngineRollingQuota4000(b *testing.B) {
	runBench(b, benchPackage(b, "rolling-quota", quotaInputJSON(4000), true))
}
func BenchmarkEngineRollingQuota20000(b *testing.B) {
	runBench(b, benchPackage(b, "rolling-quota", quotaInputJSON(20000), true))
}
func BenchmarkEngineChineseWall4000(b *testing.B) {
	runBench(b, benchPackage(b, "chinese-wall", wallInputJSON(4000), false))
}
func BenchmarkEngineChineseWall20000(b *testing.B) {
	runBench(b, benchPackage(b, "chinese-wall", wallInputJSON(20000), false))
}
