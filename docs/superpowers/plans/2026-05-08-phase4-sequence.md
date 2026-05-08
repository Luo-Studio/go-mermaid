# go-mermaid Phase 4 — Sequence Diagrams

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development.

**Goal:** Add full sequence-diagram support: parser, hand-rolled column/row layout (no autog), DisplayList emit, and the matching PDF emitter extensions for sequence-specific roles.

**Architecture:** The `sequence` package owns parser + AST + layout. Parsing is line-by-line regex. Layout is hand-rolled in two passes: assign each actor an X column, then walk the message timeline assigning Y rows. Block frames (`loop`/`alt`/`opt`/`par`/`critical`/`break`/`rect`) emit `displaylist.Cluster` items spanning their start..end rows. Notes are clusters with their own role. Activations are small rects on lifelines. Messages are edges between actor lifelines with a label above.

**Depends on:** Phase 1, 2, 3 plans complete.

---

## Spec Reference

Spec section "Sequence Feature Coverage" + "Sequence Diagram Layout".

## File Structure

```
go-mermaid/
├── sequence/
│   ├── ast.go                  # Diagram, Actor, Message, Block, Note, BlockType, ArrowKind
│   ├── parser.go               # Parse(src)
│   ├── parser_test.go
│   ├── layout.go               # Layout(ast, opts) → DisplayList
│   ├── layout_test.go
│   └── golden_test.go
├── pdf/
│   ├── style.go                # MODIFIED: add sequence role defaults
│   └── emit.go                 # Already handles Shape/Edge/Text/Cluster; sequence reuses
├── mermaid.go                  # MODIFIED: dispatch typeSequence
└── testdata/sequence/{parse,layout}/...
```

## Tasks

### Task 1: AST

**Files:** Create `sequence/ast.go`.

```go
package sequence

type ActorKind int
const (
	ActorParticipant ActorKind = iota
	ActorPerson // mermaid `actor` keyword — stick figure
)

type ArrowKind int
const (
	ArrowSync       ArrowKind = iota // ->>
	ArrowReply                        // -->>
	ArrowOpen                         // -)
	ArrowOpenDashed                   // --)
)

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

type NoteSide int
const (
	NoteLeftOf NoteSide = iota
	NoteRightOf
	NoteOver
)

type Actor struct {
	ID    string
	Label string
	Kind  ActorKind
}

type Message struct {
	From       string
	To         string
	Label      string
	Arrow      ArrowKind
	Activate   bool // + on target
	Deactivate bool // - on source
}

type Note struct {
	Side    NoteSide
	Actors  []string // 1 for left/right, 2 for over (range)
	Text    string
	Index   int      // index into the sequence (between which two messages)
}

type Block struct {
	Type     BlockType
	Label    string
	Children []Item // nested messages/notes/blocks
	Else     []*Block // alt/par dividers
}

// Item is a top-level diagram element. One of *Message | *Note | *Block.
type Item interface{ isItem() }

func (*Message) isItem() {}
func (*Note) isItem()    {}
func (*Block) isItem()   {}

type Diagram struct {
	Actors []*Actor
	Items  []Item
}
```

- [ ] Test, implement, commit: `git commit -m "sequence: AST types"`

---

### Task 2: Parser

**Files:** Create `sequence/parser.go`, `sequence/parser_test.go`.

Implementation outline:
- Preprocess (strip comments, drop blanks).
- First line must be `sequenceDiagram`.
- Parse top-down with a stack of `*Block` for `loop ... end`, `alt ... else ... end`, `opt ... end`, `par ... and ... end`, `critical ... option ... end`, `break ... end`, `rect ... end`.
- Each non-block line is one of: `participant X as L`, `actor X as L`, `Note (left of|right of|over) X[,Y]: text`, message line.

Message line regex (try each arrow in order):
```
^(\w+)\s*(->>|-->>|-\)|--\))\s*([+-]?)(\w+)\s*:\s*(.*)$
```

Where the optional `+` / `-` is activate / deactivate on the target.

Implement parser tests for each block type and each arrow variant, plus aliases (`participant A as Alice`).

- [ ] Test, implement, commit: `git commit -m "sequence: parser (actors/messages/notes/blocks)"`

---

### Task 3: Layout — actor columns + message rows

**Files:** Create `sequence/layout.go`, `sequence/layout_test.go`.

Pseudocode:

```go
func Layout(d *Diagram, opts mermaid.LayoutOptions) *displaylist.DisplayList {
	measure := opts.Measurer.Measure // assume non-nil; mermaid.go fills default
	const (
		actorTop      = 0.0
		actorPadH     = 14.0  // padding inside actor box
		actorPadV     = 8.0
		actorBoxH     = 30.0
		messagePad    = 8.0
		messageTop    = 50.0  // distance from actor box to first message
	)

	// 1. Compute actor column positions.
	var xs []float64
	x := 0.0
	for _, a := range d.Actors {
		w, _ := measure(a.Label, displaylist.RoleActorTitle)
		xs = append(xs, x + w/2 + actorPadH) // x = left edge of box; we want centre
		x += w + actorPadH*2 + opts.SequenceActorSpacing
		if opts.SequenceActorSpacing == 0 {
			x += 30
		}
	}
	totalWidth := x

	// 2. Walk top-level Items emitting rows.
	dl := &displaylist.DisplayList{Width: totalWidth}
	cursorY := messageTop

	// Emit actor headers + lifeline placeholders.
	for i, a := range d.Actors {
		bx := xs[i] - actorBoxH/2
		dl.Items = append(dl.Items, displaylist.Shape{
			Kind: displaylist.ShapeKindRect,
			BBox: displaylist.Rect{X: bx, Y: actorTop, W: actorBoxH * float64(maxLabelLen(a.Label)/3+1), H: actorBoxH},
			Role: displaylist.RoleActorBox,
		})
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: xs[i], Y: actorTop + actorBoxH/2},
			Lines:  []string{a.Label},
			Align:  displaylist.AlignCenter, VAlign: displaylist.VAlignMiddle,
			Role: displaylist.RoleActorTitle,
		})
	}

	// Emit messages, notes, blocks. (Implementation: recursive walk
	// over d.Items; each emits its own primitives + advances cursorY.)
	emitItems(dl, d.Items, xs, &cursorY, /* activations map */ nil)

	// Add closing actor footers (mirror at bottom).
	footerY := cursorY + messagePad
	for i, a := range d.Actors {
		// Lifeline: dashed line from actorTop+box to footerY.
		dl.Items = append(dl.Items, displaylist.Edge{
			Points:    []displaylist.Point{{X: xs[i], Y: actorTop + actorBoxH}, {X: xs[i], Y: footerY}},
			LineStyle: displaylist.LineStyleDashed,
			Role:      displaylist.RoleLifeline,
		})
		// Footer box (mirror of header).
		// ...
		_ = a
	}
	dl.Height = footerY + actorBoxH
	return dl
}
```

Then `emitItems` for each kind:

- **Message:** edge between source/target lifeline at `cursorY`. Label as Text just above. Activation: open a rect on target lifeline at this Y; close on next Deactivate or end.
- **Note:** Cluster + Text. For NoteOver, span from min actor X to max actor X.
- **Block:** Cluster spanning [startY, endY] with label at top-left. Recurse into Children. Alt/par dividers emitted as horizontal Edges crossing the block.

This is the largest single task. Plan to take a full session for it. Break into sub-commits:

- [ ] **3a:** Actor headers + lifelines. Commit `sequence: actor columns + lifelines`.
- [ ] **3b:** Plain messages (no activations, no labels). Commit `sequence: plain messages`.
- [ ] **3c:** Message labels + activations. Commit `sequence: message labels + activations`.
- [ ] **3d:** Notes (left/right/over). Commit `sequence: notes`.
- [ ] **3e:** Blocks (loop/opt/alt/par/critical/break/rect). Commit `sequence: block frames`.

---

### Task 4: Wire sequence into ParseAndLayout

**Files:** Modify `mermaid.go`, `mermaid_test.go`.

```go
case typeSequence:
	ast, err := sequence.Parse(src)
	if err != nil { return nil, err }
	if opts.Measurer == nil { opts.Measurer = opts.measurer() }
	return sequence.Layout(ast, opts), nil
```

Update `mermaid_test.go` to expect a non-nil DisplayList for sequence input.

- [ ] Commit — `git commit -m "mermaid: dispatch sequence"`

---

### Task 5: PDF style — sequence roles

**Files:** Modify `pdf/style.go`.

Extend `DefaultStyle()` to add entries for: `RoleActorBox`, `RoleActorTitle`, `RoleLifeline`, `RoleActivation`, `RoleMessageLabel`, `RoleNoteText`, `RoleSequenceNote`, `RoleLoopBlock`, `RoleAltBlock`, `RoleOptBlock`, `RoleParBlock`, `RoleCriticalBlock`, `RoleBreakBlock`, `RoleRectBlock`.

Default visuals:
- Actor box: solid border, no fill, body text.
- Lifeline: dashed line, muted color.
- Activation: solid border, light grey fill.
- Block frame: solid border, no fill, label bold-italic in upper left.
- Note: light yellow fill, solid border, body text.

- [ ] Commit — `git commit -m "mermaidpdf: sequence-role default styles"`

---

### Task 6: Golden + property tests

**Files:** Create `testdata/sequence/{parse,layout}/*`, `sequence/golden_test.go`, `sequence/property_test.go`.

Fixtures:
- `parse/basic.mmd` — 2 actors, 1 message.
- `parse/all-arrows.mmd` — every arrow variant.
- `parse/blocks.mmd` — nested loop+alt+opt.
- `parse/notes.mmd` — left/right/over notes.
- `layout/*` — same set, golden DisplayList.

Property test: random valid sequence diagrams, no panics, all bboxes inside the diagram bbox, lifeline X coords match actor centres.

- [ ] Commit — `git commit -m "sequence: golden + property tests"`

---

### Task 7: Phase 4 smoke test

```go
func TestPhase4Smoke(t *testing.T) {
	src := `sequenceDiagram
participant A as Alice
participant B as Bob
A->>B: Hi
B-->>A: Hello
loop Every minute
  A->>B: ping
end
`
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil { t.Fatal(err) }
	if dl.Width <= 0 || dl.Height <= 0 {
		t.Fatalf("bbox: %vx%v", dl.Width, dl.Height)
	}
}
```

- [ ] `go test ./...` clean. Commit — `git commit -m "mermaid: phase 4 smoke test"`

---

## Self-Review

| Spec | Phase 4 task |
|---|---|
| participant/actor/aliases | Task 1, 2 |
| Messages (`->>`, `-->>`, `-)`, `--)`) | Task 1, 2 |
| Activations | Task 1, 2, 3c |
| Notes (left/right/over) | Task 1, 2, 3d |
| All block types | Task 1, 2, 3e |
| Hand-rolled layout | Task 3 |
| PDF rendering | Tasks 5 + Phase 2 emitters |
| Tests | Task 6 |

## Open Questions / Risks

- **autonumber**: deferred (defer noted in the spec; mermaigo also defers).
- **Activation nesting**: an actor can have multiple stacked activations. Layout assigns nested rects with increasing X offsets. Verify behaviour with a fixture that has 2-deep nested activations.
