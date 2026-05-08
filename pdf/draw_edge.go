package mermaidpdf

import (
	"math"

	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func drawEdge(pdf *fpdf.Fpdf, e displaylist.Edge, tx func(displaylist.Point) (float64, float64), rs RoleStyle, scale float64) {
	if len(e.Points) < 2 {
		return
	}
	applyStroke(pdf, rs)
	switch e.LineStyle {
	case displaylist.LineStyleDashed:
		pdf.SetDashPattern([]float64{1.5, 1.5}, 0)
		defer pdf.SetDashPattern(nil, 0)
	case displaylist.LineStyleDotted:
		pdf.SetDashPattern([]float64{0.4, 1.0}, 0)
		defer pdf.SetDashPattern(nil, 0)
	case displaylist.LineStyleThick:
		w := rs.StrokeWidth * 2.5
		if w < 0.6 {
			w = 0.6
		}
		pdf.SetLineWidth(w)
		defer pdf.SetLineWidth(rs.StrokeWidth)
	}

	x0, y0 := tx(e.Points[0])
	for _, p := range e.Points[1:] {
		x1, y1 := tx(p)
		pdf.Line(x0, y0, x1, y1)
		x0, y0 = x1, y1
	}

	if e.ArrowEnd != displaylist.MarkerNone {
		drawArrow(pdf, e.Points[len(e.Points)-1], e.Points[len(e.Points)-2], e.ArrowEnd, tx, rs, scale)
	}
	if e.ArrowStart != displaylist.MarkerNone {
		drawArrow(pdf, e.Points[0], e.Points[1], e.ArrowStart, tx, rs, scale)
	}
}

// drawArrow renders an arrow marker at `tip` pointing away from `behind`.
func drawArrow(pdf *fpdf.Fpdf, tip, behind displaylist.Point, kind displaylist.MarkerKind, tx func(displaylist.Point) (float64, float64), rs RoleStyle, scale float64) {
	tx0, ty0 := tx(behind)
	tx1, ty1 := tx(tip)
	dx, dy := tx1-tx0, ty1-ty0
	d := math.Hypot(dx, dy)
	if d == 0 {
		return
	}
	ux, uy := dx/d, dy/d
	px, py := -uy, ux

	size := 2.5 * scale
	if size < 1.2 {
		size = 1.2
	}
	if rs.StrokeR >= 0 {
		pdf.SetFillColor(int(rs.StrokeR), int(rs.StrokeG), int(rs.StrokeB))
	}
	switch kind {
	case displaylist.MarkerArrow, displaylist.MarkerArrowOpen,
		displaylist.MarkerTriangleOpen:
		bx := tx1 - ux*size
		by := ty1 - uy*size
		l := fpdf.PointType{X: bx + px*size*0.5, Y: by + py*size*0.5}
		r := fpdf.PointType{X: bx - px*size*0.5, Y: by - py*size*0.5}
		fill := "F"
		if kind != displaylist.MarkerArrow {
			fill = "D"
		}
		pdf.Polygon([]fpdf.PointType{
			{X: tx1, Y: ty1}, l, r,
		}, fill)
	case displaylist.MarkerDiamondFilled, displaylist.MarkerDiamondOpen:
		bx := tx1 - ux*size*1.5
		by := ty1 - uy*size*1.5
		mx := tx1 - ux*size*0.75
		my := ty1 - uy*size*0.75
		l := fpdf.PointType{X: mx + px*size*0.5, Y: my + py*size*0.5}
		r := fpdf.PointType{X: mx - px*size*0.5, Y: my - py*size*0.5}
		fill := "F"
		if kind == displaylist.MarkerDiamondOpen {
			fill = "D"
		}
		pdf.Polygon([]fpdf.PointType{
			{X: tx1, Y: ty1}, l, {X: bx, Y: by}, r,
		}, fill)
	case displaylist.MarkerCross:
		s := size * 0.5
		pdf.Line(tx1-px*s-ux*s, ty1-py*s-uy*s, tx1+px*s+ux*s, ty1+py*s+uy*s)
		pdf.Line(tx1+px*s-ux*s, ty1+py*s-uy*s, tx1-px*s+ux*s, ty1-py*s+uy*s)
	case displaylist.MarkerCircleOpen:
		pdf.Ellipse(tx1-ux*size*0.5, ty1-uy*size*0.5, size*0.4, size*0.4, 0, "D")
	case displaylist.MarkerCardinalityOne:
		drawCardOne(pdf, tx1, ty1, ux, uy, px, py, size)
	case displaylist.MarkerCardinalityZeroOrOne:
		drawCardOne(pdf, tx1, ty1, ux, uy, px, py, size)
		cx := tx1 - ux*size*1.6
		cy := ty1 - uy*size*1.6
		pdf.Ellipse(cx, cy, size*0.35, size*0.35, 0, "D")
	case displaylist.MarkerCardinalityOneOrMore:
		drawCardOne(pdf, tx1, ty1, ux, uy, px, py, size)
		drawCrowsFoot(pdf, tx1, ty1, ux, uy, px, py, size)
	case displaylist.MarkerCardinalityZeroOrMore:
		cx := tx1 - ux*size*1.6
		cy := ty1 - uy*size*1.6
		pdf.Ellipse(cx, cy, size*0.35, size*0.35, 0, "D")
		drawCrowsFoot(pdf, tx1, ty1, ux, uy, px, py, size)
	}
}

// drawCardOne renders a single short tick across the line, used for
// the ER cardinality `||` (exactly one) glyph and as the inner half
// of `|o`/`}|` glyphs.
func drawCardOne(pdf *fpdf.Fpdf, tx1, ty1, ux, uy, px, py, size float64) {
	bx := tx1 - ux*size*0.7
	by := ty1 - uy*size*0.7
	pdf.Line(bx+px*size*0.5, by+py*size*0.5, bx-px*size*0.5, by-py*size*0.5)
}

// drawCrowsFoot draws three short lines fanning back from the tip,
// the "many" half of ER cardinality glyphs.
func drawCrowsFoot(pdf *fpdf.Fpdf, tx1, ty1, ux, uy, px, py, size float64) {
	for _, off := range []float64{-0.55, 0.0, 0.55} {
		// End point: angled away from the line by `off` perpendicular
		// units and `size` along the line.
		endX := tx1 - ux*size + px*off*size
		endY := ty1 - uy*size + py*off*size
		pdf.Line(tx1, ty1, endX, endY)
	}
}
