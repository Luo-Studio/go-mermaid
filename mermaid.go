// Package mermaid is the top-level entry point for go-mermaid. It
// detects the diagram type from source, dispatches to the
// appropriate per-type package, and returns a style-neutral
// DisplayList.
package mermaid

import (
	"errors"
	"strings"

	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/fontmetrics"
)

// Measurer reports the rendered width and height of a string in the
// caller's font for the given semantic Role. Implementations are
// expected to be deterministic for the same input.
type Measurer interface {
	Measure(text string, role displaylist.Role) (w, h float64)
}

// LayoutOptions are common knobs shared across diagram types.
type LayoutOptions struct {
	// Measurer measures rendered text. If nil, layout uses the
	// embedded Inter metrics measurer at FontSize.
	Measurer Measurer

	// FontSize used by the default Measurer in DisplayList units
	// (typically points). Default: 14.
	FontSize float64

	// Padding around the diagram's bbox.
	Padding float64

	// NodeSpacing is the horizontal/sibling spacing autog uses.
	NodeSpacing float64

	// LayerSpacing is the vertical/cross-layer spacing autog uses.
	LayerSpacing float64

	// Sequence-specific spacing knobs. Ignored for non-sequence
	// diagrams.
	SequenceActorSpacing   float64
	SequenceMessageSpacing float64
}

// measurer returns the Measurer to use: the explicit one if set,
// otherwise a default backed by the embedded Inter metrics.
func (o LayoutOptions) measurer() Measurer {
	if o.Measurer != nil {
		return o.Measurer
	}
	fs := o.FontSize
	if fs <= 0 {
		fs = 14
	}
	return fontmetrics.NewDefault(fs)
}

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
