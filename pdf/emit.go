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
	// If Theme is also set, Style overrides the theme.
	Style Style

	// Theme picks one of the named palettes from the theme package
	// ("dracula", "tokyo-night", "github-light", ...). Empty = default.
	// Resolved into Style at draw time.
	Theme string

	// Layout knobs forwarded to mermaid.ParseAndLayout.
	Layout layoutopts.Options

	// MaxWidth caps the rendered width in fpdf's current unit. If the
	// laid-out DisplayList is wider, it is uniformly scaled down.
	// 0 = no cap. Ignored if Width or Height is set.
	MaxWidth float64

	// Width and Height, when set, request a target render box. The
	// diagram is uniformly scaled (aspect preserved) to fit within
	// the box: scale = min(Width/dl.Width, Height/dl.Height) when
	// both are set, otherwise the single set field drives the scale.
	// 0 on both = use natural size (subject to MaxWidth).
	Width, Height float64

	// Padding around the diagram (in fpdf's current unit).
	Padding float64

	// FillBackground paints the theme's background color across the
	// diagram's bbox before drawing. Useful for dark themes; harmless
	// for light themes.
	FillBackground bool
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

// SizeForOptions returns the rendered diagram size in fpdf units
// after applying opts.Width / opts.Height / opts.MaxWidth scaling.
// Useful for callers that need to position content immediately
// below the diagram before calling DrawInto.
func SizeForOptions(dl *displaylist.DisplayList, opts EmbedOptions) (w, h float64) {
	if dl == nil || dl.Width <= 0 || dl.Height <= 0 {
		return 0, 0
	}
	scale := scaleForOptions(dl, opts)
	return dl.Width * scale, dl.Height * scale
}

func scaleForOptions(dl *displaylist.DisplayList, opts EmbedOptions) float64 {
	switch {
	case opts.Width > 0 && opts.Height > 0:
		sx := opts.Width / dl.Width
		sy := opts.Height / dl.Height
		if sy < sx {
			return sy
		}
		return sx
	case opts.Width > 0:
		return opts.Width / dl.Width
	case opts.Height > 0:
		return opts.Height / dl.Height
	}
	if opts.MaxWidth > 0 && dl.Width > opts.MaxWidth {
		return opts.MaxWidth / dl.Width
	}
	return 1.0
}

// DrawInto draws an already-laid-out DisplayList into pdf at (x, y).
func DrawInto(pdf *fpdf.Fpdf, dl *displaylist.DisplayList, x, y float64, opts EmbedOptions) error {
	if dl == nil || len(dl.Items) == 0 {
		return nil
	}
	// Register the embedded Inter UTF-8 font once per fpdf doc so
	// labels with non-Latin-1 characters (emoji, CJK, …) render
	// correctly. Idempotent.
	if err := ensureInterFont(pdf); err != nil {
		return err
	}
	// Best-effort: if a system emoji TTF is installed, register it so
	// drawText can switch fonts for emoji runs. Failure is silent —
	// drawText falls back to stripping emoji glyphs.
	ensureEmojiFont(pdf)
	style := opts.Style
	hasExplicitStyle := len(style.Roles) > 0 || style.Default.Font != "" || style.Default.StrokeWidth != 0
	if !hasExplicitStyle {
		if opts.Theme != "" {
			ts, err := StyleFromTheme(opts.Theme)
			if err != nil {
				return err
			}
			style = ts
		} else {
			style = DefaultStyle()
		}
	}
	// One DisplayList unit = one fpdf unit by default. Width/Height
	// (if set) take precedence — the diagram is scaled to fit within
	// that box, aspect preserved. Otherwise MaxWidth applies as a
	// scale-down cap. See scaleForOptions for the full rule.
	scale := scaleForOptions(dl, opts)

	if opts.FillBackground && opts.Theme != "" {
		br, bg, bb := PageBackground(opts.Theme)
		pdf.SetFillColor(br, bg, bb)
		// Bg covers diagram bbox (scaled) plus padding.
		pad := opts.Padding
		pdf.Rect(x, y, dl.Width*scale+pad*2, dl.Height*scale+pad*2, "F")
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
