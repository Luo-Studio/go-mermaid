package flowchart

import "github.com/luo-studio/go-mermaid/displaylist"

// shapeKind maps an AST NodeShape to a DisplayList ShapeKind. Some
// shapes have no native primitive; those return ShapeKindCustom and
// the caller populates Path.
func shapeKind(s NodeShape) displaylist.ShapeKind {
	switch s {
	case ShapeRect:
		return displaylist.ShapeKindRect
	case ShapeSubroutine:
		return displaylist.ShapeKindSubroutine
	case ShapeRound:
		return displaylist.ShapeKindRound
	case ShapeStadium:
		return displaylist.ShapeKindStadium
	case ShapeDiamond:
		return displaylist.ShapeKindDiamond
	case ShapeCircle:
		return displaylist.ShapeKindCircle
	case ShapeDoubleCircle:
		return displaylist.ShapeKindDoubleCircle
	case ShapeHexagon:
		return displaylist.ShapeKindHexagon
	case ShapeCylinder:
		return displaylist.ShapeKindCylinder
	default:
		return displaylist.ShapeKindCustom
	}
}

// nodeSize returns the bbox (W, H) for a node given its label
// dimensions and shape. Padding varies by shape — rounder shapes need
// more breathing room than rectangles.
func nodeSize(shape NodeShape, labelW, labelH float64) (w, h float64) {
	pad := 16.0
	switch shape {
	case ShapeDiamond, ShapeCircle, ShapeDoubleCircle, ShapeHexagon:
		pad = 24
	case ShapeStadium, ShapeRound:
		pad = 18
	}
	w = labelW + pad*2
	h = labelH + pad
	if h < 28 {
		h = 28
	}
	if shape == ShapeCircle || shape == ShapeDoubleCircle {
		side := w
		if h > side {
			side = h
		}
		return side, side
	}
	return w, h
}

// customPath returns a polygon path for shapes without a native
// DisplayList kind.
func customPath(shape NodeShape, b displaylist.Rect) []displaylist.Point {
	switch shape {
	case ShapeAsymmetric:
		return []displaylist.Point{
			{X: b.X, Y: b.Y + b.H/2},
			{X: b.X + b.W*0.15, Y: b.Y},
			{X: b.X + b.W, Y: b.Y},
			{X: b.X + b.W, Y: b.Y + b.H},
			{X: b.X + b.W*0.15, Y: b.Y + b.H},
		}
	case ShapeTrapezoid:
		off := b.W * 0.12
		return []displaylist.Point{
			{X: b.X + off, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y},
			{X: b.X + b.W, Y: b.Y + b.H},
			{X: b.X, Y: b.Y + b.H},
		}
	case ShapeTrapezoidAlt:
		off := b.W * 0.12
		return []displaylist.Point{
			{X: b.X, Y: b.Y},
			{X: b.X + b.W, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y + b.H},
			{X: b.X + off, Y: b.Y + b.H},
		}
	case ShapeParallelogram:
		off := b.W * 0.18
		return []displaylist.Point{
			{X: b.X + off, Y: b.Y},
			{X: b.X + b.W, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y + b.H},
			{X: b.X, Y: b.Y + b.H},
		}
	case ShapeParallelogramAlt:
		off := b.W * 0.18
		return []displaylist.Point{
			{X: b.X, Y: b.Y},
			{X: b.X + b.W - off, Y: b.Y},
			{X: b.X + b.W, Y: b.Y + b.H},
			{X: b.X + off, Y: b.Y + b.H},
		}
	}
	return []displaylist.Point{
		{X: b.X, Y: b.Y},
		{X: b.X + b.W, Y: b.Y},
		{X: b.X + b.W, Y: b.Y + b.H},
		{X: b.X, Y: b.Y + b.H},
	}
}
