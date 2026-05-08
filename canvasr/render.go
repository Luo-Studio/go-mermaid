package mermaidcanvasr

import (
	"bytes"
	"fmt"
	"math"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"

	mermaid "github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// RenderOptions configures rasterization / vector emission.
type RenderOptions struct {
	Style    Style
	Layout   layoutopts.Options
	DPI      float64
	MaxWidth float64
	Padding  float64
}

// RenderPNG parses src, lays it out, and returns PNG bytes.
func RenderPNG(src string, opts RenderOptions) ([]byte, error) {
	dl, err := mermaid.ParseAndLayout(src, opts.Layout)
	if err != nil {
		return nil, err
	}
	if dl == nil {
		return nil, fmt.Errorf("canvasr: empty diagram")
	}
	return RenderDisplayListPNG(dl, opts)
}

// RenderDisplayListPNG renders an already-laid-out DisplayList to PNG.
func RenderDisplayListPNG(dl *displaylist.DisplayList, opts RenderOptions) ([]byte, error) {
	c, err := buildCanvas(dl, &opts)
	if err != nil {
		return nil, err
	}
	dpi := opts.DPI
	if dpi <= 0 {
		dpi = 192
	}
	var buf bytes.Buffer
	writer := renderers.PNG(canvas.DPMM(dpi / 25.4))
	if err := writer(&buf, c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderSVG renders an already-laid-out DisplayList to SVG bytes.
func RenderSVG(src string, opts RenderOptions) ([]byte, error) {
	dl, err := mermaid.ParseAndLayout(src, opts.Layout)
	if err != nil {
		return nil, err
	}
	c, err := buildCanvas(dl, &opts)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writer := renderers.SVG()
	if err := writer(&buf, c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildCanvas(dl *displaylist.DisplayList, opts *RenderOptions) (*canvas.Canvas, error) {
	if opts.Style.FontFamily == nil {
		st, err := DefaultStyle()
		if err != nil {
			return nil, err
		}
		opts.Style = st
	}
	pad := opts.Padding
	w := dl.Width + pad*2
	h := dl.Height + pad*2
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	c := canvas.New(w, h)
	ctx := canvas.NewContext(c)
	// DisplayList: y grows DOWN. tdewolff/canvas: y grows UP from
	// origin. We flip Y by translating to the top and scaling -1.
	ctx.SetCoordSystem(canvas.CartesianIV)
	ctx.Translate(pad, pad)

	for _, it := range dl.Items {
		if cl, ok := it.(displaylist.Cluster); ok {
			drawCluster(ctx, cl, opts.Style)
		}
	}
	for _, it := range dl.Items {
		if s, ok := it.(displaylist.Shape); ok {
			drawShape(ctx, s, opts.Style.lookup(s.Role))
		}
	}
	for _, it := range dl.Items {
		if e, ok := it.(displaylist.Edge); ok {
			drawEdge(ctx, e, opts.Style.lookup(e.Role))
		}
	}
	for _, it := range dl.Items {
		if t, ok := it.(displaylist.Text); ok {
			drawText(ctx, t, opts.Style)
		}
	}
	return c, nil
}

func _avoidUnused() { _ = math.Pi }
