// Package flowchart parses Mermaid `flowchart`/`graph` source into an
// AST and lays it out into a displaylist.DisplayList.
package flowchart

// Direction is the diagram's flow direction.
type Direction int

const (
	DirectionTB Direction = iota
	DirectionBT
	DirectionLR
	DirectionRL
)

func (d Direction) String() string {
	switch d {
	case DirectionBT:
		return "BT"
	case DirectionLR:
		return "LR"
	case DirectionRL:
		return "RL"
	default:
		return "TB"
	}
}

// NodeShape mirrors mermaid's node-shape syntax.
type NodeShape int

const (
	ShapeRect NodeShape = iota
	ShapeRound
	ShapeStadium
	ShapeDiamond
	ShapeCircle
	ShapeDoubleCircle
	ShapeSubroutine
	ShapeCylinder
	ShapeHexagon
	ShapeAsymmetric
	ShapeTrapezoid
	ShapeTrapezoidAlt
	ShapeParallelogram
	ShapeParallelogramAlt
	ShapeStateStart
	ShapeStateEnd
)

// EdgeStyle is the line style of an edge.
type EdgeStyle int

const (
	EdgeSolid EdgeStyle = iota
	EdgeDotted
	EdgeThick
)

// Node is a single node in the flowchart.
type Node struct {
	ID         string
	Label      string
	Shape      NodeShape
	ClassNames []string
}

// Edge is a connection between two nodes.
type Edge struct {
	From       string
	To         string
	Label      string
	Style      EdgeStyle
	ArrowStart bool
	ArrowEnd   bool
}

// Subgraph is a `subgraph X ... end` block.
type Subgraph struct {
	ID        string
	Label     string
	NodeIDs   []string
	Children  []*Subgraph
	Direction Direction
}

// ClassDef is a `classDef name property:value;...` declaration.
type ClassDef struct {
	Name       string
	Properties map[string]string
}

// Diagram is the parsed flowchart.
type Diagram struct {
	Direction  Direction
	Nodes      []Node
	Edges      []Edge
	Subgraphs  []*Subgraph
	ClassDefs  []ClassDef
	NodeStyles map[string]map[string]string
}
