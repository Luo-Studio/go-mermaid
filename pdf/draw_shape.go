package mermaidpdf

import (
	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/displaylist"
)

// drawShape renders a Shape primitive into the pdf.
func drawShape(pdf *fpdf.Fpdf, s displaylist.Shape, tr func(displaylist.Rect) (float64, float64, float64, float64), rs RoleStyle) {
	x, y, w, h := tr(s.BBox)
	applyStroke(pdf, rs)
	fillStyle := applyFill(pdf, rs)

	switch s.Kind {
	case displaylist.ShapeKindRect, displaylist.ShapeKindSubroutine:
		pdf.Rect(x, y, w, h, fillStyle)
		if s.Kind == displaylist.ShapeKindSubroutine {
			// Inner vertical bars on each side.
			pdf.Line(x+4, y, x+4, y+h)
			pdf.Line(x+w-4, y, x+w-4, y+h)
		}
	case displaylist.ShapeKindRound:
		r := minf(8, w/2, h/2)
		pdf.RoundedRect(x, y, w, h, r, "1234", fillStyle)
	case displaylist.ShapeKindStadium:
		r := h / 2
		pdf.RoundedRect(x, y, w, h, r, "1234", fillStyle)
	case displaylist.ShapeKindCircle, displaylist.ShapeKindDoubleCircle:
		cx, cy := x+w/2, y+h/2
		r := w / 2
		pdf.Ellipse(cx, cy, r, r, 0, fillStyle)
		if s.Kind == displaylist.ShapeKindDoubleCircle {
			pdf.Ellipse(cx, cy, r-2, r-2, 0, fillStyle)
		}
	case displaylist.ShapeKindEllipse:
		pdf.Ellipse(x+w/2, y+h/2, w/2, h/2, 0, fillStyle)
	case displaylist.ShapeKindDiamond:
		pdf.Polygon([]fpdf.PointType{
			{X: x + w/2, Y: y},
			{X: x + w, Y: y + h/2},
			{X: x + w/2, Y: y + h},
			{X: x, Y: y + h/2},
		}, fillStyle)
	case displaylist.ShapeKindHexagon:
		off := w * 0.18
		pdf.Polygon([]fpdf.PointType{
			{X: x + off, Y: y},
			{X: x + w - off, Y: y},
			{X: x + w, Y: y + h/2},
			{X: x + w - off, Y: y + h},
			{X: x + off, Y: y + h},
			{X: x, Y: y + h/2},
		}, fillStyle)
	case displaylist.ShapeKindCylinder:
		ry := h * 0.12
		cx := x + w/2
		// Fill the body separately (no stroke — the rect's top/bottom
		// edges would sit on the rim equators and look like extra
		// horizontal lines closing each ellipse).
		if rs.FillR >= 0 {
			pdf.Rect(x, y+ry, w, h-ry*2, "F")
		}
		// Side walls: just the two vertical strokes.
		if rs.StrokeR >= 0 {
			pdf.Line(x, y+ry, x, y+h-ry)
			pdf.Line(x+w, y+ry, x+w, y+h-ry)
		}
		// Top rim: full ellipse (both curves visible).
		pdf.Ellipse(cx, y+ry, w/2, ry, 0, fillStyle)
		// Bottom: only the front-facing arc; the back is hidden by the
		// body. fpdf angles are CCW from 3 o'clock; in a y-down page
		// that means 0..180 sweeps right → bottom → left.
		pdf.Arc(cx, y+h-ry, w/2, ry, 0, 0, 180, "D")
	case displaylist.ShapeKindStateBullet:
		cx, cy := x+w/2, y+h/2
		r := w / 3
		if rs.StrokeR >= 0 {
			pdf.SetFillColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
		}
		pdf.Ellipse(cx, cy, r, r, 0, "F")
	case displaylist.ShapeKindStateBullseye:
		cx, cy := x+w/2, y+h/2
		rOuter := w / 2.2
		rInner := w / 4
		pdf.Ellipse(cx, cy, rOuter, rOuter, 0, "D")
		if rs.StrokeR >= 0 {
			pdf.SetFillColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
		}
		pdf.Ellipse(cx, cy, rInner, rInner, 0, "F")
	case displaylist.ShapeKindCustom:
		if len(s.Path) < 3 {
			return
		}
		pts := make([]fpdf.PointType, len(s.Path))
		for i, p := range s.Path {
			px, py := translatePoint(p, tr)
			pts[i] = fpdf.PointType{X: px, Y: py}
		}
		pdf.Polygon(pts, fillStyle)
	}
}

// drawCluster renders a Cluster backdrop + optional title above.
func drawCluster(pdf *fpdf.Fpdf, c displaylist.Cluster, tr func(displaylist.Rect) (float64, float64, float64, float64), bodyStyle, titleStyle RoleStyle) {
	x, y, w, h := tr(c.BBox)
	applyStroke(pdf, bodyStyle)
	fillStyle := applyFill(pdf, bodyStyle)
	pdf.Rect(x, y, w, h, fillStyle)
	if c.Title != "" {
		font := titleStyle.Font
		if font == "" {
			font = "Helvetica"
		}
		fontStyle := titleStyle.FontStyle
		if fontStyle == "" {
			fontStyle = "B"
		}
		fs := titleStyle.FontSize
		if fs <= 0 {
			fs = 10
		}
		pdf.SetFont(font, fontStyle, fs)
		pdf.SetTextColor(int(titleStyle.TextR), int(titleStyle.TextG), int(titleStyle.TextB))
		titleW := pdf.GetStringWidth(c.Title)
		pdf.Text(x+(w-titleW)/2, y+fs*0.45, c.Title)
	}
}

// drawMarker draws a standalone Marker glyph (rare).
func drawMarker(pdf *fpdf.Fpdf, m displaylist.Marker, tx func(displaylist.Point) (float64, float64), rs RoleStyle, scale float64) {
	cx, cy := tx(m.Pos)
	size := 2.5 * scale
	applyStroke(pdf, rs)
	switch m.Kind {
	case displaylist.MarkerArrow, displaylist.MarkerArrowOpen:
		pdf.SetFillColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
		fill := "F"
		if m.Kind == displaylist.MarkerArrowOpen {
			fill = "D"
		}
		pdf.Polygon([]fpdf.PointType{
			{X: cx, Y: cy},
			{X: cx - size, Y: cy - size*0.5},
			{X: cx - size, Y: cy + size*0.5},
		}, fill)
	case displaylist.MarkerCircleOpen:
		pdf.Ellipse(cx, cy, size*0.5, size*0.5, 0, "D")
	case displaylist.MarkerDiamondFilled, displaylist.MarkerDiamondOpen:
		fill := "D"
		if m.Kind == displaylist.MarkerDiamondFilled {
			pdf.SetFillColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
			fill = "F"
		}
		pdf.Polygon([]fpdf.PointType{
			{X: cx, Y: cy - size*0.5},
			{X: cx + size*0.7, Y: cy},
			{X: cx, Y: cy + size*0.5},
			{X: cx - size*0.7, Y: cy},
		}, fill)
	}
}

// translatePoint maps a Path point (in DisplayList absolute coords)
// to pdf coords using tr.
func translatePoint(p displaylist.Point, tr func(displaylist.Rect) (float64, float64, float64, float64)) (float64, float64) {
	x, y, _, _ := tr(displaylist.Rect{X: p.X, Y: p.Y, W: 0, H: 0})
	return x, y
}

func applyStroke(pdf *fpdf.Fpdf, rs RoleStyle) {
	if rs.StrokeR < 0 {
		pdf.SetLineWidth(0)
		return
	}
	pdf.SetDrawColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
	w := rs.StrokeWidth
	if w <= 0 {
		w = 0.3
	}
	pdf.SetLineWidth(w)
	if len(rs.DashPattern) >= 2 {
		pdf.SetDashPattern(rs.DashPattern, 0)
	} else {
		pdf.SetDashPattern(nil, 0)
	}
}

func applyFill(pdf *fpdf.Fpdf, rs RoleStyle) string {
	hasFill := rs.FillR >= 0
	hasStroke := rs.StrokeR >= 0
	if hasFill {
		pdf.SetFillColor(int(rs.FillR), int(rs.FillG), int(rs.FillB))
	}
	switch {
	case hasFill && hasStroke:
		return "FD"
	case hasFill:
		return "F"
	default:
		return "D"
	}
}

func minf(a, b, c float64) float64 {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
