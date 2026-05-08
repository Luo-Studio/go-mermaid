// Package textutil provides label-normalisation helpers shared by all
// diagram-type parsers and layouts.
package textutil

import (
	"regexp"
	"strings"
)

var brTagRE = regexp.MustCompile(`(?i)<\s*br\s*/?\s*>`)

// CleanLabel normalises a raw label string: strips surrounding double
// or single quotes (so users can write `"My Label"` to allow spaces
// without seeing the literal quotes in the rendered diagram), and
// rewrites HTML line breaks (`<br>`, `<br/>`, `<br />`,
// case-insensitive) to real newlines so multi-line labels render
// across multiple stacked rows.
func CleanLabel(s string) string {
	s = strings.TrimSpace(s)
	for len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			s = strings.TrimSpace(s[1 : len(s)-1])
			continue
		}
		break
	}
	return brTagRE.ReplaceAllString(s, "\n")
}

// SplitLabelLines splits a (cleaned) label on newlines for emit-time
// stacking. An empty input yields one empty line so a Text item with
// VAlignMiddle still has a row to position against.
func SplitLabelLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}
