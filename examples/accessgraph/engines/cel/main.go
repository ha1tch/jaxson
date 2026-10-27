// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

// celprobe asks what CEL's cost estimate says about a transitive closure.
//
// CEL is an expression language with a static cost estimate (from declared size
// bounds) and a runtime cost count. The access-decision task needs a closure over
// group membership, which CEL can only express as a fixed number of unrolled
// rounds, like the Jaxson program. For k = 1..8 rounds this builds the expression,
// takes CEL's estimate with the bounds the data actually has (the tightest honest
// declaration), runs it for the user with the largest reach, and prints the
// estimate beside the cost CEL counted.
//
//	celprobe DB.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/ext"
)

type database struct {
	Users   []string `json:"users"`
	Members []struct {
		Member string `json:"member"`
		Group  string `json:"group"`
	} `json:"members"`
}

// bounds declares the sizes the estimate may assume, by type: strings of at most
// strMax characters, lists of at most listMax elements, and a map of mapSize entries.
// These are the maxima the data actually has, the tightest honest declaration.
type bounds struct{ mapSize, listMax, strMax uint64 }

func (b bounds) EstimateSize(el checker.AstNode) *checker.SizeEstimate {
	switch el.Type().Kind() {
	case types.StringKind:
		return &checker.SizeEstimate{Min: 1, Max: b.strMax}
	case types.ListKind:
		return &checker.SizeEstimate{Min: 0, Max: b.listMax}
	case types.MapKind:
		return &checker.SizeEstimate{Min: 0, Max: b.mapSize}
	}
	return nil
}

func (b bounds) EstimateCallCost(function, overloadID string, target *checker.AstNode, args []checker.AstNode) *checker.CallEstimate {
	return nil
}

func closureExpr(k int) string {
	var s strings.Builder
	prev := "[u]"
	for i := 1; i <= k; i++ {
		cur := fmt.Sprintf("r%d", i)
		fmt.Fprintf(&s, "cel.bind(%s, (%s + %s.map(x, (x in up) ? up[x] : []).flatten()).distinct(), ", cur, prev, prev)
		prev = cur
	}
	fmt.Fprintf(&s, "size(%s)%s", prev, strings.Repeat(")", k))
	return s.String()
}

// reachExpr asks, with core macros only, whether t is reachable from x within k rounds.
func reachExpr(k int, x string, depth int) string {
	if k == 0 {
		return x + " == t"
	}
	y := fmt.Sprintf("y%d", depth)
	return fmt.Sprintf("(%s == t || ((%s in up) && up[%s].exists(%s, %s)))", x, x, x, y, reachExpr(k-1, y, depth+1))
}

func reach(up map[string][]string, u string) int {
	seen := map[string]bool{u: true}
	stack := []string{u}
	for len(stack) > 0 {
		x := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, y := range up[x] {
			if !seen[y] {
				seen[y] = true
				stack = append(stack, y)
			}
		}
	}
	return len(seen)
}

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var db database
	if err := json.Unmarshal(raw, &db); err != nil {
		panic(err)
	}
	up := map[string][]string{}
	for _, m := range db.Members {
		up[m.Member] = append(up[m.Member], m.Group)
	}
	var listMax uint64
	for _, v := range up {
		if uint64(len(v)) > listMax {
			listMax = uint64(len(v))
		}
	}
	sort.Strings(db.Users)
	heavy, heavyReach := "", 0
	for _, u := range db.Users {
		if r := reach(up, u); r > heavyReach {
			heavy, heavyReach = u, r
		}
	}
	fmt.Printf("database: %d users, %d membership edges; declared bounds: map of %d entries, lists of at most %d, strings of at most 8\n", len(db.Users), len(db.Members), len(up), listMax)
	fmt.Printf("heaviest user: %s, true reach %d (including itself)\n\n", heavy, heavyReach)

	env, err := cel.NewEnv(
		cel.Variable("up", cel.MapType(cel.StringType, cel.ListType(cel.StringType))),
		cel.Variable("u", cel.StringType),
		cel.Variable("t", cel.StringType),
		ext.Lists(), ext.Bindings(),
	)
	if err != nil {
		panic(err)
	}
	b := bounds{mapSize: uint64(len(up)), listMax: listMax, strMax: 8}
	if len(os.Args) > 2 && os.Args[2] == "parts" {
		// which piece of the closure makes the estimate unbounded?
		for _, e := range []string{
			`(u in up) ? up[u] : []`,
			`[u].map(x, (x in up) ? up[x] : [])`,
			`[u].map(x, (x in up) ? size(up[x]) : 0)`,
			`[u].map(x, (x in up) ? up[x] : []).flatten()`,
			`([u] + [u]).distinct()`,
			`cel.bind(r, [u], size(r))`,
			`up[u].exists(g, g == "g00001")`,
			`up.exists(k, up[k].exists(g, g == u))`,
		} {
			ast, iss := env.Compile(e)
			if iss.Err() != nil {
				fmt.Printf("%-62s compile error\n", e)
				continue
			}
			est, err := env.EstimateCost(ast, b)
			if err != nil {
				fmt.Printf("%-62s estimate error: %v\n", e, err)
				continue
			}
			fmt.Printf("%-62s %d .. %d\n", e, est.Min, est.Max)
		}
		return
	}
	fmt.Printf("%-7s %-34s %-12s %-12s\n", "rounds", "estimated cost (min .. max)", "counted", "reach found")
	for _, k := range []int{1, 2, 3, 4, 5, 6, 8, 16} {
		ast, iss := env.Compile(closureExpr(k))
		if iss.Err() != nil {
			fmt.Printf("%-7d compile error: %v\n", k, iss.Err())
			continue
		}
		est, err := env.EstimateCost(ast, b)
		if err != nil {
			fmt.Printf("%-7d estimate error: %v\n", k, err)
			continue
		}
		prg, err := env.Program(ast, cel.CostTracking(nil))
		if err != nil {
			fmt.Printf("%-7d program error: %v\n", k, err)
			continue
		}
		out, det, err := prg.Eval(map[string]any{"up": up, "u": heavy})
		if err != nil {
			fmt.Printf("%-7d %-34s eval error: %v\n", k, fmt.Sprintf("%d .. %d", est.Min, est.Max), err)
			continue
		}
		fmt.Printf("%-7d %-34s %-12d %-12v\n", k, fmt.Sprintf("%d .. %d", est.Min, est.Max), *det.ActualCost(), out.Value())
	}

	// the same closure, asked with core macros only: is a subject outside the user's reach
	// reachable within k rounds? The target is absent, so every path is walked.
	fmt.Printf("\ncore macros only: is a subject outside the reach reachable within k rounds?\n")
	fmt.Printf("%-7s %-40s %-12s %-8s\n", "rounds", "estimated cost (min .. max)", "counted", "answer")
	for _, k := range []int{1, 2, 3, 4, 5, 6, 8} {
		ast, iss := env.Compile(reachExpr(k, "u", 1))
		if iss.Err() != nil {
			fmt.Printf("%-7d compile error: %v\n", k, iss.Err())
			continue
		}
		est, err := env.EstimateCost(ast, b)
		if err != nil {
			fmt.Printf("%-7d estimate error: %v\n", k, err)
			continue
		}
		prg, err := env.Program(ast, cel.CostTracking(nil))
		if err != nil {
			fmt.Printf("%-7d program error: %v\n", k, err)
			continue
		}
		out, det, err := prg.Eval(map[string]any{"up": up, "u": heavy, "t": "no-such-subject"})
		if err != nil {
			fmt.Printf("%-7d eval error: %v\n", k, err)
			continue
		}
		fmt.Printf("%-7d %-40s %-12d %-8v\n", k, fmt.Sprintf("%d .. %d", est.Min, est.Max), *det.ActualCost(), out.Value())
	}
}
