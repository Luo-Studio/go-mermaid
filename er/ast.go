// Package er parses Mermaid `erDiagram` source and lays it out into
// a displaylist.DisplayList.
package er

// Cardinality enumerates ER cardinality variants.
type Cardinality int

const (
	CardZeroOrOne Cardinality = iota
	CardExactlyOne
	CardZeroOrMore
	CardOneOrMore
)

// AttrKey marks an attribute as a primary, foreign, or unique key.
type AttrKey int

const (
	KeyNone AttrKey = iota
	KeyPrimary
	KeyForeign
	KeyUnique
)

// Attribute is one row of an entity definition.
type Attribute struct {
	Type    string
	Name    string
	Key     AttrKey
	Comment string
}

// Entity is one ER entity (table-like).
type Entity struct {
	ID         string
	Attributes []Attribute
}

// Relationship connects two entities with cardinality at each side.
type Relationship struct {
	Left             string
	Right            string
	LeftCardinality  Cardinality
	RightCardinality Cardinality
	Identifying      bool
	Label            string
}

// Diagram is the parsed ER diagram.
type Diagram struct {
	Entities      []*Entity
	Relationships []*Relationship
}
