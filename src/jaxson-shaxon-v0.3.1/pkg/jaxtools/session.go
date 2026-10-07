// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// Package jaxtools provides host-side conveniences for embedding and
// exercising Jaxson packages: threading state across turns (Session),
// loading a package or a scripted list of turns from disk (LoadPackage,
// LoadTurns), and rendering a returned value for a developer to read
// (Render). It depends only on pkg/jaxson's exported surface and knows
// nothing about any particular package's own field conventions — Navy
// Wars' "game" key included. Where a convention is needed (which
// output field to carry into the next turn's input), the caller
// supplies it via Carry; jaxtools never guesses it.
//
// This formalises the pattern src/jaxson-v0.1.0/play.go proved by hand
// — clone the package, splice carried state into input, call Run, pull
// state back out — as a reusable type, per the implementation plan's
// §7.2 "Session helper" finding. It lives here as its own package
// rather than inside pkg/shaxon (as that plan originally sited it) so
// it isn't blocked on Shaxon's own phases and stays useful to any
// Jaxson embedder, Shaxon or not — a deliberate departure from the
// plan, recorded in TRACKER.md.
package jaxtools

import "github.com/ha1tch/jaxson/pkg/jaxson"

// Runner runs one package to completion. jaxson.Run has this shape, as
// does the Run method of any jaxson.Profile, so a session over a dialect
// of the language passes that dialect's Run to NewSessionWith.
type Runner func(map[string]any) (any, *jaxson.Err)

// Session wraps a Jaxson package and runs it turn by turn, without
// making the caller re-clone the package or re-thread errors by hand
// each time. It does not know or guess which of a package's own
// input/output fields carry state across turns — that convention
// belongs to the package, not to Jaxson or to this type. See Carry.
type Session struct {
	base    map[string]any // the package, minus "input"; never mutated after NewSession
	turn    int
	last    any // last successful output; nil before the first successful Step
	lastErr *jaxson.Err
	run     Runner
}

// NewSession starts a session over pkg. pkg must already be normalized
// (see jaxson.Normalize, or LoadPackage, which does this for you) —
// Session does not normalize it. pkg's own "input" member, if any, is
// discarded: each Step supplies its own, and Session never mutates the
// map pkg itself.
func NewSession(pkg map[string]any) *Session { return NewSessionWith(pkg, jaxson.Run) }

// NewSessionWith is NewSession over a dialect of the language: run
// executes each turn. A nil run means jaxson.Run.
func NewSessionWith(pkg map[string]any, run Runner) *Session {
	if run == nil {
		run = jaxson.Run
	}
	base := make(map[string]any, len(pkg))
	for k, v := range pkg {
		if k != "input" {
			base[k] = v
		}
	}
	return &Session{base: base, run: run}
}

// Step runs one turn: input becomes the package's "input" for this
// call only. On success, the result becomes Last() and the turn
// counter advances; on failure, Last() and Turn() are left exactly as
// they were, so a rejected turn (an illegal move, a bad input) can
// simply be retried without losing prior state.
func (s *Session) Step(input map[string]any) (any, *jaxson.Err) {
	p := make(map[string]any, len(s.base)+1)
	for k, v := range s.base {
		p[k] = v
	}
	p["input"] = input
	out, err := s.run(p)
	s.lastErr = err
	if err != nil {
		return nil, err
	}
	s.last = out
	s.turn++
	return out, nil
}

// Last returns the most recent successful output, or nil before the
// first successful Step.
func (s *Session) Last() any { return s.last }

// LastErr returns the error from the most recent Step, or nil if it
// succeeded (or none has run yet).
func (s *Session) LastErr() *jaxson.Err { return s.lastErr }

// Turn returns the number of turns successfully completed so far.
func (s *Session) Turn() int { return s.turn }

// Carry returns a shallow copy of input with prev stored under key —
// the mechanical half of the state-threading pattern: take the
// previous turn's output (or the piece of it a package carries
// forward, e.g. Navy Wars' "game") and splice it into the next turn's
// input. It is a plain map function, not a Session method, because the
// carried value need not be the whole of Last() — a caller is free to
// pass Session.Last() as a whole, or one field pulled out of it, under
// whatever key that package's inputSchema expects. If prev is nil,
// Carry still returns a copy of input, just without adding key — so a
// first turn, with nothing yet to carry, needs no special case at the
// call site.
func Carry(input map[string]any, key string, prev any) map[string]any {
	out := make(map[string]any, len(input)+1)
	for k, v := range input {
		out[k] = v
	}
	if prev != nil {
		out[key] = prev
	}
	return out
}
