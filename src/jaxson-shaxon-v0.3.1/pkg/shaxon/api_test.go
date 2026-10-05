// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// A package whose program and validate entries would fail if they ran, so a
// Validate that touched them would show it.
const apiPkg = `{
  "shaxon": "3.1",
  "limits": {"steps": 5000},
  "inputSchema": {"type": "object", "fields": {"never": {"type": "string"}}, "required": ["never"]},
  "shapes": {
    "Person": {"kind": "object", "required": ["name"], "closed": true,
      "fields": {"name": {"kind": "string"}, "age": {"kind": "number"}}},
    "Tagged": {"kind": "object", "fields": {"tag": {"kind": "reference", "index": "tagsById"}, "tags": {"kind": "array"}}}
  },
  "indices": {"tagsById": {"source": {"$path": ["input", "tags"]}, "key": {"$path": ["local", "item", "id"]}}},
  "validate": [{"target": {"$path": ["input", "nowhere"]}, "shape": "Person", "mode": "gate"}],
  "program": [{"op": "set", "path": ["output"], "value": {"$compute": {"with": {}, "expr": ["div", 1, 0, 2]}}}]
}`

func apiDoc(t *testing.T, s string) any { return decode(t, s) }

func TestValidateConforming(t *testing.T) {
	rep, err := Validate(decode(t, apiPkg), "Person", apiDoc(t, `{"name": "ann", "age": 7}`))
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Conforms() || len(rep.Violations) != 0 {
		t.Fatalf("want a conforming report, got %v", rep.Value())
	}
}

func TestValidateReportsEveryFinding(t *testing.T) {
	rep, err := Validate(decode(t, apiPkg), "Person", apiDoc(t, `{"age": "x", "extra": 1}`))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Conforms() || len(rep.Violations) != 3 {
		t.Fatalf("want 3 findings (missing name, bad age, closed), got %v", rep.Value())
	}
}

func TestValidateUsesIndicesOverTheDocument(t *testing.T) {
	pkg := decode(t, apiPkg)
	doc := `{"tags": [{"id": "a"}, {"id": "b"}], "tag": "zz"}`
	rep, err := Validate(pkg, "Tagged", apiDoc(t, doc))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Conforms() || len(rep.Violations) != 1 || rep.Violations[0].Code != CodeDanglingReference {
		t.Fatalf("want one dangling reference, got %v", rep.Value())
	}
	rep, err = Validate(pkg, "Tagged", apiDoc(t, `{"tags": [{"id": "a"}], "tag": "a"}`))
	if err != nil || !rep.Conforms() {
		t.Fatalf("want conforming, got %v, %v", rep.Value(), err)
	}
}

func TestValidateErrors(t *testing.T) {
	pkg := decode(t, apiPkg)
	if _, err := Validate(pkg, "NoSuchShape", apiDoc(t, `{}`)); err == nil {
		t.Error("an unknown shape was accepted")
	}
	if _, err := Validate(map[string]any{"shaxon": "9"}, "Person", apiDoc(t, `{}`)); err == nil || err.Cat != "VERSION_ERROR" {
		t.Errorf("a bad version gave %v", err)
	}
	tight := decode(t, apiPkg)
	tight["limits"] = decode(t, `{"steps": 1}`)
	if _, err := Validate(tight, "Person", apiDoc(t, `{"name": "a"}`)); err == nil || err.Cat != "RESOURCE_ERROR" {
		t.Errorf("a step limit of 1 gave %v", err)
	}
}

func TestValidateChangesNeitherArgument(t *testing.T) {
	pkg := decode(t, apiPkg)
	doc := apiDoc(t, `{"name": "ann", "tags": [{"id": "a"}]}`)
	beforeP, beforeD := jaxson.Show(pkg), jaxson.Show(doc)
	if _, err := Validate(pkg, "Person", doc); err != nil && err.Cat != "" {
		_ = err
	}
	if jaxson.Show(pkg) != beforeP || jaxson.Show(doc) != beforeD {
		t.Fatal("Validate changed its arguments")
	}
	if _, has := pkg["program"]; !has {
		t.Fatal("the package lost its program")
	}
}

func TestValidateAcceptsObjectsAndRawJSON(t *testing.T) {
	pkgRaw := []byte(`{"shaxon":"3.1","shapes":{"P":{"kind":"object","required":["n"],"fields":{"n":{"kind":"number"}}}}}`)
	rep, err := ValidateJSON(pkgRaw, "P", []byte(`{"n": 1.5}`))
	if err != nil || !rep.Conforms() {
		t.Fatalf("ValidateJSON conforming: %v, %v", rep.Value(), err)
	}
	rep, err = ValidateJSON(pkgRaw, "P", []byte(`{"n": "x"}`))
	if err != nil || rep.Conforms() {
		t.Fatalf("ValidateJSON violating: %v, %v", rep.Value(), err)
	}
	if _, err = ValidateJSON(pkgRaw, "P", []byte(`{"n": 1, "n": 2}`)); err == nil || err.Cat != "PARSE_ERROR" {
		t.Errorf("a duplicate key in the document gave %v", err)
	}
	if _, err = ValidateJSON([]byte(`[1]`), "P", []byte(`{}`)); err == nil || err.Cat != "PARSE_ERROR" {
		t.Errorf("a package that is an array gave %v", err)
	}
}

// Phase 7.1's requirement that the tables be immutable once built: many
// goroutines running and validating against one decoded package and one
// document at once. Meaningful under -race.
func TestConcurrentRunAndValidateShareAPackage(t *testing.T) {
	pkg := decode(t, `{
	  "shaxon": "3.1", "limits": {"steps": 100000},
	  "input": {"items": [{"id": "a", "n": 1}, {"id": "b", "n": 2}, {"id": "c", "n": 3}]},
	  "shapes": {"Item": {"kind": "object", "required": ["id"], "fields": {"id": {"kind": "string"}, "n": {"kind": "number"}}}},
	  "validate": [{"target": {"$each": ["input", "items"]}, "shape": "Item", "mode": "report"}],
	  "program": [
	    {"op": "set", "path": ["output"], "value": {"$lit": {"total": 0}}},
	    {"op": "for", "in": {"$path": ["input", "items"]}, "as": "it", "do": [
	      {"op": "set", "path": ["output", "total"], "value": {"$compute": {"with": {"t": {"$path": ["output", "total"]}, "n": {"$path": ["local", "it", "n"]}}, "expr": ["add", {"$v": "t"}, {"$v": "n"}]}}}
	    ]}
	  ]}`)
	doc := apiDoc(t, `{"id": "z", "n": 4}`)
	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				res, err := Run(pkg)
				if err != nil {
					errs <- fmt.Sprint("Run: ", err)
					return
				}
				if got := jaxson.Show(res.Output); got != `{"total":6}` {
					errs <- "Run output " + got
					return
				}
				rep, err := Validate(pkg, "Item", doc)
				if err != nil || !rep.Conforms() {
					errs <- fmt.Sprint("Validate: ", rep.Value(), err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}
