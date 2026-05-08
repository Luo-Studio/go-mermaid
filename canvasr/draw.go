package mermaidcanvasr

import (
	"image/color"
	"math"

	"github.com/tdewolff/canvas"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func applyStrokeFill(ctx *canvas.Context, rs RoleStyle) (hasStroke, hasFill bool) {
	hasStroke = rs.Stroke != nil
	hasFill = rs.Fill != nil
	if hasStroke {
		ctx.SetStrokeColor(rs.Stroke)
		w := rs.StrokeWidth
		if w <= 0 {
			w = 0.3
		}
		ctx.SetStrokeWidth(w)
		if len(rs.DashPattern) >= 2 {
			ctx.SetDashes(0, rs.DashPattern...)
		} else {
			ctx.SetDashes(0)
		}
	} else {
		ctx.SetStrokeColor(color.Transparent)
	}
	if hasFill {
		ctx.SetFillColor(rs.Fill)
	} else {
		ctx.SetFillColor(color.Transparent)
	}
	return
}

func drawShape(ctx *canvas.Context, s displaylist.Shape, rs RoleStyle) {
	applyStrokeFill(ctx, rs)
	x, y, w, h := s.BBox.X, s.BBox.Y, s.BBox.W, s.BBox.H

	switch s.Kind {
	case displaylist.ShapeKindRect, displaylist.ShapeKindSubroutine:
		ctx.DrawPath(x, y, canvas.Rectangle(w, h))
		if s.Kind == displaylist.ShapeKindSubroutine {
			ctx.SetFillColor(color.Transparent)
			ctx.DrawPath(x+4, y, canvas.Line(0, h))
			ctx.DrawPath(x+w-4, y, canvas.Line(0, h))
		}
	case displaylist.ShapeKindRound:
		r := minf(8, w/2, h/2)
		ctx.DrawPath(x, y, canvas.RoundedRectangle(w, h, r))
	case displaylist.ShapeKindStadium:
		r := h / 2
		ctx.DrawPath(x, y, canvas.RoundedRectangle(w, h, r))
	case displaylist.ShapeKindCircle, displaylist.ShapeKindDoubleCircle:
		cx, cy := x+w/2, y+h/2
		r := w / 2
		ctx.DrawPath(cx, cy, canvas.Ellipse(r, r))
		if s.Kind == displaylist.ShapeKindDoubleCircle {
			ctx.SetFillColor(color.Transparent)
			ctx.DrawPath(cx, cy, canvas.Ellipse(r-2, r-2))
		}
	case displaylist.ShapeKindEllipse:
		ctx.DrawPath(x+w/2, y+h/2, canvas.Ellipse(w/2, h/2))
	case displaylist.ShapeKindDiamond:
		p := &canvas.Path{}
		p.MoveTo(w/2, 0)
		p.LineTo(w, h/2)
		p.LineTo(w/2, h)
		p.LineTo(0, h/2)
		p.Close()
		ctx.DrawPath(x, y, p)
	case displaylist.ShapeKindHexagon:
		off := w * 0.18
		p := &canvas.Path{}
		p.MoveTo(off, 0)
		p.LineTo(w-off, 0)
		p.LineTo(w, h/2)
		p.LineTo(w-off, h)
		p.LineTo(off, h)
		p.LineTo(0, h/2)
		p.Close()
		ctx.DrawPath(x, y, p)
	case displaylist.ShapeKindCylinder:
		ry := h * 0.12
		cx := x + w/2
		// Body rect.
		ctx.DrawPath(x, y+ry, canvas.Rectangle(w, h-ry*2))
		// Top: full ellipse (rim).
		ctx.DrawPath(cx, y+ry, canvas.Ellipse(w/2, ry))
		// Bottom: front-arc only. The back of the bottom is hidden by
		// the body. We use a quadratic Bezier from (-rx, 0) through
		// (0, ry) to (rx, 0) — coordinate-system-agnostic. Control
		// point at (0, 2*ry) places the curve midpoint exactly at
		// (0, ry).
		arc := &canvas.Path{}
		arc.MoveTo(-w/2, 0)
		arc.QuadTo(0, 2*ry, w/2, 0)
		ctx.SetFillColor(color.Transparent)
		ctx.DrawPath(cx, y+h-ry, arc)
	case displaylist.ShapeKindStateBullet:
		cx, cy := x+w/2, y+h/2
		r := w / 3
		// Filled bullet: stroke color used for fill
		strokeForFill := rs.Stroke
		if strokeForFill == nil {
			strokeForFill = color.Black
		}
		ctx.SetFillColor(strokeForFill)
		ctx.DrawPath(cx, cy, canvas.Ellipse(r, r))
	case displaylist.ShapeKindStateBullseye:
		cx, cy := x+w/2, y+h/2
		rOuter := w / 2.2
		rInner := w / 4
		ctx.SetFillColor(color.Transparent)
		ctx.DrawPath(cx, cy, canvas.Ellipse(rOuter, rOuter))
		strokeForFill := rs.Stroke
		if strokeForFill == nil {
			strokeForFill = color.Black
		}
		ctx.SetFillColor(strokeForFill)
		ctx.DrawPath(cx, cy, canvas.Ellipse(rInner, rInner))
	case displaylist.ShapeKindCustom:
		if len(s.Path) < 3 {
			return
		}
		p := &canvas.Path{}
		p.MoveTo(s.Path[0].X-x, s.Path[0].Y-y)
		for _, pt := range s.Path[1:] {
			p.LineTo(pt.X-x, pt.Y-y)
		}
		p.Close()
		ctx.DrawPath(x, y, p)
	}
}

func drawCluster(ctx *canvas.Context, c displaylist.Cluster, st Style) {
	rs := st.lookup(c.Role)
	applyStrokeFill(ctx, rs)
	ctx.DrawPath(c.BBox.X, c.BBox.Y, canvas.Rectangle(c.BBox.W, c.BBox.H))
	if c.Title != "" && st.FontFamily != nil {
		titleStyle := st.lookup(displaylist.RoleClusterTitle)
		drawTextLine(ctx, c.BBox.X+4, c.BBox.Y+titleStyle.FontSize*0.45, c.Title, st.FontFamily, titleStyle, displaylist.AlignLeft)
	}
}

func drawEdge(ctx *canvas.Context, e displaylist.Edge, rs RoleStyle) {
	if len(e.Points) < 2 {
		return
	}
	applyStrokeFill(ctx, rs)
	switch e.LineStyle {
	case displaylist.LineStyleDashed:
		ctx.SetDashes(0, 1.5, 1.5)
		defer ctx.SetDashes(0)
	case displaylist.LineStyleDotted:
		ctx.SetDashes(0, 0.4, 1.0)
		defer ctx.SetDashes(0)
	case displaylist.LineStyleThick:
		w := rs.StrokeWidth * 2.5
		if w < 0.6 {
			w = 0.6
		}
		ctx.SetStrokeWidth(w)
	}
	p := &canvas.Path{}
	p.MoveTo(0, 0)
	prev := e.Points[0]
	for _, pt := range e.Points[1:] {
		p.LineTo(pt.X-prev.X, pt.Y-prev.Y)
	}
	ctx.SetFillColor(color.Transparent)
	ctx.DrawPath(prev.X, prev.Y, p)

	if e.ArrowEnd != displaylist.MarkerNone {
		drawArrow(ctx, e.Points[len(e.Points)-1], e.Points[len(e.Points)-2], e.ArrowEnd, rs)
	}
	if e.ArrowStart != displaylist.MarkerNone {
		drawArrow(ctx, e.Points[0], e.Points[1], e.ArrowStart, rs)
	}
}

func drawArrow(ctx *canvas.Context, tip, behind displaylist.Point, kind displaylist.MarkerKind, rs RoleStyle) {
	dx, dy := tip.X-behind.X, tip.Y-behind.Y
	d := math.Hypot(dx, dy)
	if d == 0 {
		return
	}
	ux, uy := dx/d, dy/d
	px, py := -uy, ux
	size := 2.5
	strokeForFill := rs.Stroke
	if strokeForFill == nil {
		strokeForFill = color.Black
	}
	ctx.SetFillColor(strokeForFill)

	switch kind {
	case displaylist.MarkerArrow, displaylist.MarkerArrowOpen, displaylist.MarkerTriangleOpen:
		bx := tip.X - ux*size
		by := tip.Y - uy*size
		p := &canvas.Path{}
		p.MoveTo(0, 0)
		p.LineTo(bx+px*size*0.5-tip.X, by+py*size*0.5-tip.Y)
		p.LineTo(bx-px*size*0.5-tip.X, by-py*size*0.5-tip.Y)
		p.Close()
		if kind != displaylist.MarkerArrow {
			ctx.SetFillColor(color.Transparent)
		}
		ctx.DrawPath(tip.X, tip.Y, p)
	case displaylist.MarkerDiamondFilled, displaylist.MarkerDiamondOpen:
		bx := tip.X - ux*size*1.6
		by := tip.Y - uy*size*1.6
		mx := tip.X - ux*size*0.8
		my := tip.Y - uy*size*0.8
		p := &canvas.Path{}
		p.MoveTo(0, 0)
		p.LineTo(mx+px*size*0.5-tip.X, my+py*size*0.5-tip.Y)
		p.LineTo(bx-tip.X, by-tip.Y)
		p.LineTo(mx-px*size*0.5-tip.X, my-py*size*0.5-tip.Y)
		p.Close()
		if kind == displaylist.MarkerDiamondOpen {
			ctx.SetFillColor(color.Transparent)
		}
		ctx.DrawPath(tip.X, tip.Y, p)
	case displaylist.MarkerCardinalityOne:
		drawCardOne(ctx, tip, ux, uy, px, py, size)
	case displaylist.MarkerCardinalityZeroOrOne:
		drawCardOne(ctx, tip, ux, uy, px, py, size)
		ctx.SetFillColor(color.Transparent)
		cx := tip.X - ux*size*1.6
		cy := tip.Y - uy*size*1.6
		ctx.DrawPath(cx, cy, canvas.Ellipse(size*0.35, size*0.35))
	case displaylist.MarkerCardinalityOneOrMore:
		drawCardOne(ctx, tip, ux, uy, px, py, size)
		drawCrowsFoot(ctx, tip, ux, uy, px, py, size)
	case displaylist.MarkerCardinalityZeroOrMore:
		ctx.SetFillColor(color.Transparent)
		cx := tip.X - ux*size*1.6
		cy := tip.Y - uy*size*1.6
		ctx.DrawPath(cx, cy, canvas.Ellipse(size*0.35, size*0.35))
		drawCrowsFoot(ctx, tip, ux, uy, px, py, size)
	}
}

func drawCardOne(ctx *canvas.Context, tip displaylist.Point, ux, uy, px, py, size float64) {
	bx := tip.X - ux*size*0.7
	by := tip.Y - uy*size*0.7
	p := &canvas.Path{}
	p.MoveTo(0, 0)
	p.LineTo((bx-px*size*0.5)-(bx+px*size*0.5), (by-py*size*0.5)-(by+py*size*0.5))
	ctx.SetFillColor(color.Transparent)
	ctx.DrawPath(bx+px*size*0.5, by+py*size*0.5, p)
}

func drawCrowsFoot(ctx *canvas.Context, tip displaylist.Point, ux, uy, px, py, size float64) {
	for _, off := range []float64{-0.55, 0.0, 0.55} {
		endX := tip.X - ux*size + px*off*size
		endY := tip.Y - uy*size + py*off*size
		p := &canvas.Path{}
		p.MoveTo(0, 0)
		p.LineTo(endX-tip.X, endY-tip.Y)
		ctx.SetFillColor(color.Transparent)
		ctx.DrawPath(tip.X, tip.Y, p)
	}
}

func drawText(ctx *canvas.Context, t displaylist.Text, st Style) {
	if len(t.Lines) == 0 || st.FontFamily == nil {
		return
	}
	rs := st.lookup(t.Role)
	if rs.FontSize <= 0 {
		rs.FontSize = 10
	}
	if rs.TextColor == nil {
		rs.TextColor = color.Black
	}
	lineH := rs.FontSize * 0.4
	totalH := lineH * float64(len(t.Lines))
	var startY float64
	switch t.VAlign {
	case displaylist.VAlignTop:
		startY = t.Pos.Y + lineH*0.7
	case displaylist.VAlignBottom:
		startY = t.Pos.Y - totalH + lineH*0.7
	default:
		startY = t.Pos.Y - totalH/2 + lineH*0.7
	}
	for i, line := range t.Lines {
		drawTextLine(ctx, t.Pos.X, startY+float64(i)*lineH, line, st.FontFamily, rs, t.Align)
	}
}

func drawTextLine(ctx *canvas.Context, x, y float64, text string, fam *canvas.FontFamily, rs RoleStyle, align displaylist.Align) {
	face := fam.Face(rs.FontSize, rs.TextColor, rs.FontStyle)
	tl := canvas.NewTextLine(face, text, canvas.Left)
	w := tl.Width
	if w <= 0 {
		w = face.TextWidth(text)
	}
	switch align {
	case displaylist.AlignLeft:
		// no shift
	case displaylist.AlignRight:
		x -= w
	default:
		x -= w / 2
	}
	ctx.DrawText(x, y, tl)
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
