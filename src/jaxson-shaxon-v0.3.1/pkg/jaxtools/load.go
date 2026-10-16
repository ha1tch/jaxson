// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

import (
	"fmt"
	"os"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// LoadPackage reads a Jaxson package file with the language's own reader
// (jaxson.ParseJSON), so a duplicate key is an error and a number that is not
// a Real is a PARSE_ERROR. The result is ready for NewSession or jaxson.Run
// directly.
func LoadPackage(path string) (map[string]any, error) {
	return LoadPackageMode(path, jaxson.Strict)
}

// LoadPackageMode is LoadPackage in the given numeric mode (Jaxson core
// design, section 2): in relaxed mode a number beyond the Real limits is a
// float64 if a float64 can hold it.
func LoadPackageMode(path string, mode jaxson.NumericMode) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v, perr := jaxson.ParseJSONMode(raw, mode)
	if perr != nil {
		return nil, fmt.Errorf("%s: %w", path, perr)
	}
	pkg, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: a package must be a JSON object", path)
	}
	return pkg, nil
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
	return LoadTurnsMode(path, jaxson.Strict)
}

// LoadTurnsMode is LoadTurns in the given numeric mode.
func LoadTurnsMode(path string, mode jaxson.NumericMode) ([]map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v, perr := jaxson.ParseJSONMode(raw, mode)
	if perr != nil {
		return nil, fmt.Errorf("%s: %w", path, perr)
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: must be a JSON array of turn objects", path)
	}
	turns := make([]map[string]any, len(arr))
	for i, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: turn %d must be a JSON object", path, i+1)
		}
		turns[i] = m
	}
	return turns, nil
}
