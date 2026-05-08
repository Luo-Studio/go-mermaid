package displaylist

// Point is a 2D coordinate in DisplayList units.
type Point struct {
	X, Y float64
}

// Rect is an axis-aligned rectangle. (X, Y) is the top-left corner.
// Coordinates grow right and down.
type Rect struct {
	X, Y, W, H float64
}

// Contains returns true if p lies inside r using a half-open
// convention (left/top inclusive, right/bottom exclusive).
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

// DisplayList is the layout-stage output. Width and Height bound all
// Items; (0, 0) is the top-left of the diagram.
type DisplayList struct {
	Width, Height float64
	Items         []Item
}

// Item is the closed sum type of DisplayList items.
type Item interface {
	itemKind() string
}

// Shape is a node-like geometry.
type Shape struct {
	Kind ShapeKind
	BBox Rect
	Path []Point
	Role Role
}

func (Shape) itemKind() string { return "shape" }

// Edge is a polyline with arrow markers at its endpoints.
type Edge struct {
	Points     []Point
	LineStyle  LineStyle
	ArrowStart MarkerKind
	ArrowEnd   MarkerKind
	Role       Role
}

func (Edge) itemKind() string { return "edge" }

// Text is a rendered string. Lines is the wrapped multi-line content.
type Text struct {
	Pos    Point
	Lines  []string
	Align  Align
	VAlign VAlign
	Role   Role
}

func (Text) itemKind() string { return "text" }

// Cluster is a backdrop rectangle for subgraphs, sequence blocks,
// state composite states, and similar.
type Cluster struct {
	BBox  Rect
	Title string
	Role  Role
}

func (Cluster) itemKind() string { return "cluster" }

// Marker is a standalone arrow head, diamond, or cardinality glyph.
type Marker struct {
	Pos   Point
	Angle float64
	Kind  MarkerKind
	Role  Role
}

func (Marker) itemKind() string { return "marker" }
