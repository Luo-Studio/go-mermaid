// Package state parses Mermaid `stateDiagram` / `stateDiagram-v2`
// source and lays it out into a displaylist.DisplayList.
package state

// State is a single state node, including pseudostates synthesized
// from `[*]` source-position references.
type State struct {
	ID      string
	Label   string
	IsStart bool
	IsEnd   bool
	Note    string
}

// Transition is a directed edge between two states.
type Transition struct {
	From  string
	To    string
	Label string
}

// Composite is a state-diagram cluster (`state X { ... }` block).
type Composite struct {
	ID       string
	Label    string
	StateIDs []string
	Children []*Composite
}

// Diagram is the parsed state diagram.
type Diagram struct {
	States      []State
	Transitions []Transition
	Composites  []*Composite
}
