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

	// autog's spline routing returns each edge as a piece-wise
	// cubic Bézier formatted as 4*K points: each consecutive group
	// of four points (start, ctrl1, ctrl2, end) is one cubic Bézier
	// segment, and consecutive segments come back with a duplicated
	// shared endpoint (the end of segment i ≈ the start of segment
	// i+1). Render each 4-tuple as one CurveBezierCubicTo so the
	// edge follows the obstacle-aware path autog computed.
	if len(e.Points) >= 4 && len(e.Points)%4 == 0 {
		x0, y0 := tx(e.Points[0])
		pdf.MoveTo(x0, y0)
		for i := 0; i < len(e.Points); i += 4 {
			c1x, c1y := tx(e.Points[i+1])
			c2x, c2y := tx(e.Points[i+2])
			ex, ey := tx(e.Points[i+3])
			pdf.CurveBezierCubicTo(c1x, c1y, c2x, c2y, ex, ey)
			// Skip the duplicated shared endpoint at the start of the
			// next segment by advancing past the (start, ...) pair —
			// loop step += 4 already handles that since each segment
			// is exactly 4 points.
		}
		pdf.DrawPath("D")
	} else {
		// Fallback for edges that didn't come through the spline
		// pipeline (malformed polylines, the rare 2-point case).
		x0, y0 := tx(e.Points[0])
		for _, p := range e.Points[1:] {
			x1, y1 := tx(p)
			pdf.Line(x0, y0, x1, y1)
			x0, y0 = x1, y1
		}
	}

	if e.ArrowEnd != displaylist.MarkerNone {
		tip := e.Points[len(e.Points)-1]
		behind := lastDistinct(e.Points, len(e.Points)-2, -1, tip)
		drawArrow(pdf, tip, behind, e.ArrowEnd, tx, rs, scale)
	}
	if e.ArrowStart != displaylist.MarkerNone {
		tip := e.Points[0]
		behind := lastDistinct(e.Points, 1, 1, tip)
		drawArrow(pdf, tip, behind, e.ArrowStart, tx, rs, scale)
	}
}

// lastDistinct walks the polyline starting at `start` (stepping by
// `step`, +1 or -1) and returns the first point whose position
// differs from `tip`. Used to pick a "behind" point for arrow
// direction when the endpoint coincides with adjacent control
// points (which happens with autog's spline output where the
// final cubic segment's ctrl2 sits exactly at the endpoint).
func lastDistinct(pts []displaylist.Point, start, step int, tip displaylist.Point) displaylist.Point {
	for i := start; i >= 0 && i < len(pts); i += step {
		if pts[i] != tip {
			return pts[i]
		}
	}
	return pts[start]
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
