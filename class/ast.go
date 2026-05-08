// Package class parses Mermaid `classDiagram` source and lays it out
// into a displaylist.DisplayList.
package class

// Visibility is the visibility marker on a class member.
type Visibility int

const (
	VisDefault Visibility = iota
	VisPublic
	VisPrivate
	VisProtected
	VisPackage
)

// RelationKind enumerates supported relationship variants.
type RelationKind int

const (
	RelInheritance RelationKind = iota // <|-- (or <|.. dashed)
	RelComposition                     // *--
	RelAggregation                     // o--
	RelAssociation                     // -->
	RelDependency                      // ..>
	RelRealization                     // ..|>
	RelLink                            // -- (no arrow)
)

// Member is one attribute or method on a class.
type Member struct {
	Visibility Visibility
	Name       string
	Type       string
	Args       string
	IsMethod   bool
	IsStatic   bool
	IsAbstract bool
}

// Class is a single class node.
type Class struct {
	ID         string
	Label      string
	Annotation string
	Members    []Member
	Namespace  string
}

// Relationship connects two classes.
type Relationship struct {
	From     string
	To       string
	Kind     RelationKind
	Label    string
	FromCard string
	ToCard   string
	Dashed   bool
}

// Namespace groups classes for cluster-style framing.
type Namespace struct {
	Name     string
	ClassIDs []string
}

// Diagram is the parsed class diagram.
type Diagram struct {
	Classes       []Class
	Relationships []Relationship
	Namespaces    []Namespace
}
