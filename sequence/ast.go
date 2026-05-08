// Package sequence parses Mermaid `sequenceDiagram` source into an
// AST and lays it out into a displaylist.DisplayList using a
// hand-rolled column/row layout (no autog).
package sequence

// ActorKind distinguishes participant boxes from `actor` stick figures.
type ActorKind int

const (
	ActorParticipant ActorKind = iota
	ActorPerson
)

// ArrowKind enumerates message arrow types.
type ArrowKind int

const (
	ArrowSync       ArrowKind = iota // ->>
	ArrowReply                       // -->>
	ArrowOpen                        // -)
	ArrowOpenDashed                  // --)
	ArrowSolid                       // -> (sync, no arrow head as in mermaid: tee)
	ArrowDashed                      // --> (dashed, no arrow head)
	ArrowCross                       // -x (lost)
	ArrowDashedCross                 // --x
)

// BlockType enumerates the supported block frame types.
type BlockType int

const (
	BlockLoop BlockType = iota
	BlockAlt
	BlockOpt
	BlockPar
	BlockCritical
	BlockBreak
	BlockRect
)

// NoteSide indicates note placement.
type NoteSide int

const (
	NoteLeftOf NoteSide = iota
	NoteRightOf
	NoteOver
)

// Actor is a sequence participant.
type Actor struct {
	ID    string
	Label string
	Kind  ActorKind
}

// Message is a single message exchange.
type Message struct {
	From       string
	To         string
	Label      string
	Arrow      ArrowKind
	Activate   bool // + on target
	Deactivate bool // - on source after message
}

// Note is a note placed near one or more actors.
type Note struct {
	Side   NoteSide
	Actors []string
	Text   string
}

// Block is a framed group of items (loop/alt/opt/par/critical/break/rect).
type Block struct {
	Type     BlockType
	Label    string
	Children []Item
	// Else holds alternative branches for alt/par/critical. Each
	// branch carries its own label and Children list.
	Else []*Block
}

// Activate / Deactivate are explicit standalone activation directives.
type Activate struct {
	Actor string
}

type Deactivate struct {
	Actor string
}

// Item is a top-level diagram element.
type Item interface{ isItem() }

func (*Message) isItem()    {}
func (*Note) isItem()       {}
func (*Block) isItem()      {}
func (*Activate) isItem()   {}
func (*Deactivate) isItem() {}

// Diagram is the parsed sequence diagram.
type Diagram struct {
	Actors []*Actor
	Items  []Item
}
