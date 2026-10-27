// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func mustGen(t testing.TB, seed uint64, size string) *DB {
	t.Helper()
	db, err := Generate(seed, size, seed%7 != 0)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestGenerateIsDeterministic(t *testing.T) {
	a, b := mustGen(t, 3, "medium"), mustGen(t, 3, "medium")
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed gave different databases")
	}
	if reflect.DeepEqual(a, mustGen(t, 4, "medium")) {
		t.Fatal("different seeds gave the same database")
	}
}

func TestGeneratedDatabasesAreReferentiallyClean(t *testing.T) {
	for _, size := range PresetOrder {
		if err := checkRefs(mustGen(t, 1, size)); err != nil {
			t.Fatalf("%s: %v", size, err)
		}
	}
}

func TestRoundTripThroughJSON(t *testing.T) {
	db := mustGen(t, 5, "small")
	raw, err := json.Marshal(db)
	if err != nil {
		t.Fatal(err)
	}
	var back DB
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(db, &back) {
		t.Fatal("database changed in a JSON round trip")
	}
}

func TestFastAgreesWithReference(t *testing.T) {
	sizes := []string{"tiny", "small", "small", "medium"}
	for seed := uint64(0); seed < 40; seed++ {
		db := mustGen(t, seed, sizes[seed%4])
		for hops := 0; hops <= MaxHops+1; hops++ {
			ref, err := EvalRef(db, hops)
			if err != nil {
				t.Fatal(err)
			}
			fast, err := EvalFast(db, hops)
			if err != nil {
				t.Fatal(err)
			}
			if !sameResults(db, ref, fast) {
				t.Fatalf("seed %d hops %d: results differ", seed, hops)
			}
		}
	}
}

func TestDanglingReferenceIsRejectedByBoth(t *testing.T) {
	db := mustGen(t, 2, "tiny")
	db.Grants[0].Role = "r99"
	if _, err := EvalRef(db, MaxHops); err == nil {
		t.Error("reference accepted a dangling role")
	}
	if _, err := EvalFast(db, MaxHops); err == nil {
		t.Error("optimized oracle accepted a dangling role")
	}
}

func TestUnknownActionIsDeniedWithoutSupport(t *testing.T) {
	db := mustGen(t, 2, "tiny")
	db.Requests[0].Action = "teleport"
	ref, _ := EvalRef(db, MaxHops)
	fast, _ := EvalFast(db, MaxHops)
	if !sameResults(db, ref, fast) || fast[0].Allow || len(fast[0].DeniedBy) != 0 {
		t.Fatalf("unknown action: %+v", fast[0])
	}
}

// The generator must exercise what the example is about: a meaningful share of
// decisions that rest on delegation and some that deny.
func TestGeneratorPlantsTheHardCases(t *testing.T) {
	db := mustGen(t, 11, "medium")
	res, _ := EvalFast(db, MaxHops)
	allow, deny, deleg := 0, 0, 0
	for _, r := range res {
		if r.Allow {
			allow++
			if r.Via != nil && len(*r.Via) > 5 {
				deleg++
			}
		}
		if len(r.DeniedBy) > 0 {
			deny++
		}
	}
	if allow == 0 || deny == 0 || deleg == 0 {
		t.Fatalf("allow=%d deny=%d delegated=%d", allow, deny, deleg)
	}
}

func benchEval(b *testing.B, size string, fn func(*DB, int)) {
	db, err := Generate(0, size, true)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn(db, MaxHops)
	}
}

func BenchmarkRefLarge(b *testing.B) {
	benchEval(b, "large", func(db *DB, h int) { EvalRef(db, h) })
}
func BenchmarkFastLarge(b *testing.B) {
	benchEval(b, "large", func(db *DB, h int) { EvalFast(db, h) })
}
func BenchmarkRefXLarge(b *testing.B) {
	benchEval(b, "xlarge", func(db *DB, h int) { EvalRef(db, h) })
}
func BenchmarkFastXLarge(b *testing.B) {
	benchEval(b, "xlarge", func(db *DB, h int) { EvalFast(db, h) })
}

func TestWrongWidthIdIsRejectedByBoth(t *testing.T) {
	for _, mutate := range []func(*DB){
		func(db *DB) { db.Grants[0].ID = "G0001" },
		func(db *DB) { db.Delegations[0].ID = db.Delegations[0].ID + "0" },
	} {
		db := mustGen(t, 2, "tiny")
		mutate(db)
		if _, err := EvalRef(db, MaxHops); err == nil {
			t.Error("reference accepted a wrong-width id")
		}
		if _, err := EvalFast(db, MaxHops); err == nil {
			t.Error("optimized oracle accepted a wrong-width id")
		}
	}
}

// More than 9,999 grants: past the old four-digit cap, answered the same by both.
func TestMoreThanFourDigitsOfGrants(t *testing.T) {
	Presets["wideids"] = Preset{60, 14, 10, 60, 10050, 18, 40, 6, 200}
	defer delete(Presets, "wideids")
	db := mustGen(t, 2, "wideids")
	if len(db.Grants) <= 9999 {
		t.Fatalf("only %d grants", len(db.Grants))
	}
	ref, err := EvalRef(db, MaxHops)
	if err != nil {
		t.Fatal(err)
	}
	fast, err := EvalFast(db, MaxHops)
	if err != nil {
		t.Fatal(err)
	}
	if !sameResults(db, ref, fast) {
		t.Fatal("results differ past the four-digit cap")
	}
}
