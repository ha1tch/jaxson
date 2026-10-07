// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// LoadPackage reads and decodes a Jaxson package file, normalizing its
// numbers (json.Number -> *big.Rat, recursively, via jaxson.Normalize)
// so the result is ready for NewSession or jaxson.Run directly.
func LoadPackage(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var pkg map[string]any
	if err := dec.Decode(&pkg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return jaxson.Normalize(pkg).(map[string]any), nil
}

// LoadTurns reads a scripted-session file: a JSON array of per-turn
// input objects, e.g.
//
//	[
//	  {"action": "new", "seed": 0},
//	  {"action": "fire", "x": 3, "y": 4},
//	  {"action": "fire", "x": 5, "y": 5}
//	]
//
// Numbers are normalized the same way LoadPackage's are. Each element
// must be a JSON object; it becomes one Step's input, unmodified
// except for whatever Carry (or an equivalent) the caller applies
// before passing it to Session.Step.
func LoadTurns(path string) ([]map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var turns []map[string]any
	if err := dec.Decode(&turns); err != nil {
		return nil, fmt.Errorf("%s: must be a JSON array of turn objects: %w", path, err)
	}
	for i, t := range turns {
		turns[i] = jaxson.Normalize(t).(map[string]any)
	}
	return turns, nil
}
