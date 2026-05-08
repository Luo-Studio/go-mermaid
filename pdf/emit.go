package mermaidpdf

import (
	"fmt"

	"codeberg.org/go-pdf/fpdf"

	mermaid "github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// EmbedOptions configures DrawMermaid / DrawInto.
type EmbedOptions struct {
	// Style maps DisplayList roles to fpdf colors / fonts / widths.
	Style Style

	// Layout knobs forwarded to mermaid.ParseAndLayout.
	Layout layoutopts.Options

	// MaxWidth caps the rendered width in fpdf's current unit. If the
	// laid-out DisplayList is wider, it is uniformly scaled down.
	// 0 = no cap.
	MaxWidth float64

	// Padding around the diagram (in fpdf's current unit).
	Padding float64
}

// EmbedDefaults returns sensible defaults (DefaultStyle, no cap).
func EmbedDefaults() EmbedOptions {
	return EmbedOptions{Style: DefaultStyle()}
}

// DrawMermaid is the one-call helper: parse → layout → draw at (x, y).
func DrawMermaid(pdf *fpdf.Fpdf, src string, x, y float64, opts EmbedOptions) error {
	dl, err := mermaid.ParseAndLayout(src, opts.Layout)
	if err != nil {
		return err
	}
	return DrawInto(pdf, dl, x, y, opts)
}

// DrawInto draws an already-laid-out DisplayList into pdf at (x, y).
func DrawInto(pdf *fpdf.Fpdf, dl *displaylist.DisplayList, x, y float64, opts EmbedOptions) error {
	if dl == nil || len(dl.Items) == 0 {
		return nil
	}
	style := opts.Style
	if len(style.Roles) == 0 && style.Default.Font == "" && style.Default.StrokeWidth == 0 {
		style = DefaultStyle()
	}

	// One DisplayList unit = one fpdf unit. Callers wanting a
	// different mapping should scale via opts.MaxWidth.
	scale := 1.0
	if opts.MaxWidth > 0 && dl.Width*scale > opts.MaxWidth {
		scale = opts.MaxWidth / dl.Width
	}

	dx := x + opts.Padding
	dy := y + opts.Padding

	tx := func(p displaylist.Point) (float64, float64) {
		return dx + p.X*scale, dy + p.Y*scale
	}
	tr := func(r displaylist.Rect) (float64, float64, float64, float64) {
		return dx + r.X*scale, dy + r.Y*scale, r.W * scale, r.H * scale
	}

	// Pass 1: clusters (so they sit behind nodes).
	for _, it := range dl.Items {
		if c, ok := it.(displaylist.Cluster); ok {
			drawCluster(pdf, c, tr, style.lookup(c.Role), style.lookup(displaylist.RoleClusterTitle))
		}
	}

	// Pass 2: shapes, edges, text, markers (shapes before edges so
	// edge-label text overlays cleanly).
	for _, it := range dl.Items {
		switch v := it.(type) {
		case displaylist.Shape:
			drawShape(pdf, v, tr, style.lookup(v.Role))
		}
	}
	for _, it := range dl.Items {
		switch v := it.(type) {
		case displaylist.Edge:
			drawEdge(pdf, v, tx, style.lookup(v.Role), scale)
		case displaylist.Marker:
			drawMarker(pdf, v, tx, style.lookup(v.Role), scale)
		}
	}
	for _, it := range dl.Items {
		switch v := it.(type) {
		case displaylist.Text:
			drawText(pdf, v, tx, style.lookup(v.Role))
		case displaylist.Cluster, displaylist.Shape, displaylist.Edge, displaylist.Marker:
			// already handled
		default:
			return fmt.Errorf("mermaidpdf: unknown DisplayList item kind %T", v)
		}
	}
	return nil
}
