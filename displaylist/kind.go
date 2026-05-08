// Package displaylist defines the cross-emitter intermediate
// representation produced by go-mermaid's layout stage.
//
// Layout produces a DisplayList of Items (Shape, Edge, Text, Cluster,
// Marker). Items carry a semantic Role; emitters look up colors,
// fonts, and line widths in a caller-supplied per-Role style map.
// The DisplayList itself is style-neutral.
package displaylist

// ShapeKind identifies the geometry of a Shape item. Common kinds are
// rendered natively by emitters; ShapeKindCustom carries an explicit
// polygon path for arbitrary shapes (trapezoid, parallelogram, ...).
type ShapeKind string

const (
	ShapeKindRect         ShapeKind = "rect"
	ShapeKindRound        ShapeKind = "round"
	ShapeKindStadium      ShapeKind = "stadium"
	ShapeKindDiamond      ShapeKind = "diamond"
	ShapeKindCircle       ShapeKind = "circle"
	ShapeKindDoubleCircle ShapeKind = "doubleCircle"
	ShapeKindEllipse      ShapeKind = "ellipse"
	ShapeKindHexagon      ShapeKind = "hexagon"
	ShapeKindCylinder     ShapeKind = "cylinder"
	ShapeKindSubroutine   ShapeKind = "subroutine"
	ShapeKindCustom       ShapeKind = "custom"

	// State-diagram pseudostates. Bullet = filled disk, Bullseye =
	// disk inside a ring (UML "final" state).
	ShapeKindStateBullet   ShapeKind = "stateBullet"
	ShapeKindStateBullseye ShapeKind = "stateBullseye"
)

// LineStyle identifies the stroke style of an Edge or Cluster border.
type LineStyle string

const (
	LineStyleSolid  LineStyle = "solid"
	LineStyleDashed LineStyle = "dashed"
	LineStyleThick  LineStyle = "thick"
	LineStyleDotted LineStyle = "dotted"
)

// MarkerKind identifies the kind of arrow head or relationship marker
// at an Edge endpoint.
type MarkerKind string

const (
	MarkerNone                  MarkerKind = ""
	MarkerArrow                 MarkerKind = "arrow"
	MarkerArrowOpen             MarkerKind = "arrowOpen"
	MarkerDiamondFilled         MarkerKind = "diamondFilled"
	MarkerDiamondOpen           MarkerKind = "diamondOpen"
	MarkerTriangleOpen          MarkerKind = "triangleOpen"
	MarkerCross                 MarkerKind = "cross"
	MarkerCircleOpen            MarkerKind = "circleOpen"
	MarkerCardinalityOne        MarkerKind = "cardOne"
	MarkerCardinalityZeroOrOne  MarkerKind = "cardZeroOrOne"
	MarkerCardinalityOneOrMore  MarkerKind = "cardOneOrMore"
	MarkerCardinalityZeroOrMore MarkerKind = "cardZeroOrMore"
)

// Align is horizontal text alignment.
type Align string

const (
	AlignLeft   Align = "left"
	AlignCenter Align = "center"
	AlignRight  Align = "right"
)

// VAlign is vertical text alignment.
type VAlign string

const (
	VAlignTop      VAlign = "top"
	VAlignMiddle   VAlign = "middle"
	VAlignBaseline VAlign = "baseline"
	VAlignBottom   VAlign = "bottom"
)
