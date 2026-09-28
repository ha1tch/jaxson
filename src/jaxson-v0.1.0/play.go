// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0

package main

// The play driver: `jaxrun play navywars.json [-seed N]`.
//
// The game logic lives entirely in the Jaxson package. This host loop only
// reads a move, builds the package input, calls Run, and prints the result.
// The seed is the one non-deterministic value, and it enters as explicit input.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "play" {
		os.Exit(playMain(os.Args[2:]))
	}
}

func loadPackage(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var pkg map[string]any
	if err := dec.Decode(&pkg); err != nil {
		return nil, err
	}
	norm(pkg)
	return pkg, nil
}

func printTurn(res map[string]any) {
	for _, l := range res["view"].([]any) {
		fmt.Println(l.(string))
	}
	fmt.Println()
	for _, m := range res["messages"].([]any) {
		fmt.Println("  " + m.(string))
	}
}

func playMain(args []string) int {
	path := ""
	seed := time.Now().UnixNano() % 2147483647
	for i := 0; i < len(args); i++ {
		if args[i] == "-seed" && i+1 < len(args) {
			s, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || s < 0 || s > 2147483647 {
				fmt.Println("seed must be an integer from 0 to 2147483647")
				return 2
			}
			seed = s
			i++
		} else {
			path = args[i]
		}
	}
	if path == "" {
		fmt.Println("usage: jaxrun play <package.json> [-seed N]")
		return 2
	}
	pkg, err := loadPackage(path)
	if err != nil {
		fmt.Println(err)
		return 2
	}
	run := func(input map[string]any) (map[string]any, *Err) {
		p := make(map[string]any, len(pkg))
		for k, v := range pkg {
			p[k] = v
		}
		p["input"] = input
		out, e := Run(p)
		if e != nil {
			return nil, e
		}
		return out.(map[string]any), nil
	}

	fmt.Printf("NAVY WARS   (seed %d; replay with -seed %d)\n\n", seed, seed)
	res, e := run(map[string]any{"action": "new", "seed": big.NewRat(seed, 1)})
	if e != nil {
		fmt.Println("error:", e)
		return 1
	}
	printTurn(res)

	sc := bufio.NewScanner(os.Stdin)
	for {
		game := res["game"].(map[string]any)
		if game["status"].(string) != "playing" {
			fmt.Printf("\nGame over after %s turns.\n", fmtNum(game["turn"].(*big.Rat)))
			return 0
		}
		fmt.Print("\nfire (x y, or q)> ")
		if !sc.Scan() {
			fmt.Println()
			return 0
		}
		line := strings.TrimSpace(sc.Text())
		if line == "q" || line == "quit" {
			fmt.Println("Retreating.")
			return 0
		}
		f := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == ',' })
		if len(f) != 2 {
			fmt.Println("Type two numbers, for example: 3 4")
			continue
		}
		x, e1 := strconv.Atoi(f[0])
		y, e2 := strconv.Atoi(f[1])
		if e1 != nil || e2 != nil {
			fmt.Println("Type two whole numbers, for example: 3 4")
			continue
		}
		next, e := run(map[string]any{
			"action": "fire", "x": big.NewRat(int64(x), 1), "y": big.NewRat(int64(y), 1), "game": res["game"],
		})
		if e != nil {
			if e.Cat == "EXECUTION_ERROR" && e.Code == "ASSERTION_FAILED" {
				fmt.Println("Illegal move:", e.Msg)
				continue
			}
			fmt.Println("error:", e)
			return 1
		}
		res = next
		fmt.Println()
		printTurn(res)
	}
}
