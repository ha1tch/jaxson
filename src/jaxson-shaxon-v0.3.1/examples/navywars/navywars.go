// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// Package navywars is the Navy Wars game as a Jaxson package, written
// with the jaxson/build API. It is the Go port of
// examples/game/build_navywars.py and must produce the same document as
// examples/game/navywars.json (navywars_test.go checks this).
//
// The game is a pure function: input {action, seed | x, y, game} produces
// output {game, view, messages}. The host owns the loop and the seed.
package navywars

import (
	jb "github.com/ha1tch/jaxson/pkg/jaxson/build"
)

// Fleet layouts: three ships each, cells are y*6+x. Sizes 3, 2, 2.
var layouts = [][][]int{
	{{0, 1, 2}, {11, 17}, {26, 27}},
	{{12, 18, 24}, {4, 5}, {33, 34}},
	{{19, 20, 21}, {23, 29}, {0, 1}},
	{{3, 9, 15}, {30, 31}, {23, 29}},
}

var (
	shipNames = []string{"Cruiser", "Destroyer", "Patrol boat"}
	shipHP    = []int{3, 2, 2}
)

func init() {
	for _, lay := range layouts {
		seen := map[int]bool{}
		for i, ship := range lay {
			if len(ship) != shipHP[i] {
				panic("navywars: ship size does not match its hit points")
			}
			for _, c := range ship {
				if c < 0 || c >= 36 || seen[c] {
					panic("navywars: layout cell out of range or overlapping")
				}
				seen[c] = true
			}
		}
	}
}

func fill(n, v int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func seq(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// Expression variables, declared once and reused across compute islands.
var (
	vS, vH, vX, vY, vE, vP = jb.Var("s"), jb.Var("h"), jb.Var("x"), jb.Var("y"), jb.Var("e"), jb.Var("p")
	vD, vC, vK, vR, vT, vL = jb.Var("d"), jb.Var("c"), jb.Var("k"), jb.Var("r"), jb.Var("t"), jb.Var("l")
	vN, vI, vB, vA         = jb.Var("N"), jb.Var("i"), jb.Var("b"), jb.Var("a")
	vSh, vSp, vSt, vRt     = jb.Var("sh"), jb.Var("sp"), jb.Var("st"), jb.Var("rt")
	vLayouts               = jb.Var("L")
)

// Loop bindings.
var (
	loopShip, loopCell, loopShipNo = jb.Loop("ship"), jb.Loop("c"), jb.Loop("s")
	loopK, loopRow, loopCol        = jb.Loop("k"), jb.Loop("r"), jb.Loop("c")
)

// Paths.
var (
	game  = jb.State("game")
	msgs  = jb.State("msgs")
	lines = jb.State("lines")
)

func tmp(name string) jb.Path { return jb.State("tmp", name) }

func n(i int) jb.Const { return jb.Int(i) }

// sum3 adds the three elements of the array bound to v.
func sum3(v jb.Var) jb.Expr {
	return jb.Add(jb.Get(v, n(0)), jb.Get(v, n(1)), jb.Get(v, n(2)))
}

// afloat counts the elements of the array bound to v that are above zero.
func afloat(v jb.Var) jb.Expr {
	one := func(i int) jb.Expr { return jb.Select(jb.Gt(jb.Get(v, n(i)), n(0)), n(1), n(0)) }
	return jb.Add(one(0), one(1), one(2))
}

func miss(prefix string) jb.Expr {
	return jb.Concat(jb.Str(prefix), jb.ToString(vX), jb.Str(","), jb.ToString(vY), jb.Str("): MISS."))
}

// placeFleet writes the ships of layout number layoutAt into the board.
func placeFleet(board string, layoutAt jb.Path) jb.Instr {
	layout := jb.Calc(jb.Get(vLayouts, vI)).With(vLayouts, jb.Lit(layouts)).With(vI, layoutAt)
	return jb.For(layout).As(loopShip).Index(loopShipNo).Do(
		jb.For(loopShip).As(loopCell).Do(
			jb.Set(game.Key(board).At(loopCell),
				jb.Calc(jb.Add(vS, n(1))).With(vS, loopShipNo)),
		),
	)
}

func newGame() []jb.Instr {
	return []jb.Instr{
		jb.Set(game, jb.Tpl().
			Set("seed", jb.Input("seed")).Set("turn", n(0)).Set("status", jb.Str("playing")).
			Set("eShots", jb.Data(fill(36, 0))).Set("pShots", jb.Data(fill(36, 0))).
			Set("eBoard", jb.Data(fill(36, 0))).Set("pBoard", jb.Data(fill(36, 0))).
			Set("eHP", jb.Data(shipHP)).Set("pHP", jb.Data(shipHP))),
		jb.Set(jb.State("tmp"), jb.Tpl().
			Set("el", jb.Calc(jb.Mod(vS, n(4))).With(vS, jb.Input("seed")))),
		jb.Set(tmp("pl"), jb.Calc(jb.Mod(jb.Add(vE, n(2)), n(4))).With(vE, tmp("el"))),
		placeFleet("eBoard", tmp("el")),
		placeFleet("pBoard", tmp("pl")),
		jb.Append(msgs, jb.Str("New game. Sink the enemy fleet before it sinks yours.")),
		jb.Append(msgs, jb.Str("Fire with: x y   (both 0 to 5)")),
	}
}

func playerFires() []jb.Instr {
	idx, ship := tmp("idx"), tmp("ship")
	shipIdx := jb.Calc(jb.Sub(vS, n(1))).With(vS, ship)
	hpNow := game.Key("eHP").At(shipIdx)
	hit := jb.Concat(
		jb.Str("You fire at ("), jb.ToString(vX), jb.Str(","), jb.ToString(vY), jb.Str("): "),
		jb.Select(jb.Eq(vH, n(0)),
			jb.Concat(jb.Str("HIT! You sank the "), jb.Get(vN, jb.Sub(vS, n(1))), jb.Str("!")),
			jb.Str("HIT!")))
	return []jb.Instr{
		jb.Set(tmp("ship"), jb.Calc(jb.Get(vB, vI)).With(vB, game.Key("eBoard")).With(vI, idx)),
		jb.Set(game.Key("eShots").At(idx), n(1)),
		jb.If(jb.Calc(jb.Gt(vS, n(0))).With(vS, ship)).Then(
			jb.Set(game.Key("eHP").At(shipIdx), jb.Calc(jb.Sub(vH, n(1))).With(vH, hpNow)),
			jb.Append(msgs, jb.Calc(hit).
				With(vX, tmp("x")).With(vY, tmp("y")).With(vH, hpNow).With(vS, ship).With(vN, jb.Lit(shipNames))),
		).Else(
			jb.Append(msgs, jb.Calc(miss("You fire at (")).With(vX, tmp("x")).With(vY, tmp("y"))),
		),
		jb.Set(tmp("eDead"), jb.Calc(jb.Eq(sum3(vH), n(0))).With(vH, game.Key("eHP"))),
	}
}

func enemyTurn() []jb.Instr {
	cidx, eship := tmp("c"), tmp("es")
	esIdx := jb.Calc(jb.Sub(vS, n(1))).With(vS, eship)
	hpNow := game.Key("pHP").At(esIdx)
	hit := jb.Concat(
		jb.Str("Enemy fires at ("), jb.ToString(vX), jb.Str(","), jb.ToString(vY), jb.Str("): HIT! Your "),
		jb.Get(vN, jb.Sub(vS, n(1))),
		jb.Select(jb.Eq(vH, n(0)), jb.Str(" was sunk!"), jb.Str(" was hit!")))
	return []jb.Instr{
		jb.Set(game.Key("seed"), jb.Calc(jb.Mod(jb.Add(jb.Mul(vS, n(1103515245)), n(12345)), n(2147483648))).
			With(vS, game.Key("seed"))),
		jb.Set(tmp("start"), jb.Calc(jb.Mod(vS, n(36))).With(vS, game.Key("seed"))),
		jb.Set(tmp("done"), jb.Bool(false)),
		jb.For(jb.Data(seq(36))).As(loopK).Do(
			jb.Set(tmp("c"), jb.Calc(jb.Mod(jb.Add(vS, vK), n(36))).With(vS, tmp("start")).With(vK, loopK)),
			jb.If(jb.Calc(jb.And(jb.Not(vD), jb.Eq(vSh, n(0)))).
				With(vD, tmp("done")).With(vSh, game.Key("pShots").At(cidx))).Then(
				jb.Set(tmp("done"), jb.Bool(true)),
				jb.Set(game.Key("pShots").At(cidx), n(1)),
				jb.Set(tmp("es"), jb.Calc(jb.Get(vB, vI)).With(vB, game.Key("pBoard")).With(vI, cidx)),
				jb.Set(tmp("ex"), jb.Calc(jb.Mod(vC, n(6))).With(vC, cidx)),
				jb.Set(tmp("ey"), jb.Calc(jb.Div(vC, n(6), 0, jb.Down)).With(vC, cidx)),
				jb.If(jb.Calc(jb.Gt(vS, n(0))).With(vS, eship)).Then(
					jb.Set(game.Key("pHP").At(esIdx), jb.Calc(jb.Sub(vH, n(1))).With(vH, hpNow)),
					jb.Append(msgs, jb.Calc(hit).
						With(vX, tmp("ex")).With(vY, tmp("ey")).With(vS, eship).With(vH, hpNow).With(vN, jb.Lit(shipNames))),
				).Else(
					jb.Append(msgs, jb.Calc(miss("Enemy fires at (")).With(vX, tmp("ex")).With(vY, tmp("ey"))),
				),
			),
		),
		jb.Set(tmp("pDead"), jb.Calc(jb.Eq(sum3(vH), n(0))).With(vH, game.Key("pHP"))),
		jb.If(tmp("pDead")).Then(
			jb.Set(game.Key("status"), jb.Str("lost")),
			jb.Append(msgs, jb.Str("DEFEAT! Your fleet has been sunk.")),
		),
	}
}

func fire() []jb.Instr {
	out := []jb.Instr{
		jb.Set(game, jb.Input("game")),
		jb.Assert(jb.Calc(jb.Eq(vS, jb.Str("playing"))).With(vS, game.Key("status"))).Msg("The game is over."),
		jb.Set(jb.State("tmp"), jb.Tpl().Set("x", jb.Input("x")).Set("y", jb.Input("y"))),
		jb.Assert(jb.Calc(jb.And(jb.Ge(vX, n(0)), jb.Le(vX, n(5)), jb.Ge(vY, n(0)), jb.Le(vY, n(5)))).
			With(vX, tmp("x")).With(vY, tmp("y"))).Msg("Off the board: x and y must be between 0 and 5."),
		jb.Set(tmp("idx"), jb.Calc(jb.Add(jb.Mul(vY, n(6)), vX)).With(vX, tmp("x")).With(vY, tmp("y"))),
		jb.Assert(jb.Calc(jb.Eq(vS, n(0))).With(vS, game.Key("eShots").At(tmp("idx")))).Msg("You already fired there."),
	}
	out = append(out, playerFires()...)
	return append(out,
		jb.If(tmp("eDead")).Then(
			jb.Set(game.Key("status"), jb.Str("won")),
			jb.Append(msgs, jb.Str("VICTORY! You sank the entire enemy fleet.")),
		).Else(enemyTurn()...),
		jb.Set(game.Key("turn"), jb.Calc(jb.Add(vT, n(1))).With(vT, game.Key("turn"))),
	)
}

// cell is the character drawn for one square. revealShips shows unshot
// enemy ships once the game is over.
func cell(revealShips bool) jb.Expr {
	shot := jb.Eq(vSh, n(0))
	ship := jb.Gt(vSp, n(0))
	unshot := jb.Select(ship, jb.Str("#"), jb.Str("~"))
	if revealShips {
		unshot = jb.Select(jb.And(jb.Ne(vSt, jb.Str("playing")), ship), jb.Str("#"), jb.Str("~"))
	}
	return jb.Select(shot, unshot, jb.Select(ship, jb.Str("X"), jb.Str("o")))
}

func render() []jb.Instr {
	left, right, ti := tmp("left"), tmp("right"), tmp("i")
	return []jb.Instr{
		jb.Append(lines, jb.Str("      ENEMY WATERS              YOUR FLEET")),
		jb.Append(lines, jb.Str("    0 1 2 3 4 5          0 1 2 3 4 5")),
		jb.For(jb.Data(seq(6))).As(loopRow).Do(
			jb.Set(left, jb.Str("")),
			jb.Set(right, jb.Str("")),
			jb.For(jb.Data(seq(6))).As(loopCol).Do(
				jb.Set(tmp("i"), jb.Calc(jb.Add(jb.Mul(vR, n(6)), vC)).With(vR, loopRow).With(vC, loopCol)),
				jb.Set(left, jb.Calc(jb.Concat(vL, cell(true), jb.Str(" "))).
					With(vL, left).With(vSh, game.Key("eShots").At(ti)).With(vSp, game.Key("eBoard").At(ti)).
					With(vSt, game.Key("status"))),
				jb.Set(right, jb.Calc(jb.Concat(vL, cell(false), jb.Str(" "))).
					With(vL, right).With(vSh, game.Key("pShots").At(ti)).With(vSp, game.Key("pBoard").At(ti))),
			),
			jb.Append(lines, jb.Calc(jb.Concat(jb.ToString(vR), jb.Str("   "), vL, jb.Str("        "),
				jb.ToString(vR), jb.Str("   "), vRt)).
				With(vR, loopRow).With(vL, left).With(vRt, right)),
		),
		jb.Append(lines, jb.Calc(jb.Concat(jb.Str("Ships afloat - enemy: "), jb.ToString(afloat(vE)),
			jb.Str("   you: "), jb.ToString(afloat(vP)), jb.Str("   turn: "), jb.ToString(vT))).
			With(vE, game.Key("eHP")).With(vP, game.Key("pHP")).With(vT, game.Key("turn"))),
	}
}

func cells(lo, hi int) jb.ArraySchema { return jb.Array(jb.Integer().Between(lo, hi)).Len(36) }

func gameSchema() jb.ObjectSchema {
	hp := jb.Array(jb.Integer().Between(0, 3)).Len(3)
	return jb.Object().
		Req("seed", jb.Integer().Between(0, 2147483647)).
		Req("turn", jb.Integer().Min(0)).
		Req("status", jb.String().Enum("playing", "won", "lost")).
		Req("eShots", cells(0, 1)).Req("pShots", cells(0, 1)).
		Req("eBoard", cells(0, 3)).Req("pBoard", cells(0, 3)).
		Req("eHP", hp).Req("pHP", hp)
}

// Package returns the complete Navy Wars package, with its sample input
// (start a new game with seed 0).
func Package() jb.Package {
	prog := []jb.Instr{
		jb.Set(msgs, jb.Data([]any{})),
		jb.Set(lines, jb.Data([]any{})),
		jb.If(jb.Calc(jb.Eq(vA, jb.Str("new"))).With(vA, jb.Input("action"))).
			Then(newGame()...).Else(fire()...),
	}
	prog = append(prog, render()...)
	prog = append(prog, jb.Set(jb.Output(), jb.Tpl().
		Set("game", game).Set("view", lines).Set("messages", msgs)))

	return jb.NewPackage().
		Steps(20000).
		Input(map[string]any{"action": "new", "seed": 0}).
		InputSchema(jb.Object().
			Req("action", jb.String().Enum("new", "fire")).
			Field("seed", jb.Integer().Between(0, 2147483647)).
			Field("x", jb.Integer()).Field("y", jb.Integer()).
			Field("game", gameSchema())).
		OutputSchema(jb.Object().
			Req("game", gameSchema()).
			Req("view", jb.Array(jb.String())).
			Req("messages", jb.Array(jb.String()))).
		Program(prog...)
}
