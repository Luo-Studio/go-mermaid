// Package layoutopts holds the shared layout-stage knobs (Measurer,
// spacings, padding) so per-diagram-type packages and the top-level
// `mermaid` package can both refer to them without a cycle.
package layoutopts

import (
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/fontmetrics"
)

// Measurer reports the rendered width and height of a string in the
// caller's font for the given semantic Role. Implementations are
// expected to be deterministic for the same input.
type Measurer interface {
	Measure(text string, role displaylist.Role) (w, h float64)
}

// Options are the layout-stage knobs shared across diagram types.
// Per-type packages may consume only a subset of fields.
type Options struct {
	Measurer Measurer

	// FontSize used by the default Measurer when Measurer is nil. The
	// value is in DisplayList units — when those map to mm (the
	// typical fpdf default), 4 corresponds to ≈10pt body text. Pass
	// a larger value if your DL units are pt and you want bigger
	// text. Default: 4.
	FontSize float64

	// Padding around the diagram bbox.
	Padding float64

	// NodeSpacing / LayerSpacing — autog tuning. Defaults sized for
	// legibility at typical PDF font sizes.
	NodeSpacing  float64
	LayerSpacing float64

	// Sequence-specific spacing knobs. Ignored for non-sequence
	// diagrams.
	SequenceActorSpacing   float64
	SequenceMessageSpacing float64
}

// ResolveMeasurer returns the Measurer to use: the explicit one if
// set, otherwise a default backed by the embedded Inter metrics.
func (o Options) ResolveMeasurer() Measurer {
	if o.Measurer != nil {
		return o.Measurer
	}
	fs := o.FontSize
	if fs <= 0 {
		fs = 4
	}
	return fontmetrics.NewDefault(fs)
}
