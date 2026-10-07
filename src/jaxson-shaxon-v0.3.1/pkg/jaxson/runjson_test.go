// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import "testing"

const runJSONPkg = `{"jaxson":"1.0","inputSchema":{"type":"any"},"outputSchema":{"type":"any"},"input":{"a":2,"b":{"c":3}},
 "program":[{"op":"set","path":["output"],"value":{"$compute":{"with":{"x":{"$path":["input","a"]},"y":{"$path":["input","b","c"]}},"expr":["mul",{"$v":"x"},{"$v":"y"}]}}}]}`

func TestRunJSONAgreesWithRun(t *testing.T) {
	out, err := RunJSON([]byte(runJSONPkg))
	if err != nil {
		t.Fatal(err)
	}
	v, perr := ParseJSON([]byte(runJSONPkg))
	if perr != nil {
		t.Fatal(perr)
	}
	want, err := Run(v.(map[string]any))
	if err != nil || !Equal(out, want) || Show(out) != "6" {
		t.Fatalf("RunJSON %s, Run %s, %v", Show(out), Show(want), err)
	}
}

func TestRunJSONRejectsBadDocuments(t *testing.T) {
	for _, doc := range []string{``, `[1]`, `{"a":1,"a":2}`, `{"input":{"k":1,"k":2}}`, `{} {}`, `{"jaxson":`} {
		if _, err := RunJSON([]byte(doc)); err == nil || err.Cat != "PARSE_ERROR" {
			t.Errorf("RunJSON(%q) = %v", doc, err)
		}
	}
}

func TestErrString(t *testing.T) {
	if got := (&Err{Cat: "PARSE_ERROR", Msg: "x"}).Error(); got != "PARSE_ERROR: x" {
		t.Errorf("no code: %q", got)
	}
	if got := (&Err{Cat: "EXECUTION_ERROR", Code: "DIV_ZERO", Msg: "x"}).Error(); got != "EXECUTION_ERROR/DIV_ZERO: x" {
		t.Errorf("with code: %q", got)
	}
}
