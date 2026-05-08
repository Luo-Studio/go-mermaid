// Package mermaid is the top-level entry point for go-mermaid. It
// detects the diagram type from source, dispatches to the
// appropriate per-type package, and returns a style-neutral
// DisplayList.
package mermaid

import (
	"errors"
	"strings"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// LayoutOptions is the user-facing alias for layoutopts.Options.
// Defined as a type alias so per-diagram-type packages can take the
// same struct without importing this top-level package (which would
// create an import cycle).
type LayoutOptions = layoutopts.Options

// Measurer is the user-facing alias for layoutopts.Measurer.
type Measurer = layoutopts.Measurer

// Errors.
var (
	ErrUnknownDiagram = errors.New("mermaid: unrecognized diagram type")
	ErrNotImplemented = errors.New("mermaid: diagram type recognized but not implemented in this build")
	ErrParse          = errors.New("mermaid: parse error")
	ErrLayout         = errors.New("mermaid: layout error")
)

// ParseAndLayout detects the diagram type from src, runs the parser
// and layout for that type, and returns the resulting DisplayList.
func ParseAndLayout(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	switch detectType(src) {
	case typeFlowchart:
		return parseAndLayoutFlowchart(src, opts)
	case typeSequence:
		return parseAndLayoutSequence(src, opts)
	case typeClass:
		return parseAndLayoutClass(src, opts)
	case typeER:
		return parseAndLayoutER(src, opts)
	case typeState:
		return parseAndLayoutState(src, opts)
	default:
		return nil, ErrUnknownDiagram
	}
}

type diagramType int

const (
	typeUnknown diagramType = iota
	typeFlowchart
	typeSequence
	typeClass
	typeER
	typeState
)

// DetectDiagramType returns a string identifier for the diagram type
// implied by src ("flowchart", "sequence", "class", "er", "state",
// or "" for unknown). Used by external tools (e.g., cmd/parse).
func DetectDiagramType(src string) string {
	switch detectType(src) {
	case typeFlowchart:
		return "flowchart"
	case typeSequence:
		return "sequence"
	case typeClass:
		return "class"
	case typeER:
		return "er"
	case typeState:
		return "state"
	default:
		return ""
	}
}

// detectType returns the diagram type implied by src's first non-
// blank, non-comment line.
func detectType(src string) diagramType {
	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "%%") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		head := strings.ToLower(fields[0])
		switch {
		case head == "flowchart" || head == "graph":
			return typeFlowchart
		case head == "sequencediagram":
			return typeSequence
		case head == "classdiagram":
			return typeClass
		case head == "erdiagram":
			return typeER
		case head == "statediagram" || head == "statediagram-v2":
			return typeState
		default:
			return typeUnknown
		}
	}
	return typeUnknown
}
