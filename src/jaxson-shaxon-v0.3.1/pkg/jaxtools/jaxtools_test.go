// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxtools

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// echoPackage is the smallest package that lets Session tests exercise
// success, failure, and state carried by hand: output is just
// input.n, and input.n is required, so an empty input reliably fails
// with INPUT_ERROR without needing any real program logic.
func echoPackage() map[string]any {
	return map[string]any{
		"jaxson": "1.0",
		"inputSchema": map[string]any{
			"type":     "object",
			"fields":   map[string]any{"n": map[string]any{"type": "number"}},
			"required": []any{"n"},
		},
		"program": []any{
			map[string]any{"op": "set", "path": []any{"output"}, "value": map[string]any{"$path": []any{"input", "n"}}},
		},
		"outputSchema": map[string]any{"type": "number"},
	}
}

func rat(n int64) *big.Rat { return big.NewRat(n, 1) }

func TestSessionStepAdvancesOnSuccess(t *testing.T) {
	s := NewSession(echoPackage())
	if s.Turn() != 0 || s.Last() != nil {
		t.Fatalf("new session: got turn=%d last=%v, want 0/nil", s.Turn(), s.Last())
	}
	out, err := s.Step(map[string]any{"n": rat(1)})
	if err != nil {
		t.Fatalf("turn 1: unexpected error %v", err)
	}
	if r, ok := out.(*big.Rat); !ok || r.Cmp(rat(1)) != 0 {
		t.Fatalf("turn 1: got output %v, want 1", out)
	}
	if s.Turn() != 1 {
		t.Fatalf("turn 1: got Turn()=%d, want 1", s.Turn())
	}
	if r, ok := s.Last().(*big.Rat); !ok || r.Cmp(rat(1)) != 0 {
		t.Fatalf("turn 1: got Last()=%v, want 1", s.Last())
	}

	out, err = s.Step(map[string]any{"n": rat(2)})
	if err != nil {
		t.Fatalf("turn 2: unexpected error %v", err)
	}
	if r := out.(*big.Rat); r.Cmp(rat(2)) != 0 {
		t.Fatalf("turn 2: got output %v, want 2", out)
	}
	if s.Turn() != 2 {
		t.Fatalf("turn 2: got Turn()=%d, want 2", s.Turn())
	}
}

func TestSessionStepHoldsStateOnError(t *testing.T) {
	s := NewSession(echoPackage())
	if _, err := s.Step(map[string]any{"n": rat(7)}); err != nil {
		t.Fatalf("setup turn: unexpected error %v", err)
	}

	// Missing "n" violates inputSchema's required list -> INPUT_ERROR.
	// Last()/Turn() must be exactly what the prior successful turn left,
	// so a rejected turn is retryable without losing state.
	_, err := s.Step(map[string]any{})
	if err == nil {
		t.Fatal("expected an error for a missing required field, got none")
	}
	if err.Cat != "INPUT_ERROR" {
		t.Fatalf("got error category %q, want INPUT_ERROR", err.Cat)
	}
	if s.Turn() != 1 {
		t.Fatalf("after failed turn: got Turn()=%d, want unchanged 1", s.Turn())
	}
	if r := s.Last().(*big.Rat); r.Cmp(rat(7)) != 0 {
		t.Fatalf("after failed turn: got Last()=%v, want unchanged 7", s.Last())
	}
	if s.LastErr() != err {
		t.Fatalf("LastErr() did not return the error Step just returned")
	}
}

func TestSessionBaseIsNotMutated(t *testing.T) {
	pkg := echoPackage()
	s := NewSession(pkg)
	if _, err := s.Step(map[string]any{"n": rat(1)}); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if _, has := pkg["input"]; has {
		t.Fatal("NewSession's caller-owned package map gained an \"input\" key")
	}
}

func TestCarry(t *testing.T) {
	base := map[string]any{"action": rat(1)}

	withPrev := Carry(base, "game", "state-blob")
	if withPrev["game"] != "state-blob" {
		t.Fatalf("got %v, want game=state-blob", withPrev)
	}
	if _, has := base["game"]; has {
		t.Fatal("Carry mutated its input map")
	}

	withoutPrev := Carry(base, "game", nil)
	if _, has := withoutPrev["game"]; has {
		t.Fatalf("Carry with a nil prev added a game key: %v", withoutPrev)
	}
	if len(withoutPrev) != len(base) {
		t.Fatalf("got %v, want an unchanged copy of %v", withoutPrev, base)
	}
}

func TestRenderJSONIsValidAndIndented(t *testing.T) {
	v := map[string]any{"b": rat(2), "a": []any{"x", true, nil}}
	out, err := Render(v, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	want := "{\n  \"a\": [\n    \"x\",\n    true,\n    null\n  ],\n  \"b\": 2\n}\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRenderTextAndMarkdownAreDeterministic(t *testing.T) {
	v := map[string]any{"z": rat(1), "a": "hi"}
	text, err := Render(v, FormatText)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if text != "a: hi\nz: 1\n" {
		t.Fatalf("got %q", text)
	}

	md, err := Render(v, FormatMarkdown)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if md != "- **a:** `hi`\n- **z:** 1\n" {
		t.Fatalf("got %q", md)
	}
}

func TestRenderUnknownFormat(t *testing.T) {
	if _, err := Render(nil, "yaml"); err == nil {
		t.Fatal("expected an error for an unknown format")
	}
}

// sanity check that jaxson.Normalize round-trips the way LoadPackage
// and LoadTurns rely on, without needing a file on disk for this case.
func TestNormalizeSanity(t *testing.T) {
	v := jaxson.Normalize(map[string]any{"n": json.Number("3")})
	m := v.(map[string]any)
	if r, ok := m["n"].(*big.Rat); !ok || r.Cmp(rat(3)) != 0 {
		t.Fatalf("got %v, want *big.Rat(3)", m["n"])
	}
}
