// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// Format selects how Render renders a Jaxson value for a developer to
// read. None of the three formats know anything about any particular
// package's field meanings — they render the value's own shape
// (object, array, string, number, boolean, null), nothing more. A
// package that wants a friendlier, field-aware presentation (Navy
// Wars' board-and-messages view, say) still needs its own formatter on
// top of the raw output; these three are the generic floor every
// package gets for free.
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
)

// Render renders v — a Jaxson value: nil, bool, string, *big.Rat,
// []any, or map[string]any, as returned by Session.Step or jaxson.Run
// — in the given format. An empty Format renders as FormatText.
func Render(v any, f Format) (string, error) {
	switch f {
	case FormatText, "":
		var b strings.Builder
		writeText(&b, v, 0)
		return b.String(), nil
	case FormatMarkdown:
		var b strings.Builder
		writeMarkdown(&b, v, 0)
		return b.String(), nil
	case FormatJSON:
		compact := jaxson.Show(v)
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(compact), "", "  "); err != nil {
			// jaxson.Show always produces valid JSON text; this is not
			// expected to happen, but fall back to the compact form
			// rather than lose the value.
			return compact + "\n", nil
		}
		return pretty.String() + "\n", nil
	default:
		return "", fmt.Errorf("unknown format %q (want %q, %q, or %q)", f, FormatText, FormatMarkdown, FormatJSON)
	}
}

func isScalar(v any) bool {
	switch v.(type) {
	case nil, bool, string, *big.Rat:
		return true
	}
	return false
}

// scalarText renders a leaf value as plain text, unquoted.
func scalarText(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return t
	case *big.Rat:
		return jaxson.Show(t) // show()'s *big.Rat case is exactly FormatDecimal, unwrapped
	}
	return fmt.Sprintf("%v", v)
}

func writeText(b *strings.Builder, v any, ind int) {
	pad := strings.Repeat("  ", ind)
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			fmt.Fprintf(b, "%s{}\n", pad)
			return
		}
		for _, k := range sortedKeysOf(t) {
			val := t[k]
			if isScalar(val) {
				fmt.Fprintf(b, "%s%s: %s\n", pad, k, scalarText(val))
			} else {
				fmt.Fprintf(b, "%s%s:\n", pad, k)
				writeText(b, val, ind+1)
			}
		}
	case []any:
		if len(t) == 0 {
			fmt.Fprintf(b, "%s[]\n", pad)
			return
		}
		for _, item := range t {
			if isScalar(item) {
				fmt.Fprintf(b, "%s- %s\n", pad, scalarText(item))
			} else {
				fmt.Fprintf(b, "%s-\n", pad)
				writeText(b, item, ind+1)
			}
		}
	default:
		fmt.Fprintf(b, "%s%s\n", pad, scalarText(v))
	}
}

func writeMarkdown(b *strings.Builder, v any, ind int) {
	pad := strings.Repeat("  ", ind)
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			fmt.Fprintf(b, "%s*(empty)*\n", pad)
			return
		}
		for _, k := range sortedKeysOf(t) {
			val := t[k]
			if isScalar(val) {
				fmt.Fprintf(b, "%s- **%s:** %s\n", pad, k, mdScalar(val))
			} else {
				fmt.Fprintf(b, "%s- **%s:**\n", pad, k)
				writeMarkdown(b, val, ind+1)
			}
		}
	case []any:
		if len(t) == 0 {
			fmt.Fprintf(b, "%s*(empty)*\n", pad)
			return
		}
		for _, item := range t {
			if isScalar(item) {
				fmt.Fprintf(b, "%s- %s\n", pad, mdScalar(item))
			} else {
				fmt.Fprintf(b, "%s-\n", pad)
				writeMarkdown(b, item, ind+1)
			}
		}
	default:
		fmt.Fprintf(b, "%s%s\n", pad, mdScalar(v))
	}
}

// mdScalar renders a leaf value for markdown: strings go in inline
// code spans so any markdown-significant characters a package's own
// text happens to contain (a "*", a "_", a "`") can't be mistaken for
// markup; other scalars render the same as in text mode.
func mdScalar(v any) string {
	if s, ok := v.(string); ok {
		return "`" + strings.ReplaceAll(s, "`", "'") + "`"
	}
	return scalarText(v)
}

func sortedKeysOf(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
