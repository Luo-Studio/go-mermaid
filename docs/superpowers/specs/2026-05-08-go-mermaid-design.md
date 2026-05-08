# go-mermaid: Pure-Go Mermaid Renderer for PDF and Canvas

**Status:** Draft — pending implementation
**Date:** 2026-05-08
**Owner:** orktes / luo-studio
**Module:** `github.com/luo-studio/go-mermaid`

## Goal

A pure-Go library that parses Mermaid diagram source, lays it out, and produces vector output suitable for embedding in fpdf-based PDFs. Built to be the Mermaid counterpart to `github.com/luo-studio/go-tex` — same end-to-end role, dropped into the platform's PDF executor lib (`lib/go/executor/libs/pdf`) the same way LaTeX is dropped in today.

The defining design choices:

- **Style-neutral DisplayList.** Layout output is structural (`Shape`, `Edge`, `Text`, `Cluster`, `Marker`) tagged with semantic roles. Caller supplies colors/fonts/widths per role. No baked-in themes.
- **Hybrid text measurement.** Default measurer uses embedded Inter metrics; caller can override with a `Measurer` interface (e.g., `pdf.GetStringWidth`) for tight fit.
- **Vector-first.** PDF emitter draws via fpdf primitives (`Rect`, `Line`, `Curve`, `Text`) — no SVG rasterization in the PDF path. Selectable, searchable text and crisp output.
- **autog for layout** (`github.com/nulab/autog`) — pure-Go Sugiyama-style hierarchical layout. No graphviz, no wasm, no CGO.
- **Recursive cluster handling.** Subgraph nesting handled by laying out each cluster's interior independently, treating clusters as super-nodes at the parent level, and stitching results.

## Scope

### Diagram Types (v1)

Aiming for **mermaigo parity** plus state diagrams:

| Type | Source keywords | Layout | Notes |
|---|---|---|---|
| Flowchart | `flowchart`, `graph` | autog + cluster recursion | All shapes & edge styles below. Subgraphs supported, nestable. |
| Sequence | `sequenceDiagram` | hand-rolled (column/row) | All block types: loop, alt/else, opt, par/and, critical, break, rect. Notes (left/right/over). Activations. |
| Class | `classDiagram` | autog | All relationship variants (both directions). Cardinality. Annotations. Namespaces. |
| ER | `erDiagram` | autog | All cardinality/line-style combinations. PK/FK/UK markers + comments. |
| State | `stateDiagram`, `stateDiagram-v2` | autog | Pseudostates `[*]` (start/end). Composite states (subgraphs). Transitions with labels. |

### Flowchart Feature Coverage

**Directions:** `TD`, `TB`, `BT`, `LR`, `RL`.

**Node shapes:** rect `[]`, rounded `()`, stadium `([])`, diamond `{}`, circle `(())`, double-circle `(((...)))`, subroutine `[[]]`, cylinder `[()]`, hexagon `{{}}`, asymmetric `>...]`, trapezoid `[/...\]`, trapezoid-alt `[\.../]`, parallelogram, parallelogram-alt, state-start/end pseudostates `[*]`.

**Edges:** `-->` `---` `-.->` `-.-` `==>` `===` plus bidirectional variants `<-->` `<-.->` `<==>`. Labels via `-- text -->` and `|text|`.

**Other:** subgraph blocks (with title, nestable), `classDef` + `class A foo`, `:::className` shorthand, `style` directives, `%% comments`, `;` statement separators.

### Sequence Feature Coverage

`participant`/`actor`/aliases. Messages: `->>` (sync), `-->>` (return), `-)` (open), `--)` (dashed open). Activation bars (`activate`/`deactivate`, `+`/`-` shorthand). Notes: `Note left of`, `Note right of`, `Note over`. Blocks: `loop`, `alt`/`else`, `opt`, `par`/`and`, `critical`, `break`, `rect`.

### Class Feature Coverage

Class definitions with attributes & methods (visibility `+`/`-`/`#`/`~`). All relationship variants both directions: `<|--`, `<|..`, `*--`, `--*`, `o--`, `--o`, `-->`, `..>`, `..|>`, `--`. Annotations `<<interface>>`. Cardinality strings `"1" --> "*"`. `namespace` blocks.

### ER Feature Coverage

Entity blocks with attribute lists (type, name, optional PK/FK/UK marker, optional comment string). Cardinality: `||`, `|o`, `o|`, `}|`, `|{`, `o{`, `{o`. Line style: `--` (identifying) and `..` (non-identifying). Relationship labels.

### Out of Scope (v1)

Gantt, pie, mindmap, journey, gitGraph, timeline, quadrantChart, requirementDiagram, C4, sankey, xychart, block diagrams. Themes/CSS variables. Animation. mermaid-js parser quirks beyond mermaigo's coverage. Math-mode in labels (defer; could be added later by integrating go-tex).

## Architecture

### Package Layout

```
go-mermaid/
├── mermaid.go                 # public top-level API: ParseAndLayout, LayoutOptions
├── flowchart/                 # parser + ast + layout (autog + cluster recursion)
├── sequence/                  # parser + ast + layout (hand-rolled)
├── class/                     # parser + ast + layout (autog)
├── er/                        # parser + ast + layout (autog)
├── state/                     # parser + ast + layout (autog)
├── displaylist/               # cross-type Shape/Edge/Text/Cluster/Marker + Role/Kind enums
├── autog/                     # adapter over nulab/autog with cluster recursion
├── fonts/                     # embedded Inter Regular + Bold + Italic TTFs + loader
├── fontmetrics/               # default Measurer using embedded Inter metrics
├── canvasr/                   # DisplayList → tdewolff/canvas (PNG/SVG/PDF)
├── pdf/                       # DisplayList → fpdf (DrawInto) + DrawMermaid one-call
├── cmd/
│   ├── parse/                 # stdin .mmd → JSON AST (debugging)
│   └── render/                # stdin .mmd → PDF/PNG (CLI)
└── testdata/
    ├── flowchart/, sequence/, class/, er/, state/    # .mmd inputs + golden DisplayList JSON
    └── corpus/                                        # canonical examples
```

### Why diagram-type-as-package, not stage-as-package

Mermaid's 5 diagram types share almost no parser or layout code — flowchart syntax has nothing in common with sequence, class with ER. What they share is small and well-bounded: the `displaylist` output format, the `autog` adapter (used by 4 of 5 types), `fonts`/`fontmetrics`, and the emitters.

Each diagram type's parser+ast+layout are tightly coupled internally and loosely coupled across types. Iterating on the sequence parser only touches `sequence/`; adding a new diagram type is one new package.

The trade-off: cross-cutting changes (e.g., adding a new node shape kind) touch multiple packages. Acceptable — that happens rarely and is the natural place for it to happen.

### Pipeline

```
Mermaid source
     │  detect diagram type (first non-comment line)
     ▼
{flowchart, sequence, class, er, state}.Parse
     │
     ▼
typed AST (per package)
     │  {flowchart, sequence, ...}.Layout (with autog adapter or hand-rolled)
     │  + opts.Measurer for text width/height
     ▼
displaylist.DisplayList
     │
     │       ├── pdf.DrawInto      → into an fpdf canvas
     │       │
     │       └── canvasr.RenderPNG → bytes via tdewolff/canvas
```

### DisplayList Primitive Set

```go
package displaylist

type DisplayList struct {
    Width, Height float64
    Items         []Item
}

type Item interface{ isItem() }

// Shape — node-like geometry. Common kinds rendered natively by emitters;
// custom kinds carry an explicit polygon path.
type Shape struct {
    Kind ShapeKind   // rect, round, stadium, diamond, circle, doubleCircle,
                     // ellipse, hexagon, cylinder, custom
    BBox Rect        // x, y, w, h
    Path []Point     // populated iff Kind == ShapeKindCustom
    Role Role        // semantic tag for caller styling
}

// Edge — polyline with arrow markers at endpoints.
type Edge struct {
    Points     []Point   // waypoints incl. endpoints
    LineStyle  LineStyle // solid, dashed, thick
    ArrowStart MarkerKind // none, arrow, arrowOpen, diamondFilled, diamondOpen,
                          // triangleOpen, cross, circleOpen
    ArrowEnd   MarkerKind
    Role       Role
}

// Text — rendered string (possibly multi-line).
type Text struct {
    Pos    Point
    Lines  []string
    Align  Align        // left, center, right
    VAlign VAlign       // top, middle, baseline, bottom
    Role   Role         // node, edgeLabel, clusterTitle, classMember,
                        // attribute, actorTitle, noteText, ...
}

// Cluster — backdrop rectangle for subgraphs / sequence-blocks / state-composite.
type Cluster struct {
    BBox  Rect
    Title string
    Role  Role           // subgraph, altBlock, loopBlock, optBlock,
                         // parBlock, criticalBlock, breakBlock, rectBlock,
                         // sequenceNote, stateComposite, ...
}

// Marker — standalone marker (rare; usually inline on Edge).
type Marker struct {
    Pos   Point
    Angle float64       // radians, rotation
    Kind  MarkerKind
    Role  Role
}

type Role string         // open enum; emitter style maps key off these
type ShapeKind string
type LineStyle string
type MarkerKind string
type Align string
type VAlign string

type Rect struct{ X, Y, W, H float64 }
type Point struct{ X, Y float64 }
```

`Role` is intentionally an open string enum. Standard roles are predefined as constants (`displaylist.RoleNode`, `RoleEdge`, `RoleEdgeLabel`, `RoleClusterTitle`, `RoleSubgraph`, `RoleActorBox`, `RoleActorTitle`, `RoleLifeline`, `RoleActivation`, `RoleMessageLabel`, `RoleNoteText`, `RoleSequenceNote`, `RoleLoopBlock`, `RoleAltBlock`, `RoleOptBlock`, `RoleParBlock`, `RoleCriticalBlock`, `RoleBreakBlock`, `RoleRectBlock`, `RoleClassBox`, `RoleClassMember`, `RoleClassAnnotation`, `RoleEntityBox`, `RoleEntityAttribute`, `RoleStateBox`, `RoleStateComposite`, `RolePseudostateStart`, `RolePseudostateEnd`). Emitters look up styling via `Style.Roles[role]` with `Style.Default` as fallback, so unknown/custom roles still render — they just inherit defaults. The set of standard roles is a closed contract between layout and emitters; new roles get added simultaneously to layout (which produces them) and the default style (which provides reasonable visuals).

### Public API Surface

#### Top-level entry point (`mermaid.go`)

```go
package mermaid

// ParseAndLayout detects the diagram type from src, runs the parser and
// layout for that type, and returns the resulting DisplayList.
//
// Returns ErrUnknownDiagram if src does not start with a recognized
// diagram-type keyword.
func ParseAndLayout(src string, opts LayoutOptions) (*displaylist.DisplayList, error)

// LayoutOptions are common knobs shared across diagram types.
type LayoutOptions struct {
    // Measurer measures rendered text width and ascent/descent height.
    // If nil, an embedded-Inter Measurer is used.
    Measurer Measurer

    // FontSize used by the default Measurer in points. Default: 14.
    // Measurer implementations are free to ignore this.
    FontSize float64

    // Padding, NodeSpacing, LayerSpacing — autog tuning. Defaults sized
    // for legibility at typical PDF fonts.
    Padding      float64
    NodeSpacing  float64
    LayerSpacing float64

    // Sequence-specific spacing knobs. Ignored for non-sequence diagrams.
    SequenceActorSpacing   float64
    SequenceMessageSpacing float64
}

type Measurer interface {
    Measure(text string, role displaylist.Role) (w, h float64)
}

var ErrUnknownDiagram = errors.New("mermaid: unrecognized diagram type")
```

#### Per-diagram-type entry points

Each of `flowchart/`, `sequence/`, `class/`, `er/`, `state/` exposes:

```go
func Parse(src string) (*AST, error)
func Layout(ast *AST, opts mermaid.LayoutOptions) *displaylist.DisplayList
```

These are escape hatches for callers who want to inspect or modify the AST before layout, or for parser-only tests.

#### PDF emitter (`pdf/`)

```go
package mermaidpdf

// DrawMermaid is the one-call helper: parse → layout → draw. Good default.
func DrawMermaid(pdf *fpdf.Fpdf, src string, x, y float64, opts EmbedOptions) error

// DrawInto draws an already-laid-out DisplayList. Use this when you want
// to inspect the DisplayList (e.g., check width, decide on scaling)
// before drawing.
func DrawInto(pdf *fpdf.Fpdf, dl *displaylist.DisplayList, x, y float64, opts EmbedOptions) error

type EmbedOptions struct {
    // Style maps DisplayList roles to colors/fonts/line widths.
    // EmbedDefaults() returns a sensible black-on-white style.
    Style Style

    // Layout options forwarded to ParseAndLayout. Layout.Measurer should
    // typically be a wrapper around pdf.GetStringWidth for tight fit.
    Layout mermaid.LayoutOptions

    // MaxWidth caps the rendered width. If the laid-out DisplayList is
    // wider, it is uniformly scaled down. 0 = no cap.
    MaxWidth float64
}

type Style struct {
    // Per-role visual style. Keys are displaylist.Role values.
    Roles map[displaylist.Role]RoleStyle
    // Fallback used when a role has no explicit entry.
    Default RoleStyle
}

type RoleStyle struct {
    // Stroke color (RGB 0-255). Empty = no stroke.
    StrokeR, StrokeG, StrokeB float64
    StrokeWidth                float64
    DashPattern                []float64

    // Fill color. Empty = no fill (transparent).
    FillR, FillG, FillB float64

    // Text color and font (only relevant to Text items).
    TextR, TextG, TextB float64
    Font                string  // fpdf font family
    FontStyle           string  // "", "B", "I", "BI"
    FontSize            float64
}

func EmbedDefaults() EmbedOptions
```

#### Canvas rasterizer (`canvasr/`)

```go
package mermaidcanvasr

func RenderPNG(src string, opts RenderOptions) ([]byte, error)
func RenderInto(c *canvas.Canvas, dl *displaylist.DisplayList, opts RenderOptions) error

type RenderOptions struct {
    Style    Style
    Layout   mermaid.LayoutOptions
    DPI      float64        // for PNG, default 192
    MaxWidth float64        // pixels
}
```

### Cluster (Subgraph) Layout

Recursive per-cluster layout for flowchart and state diagrams:

1. **Build cluster tree** during parse — each subgraph block becomes a `Cluster` node containing child Nodes and child Clusters.
2. **Recursive layout**:
   - For each leaf cluster (no nested children): build an autog edge list from its member nodes, run `autog.Layout` with those nodes' measured sizes, capture (w, h, node positions, edge waypoints) into a cached "cluster size + interior layout".
   - For each non-leaf cluster: do the same, but use cached child-cluster sizes as if they were nodes; their interiors get translated to absolute positions in a final pass.
3. **Cross-cluster edges** — edges where source and target are in different clusters — are laid out at the lowest common ancestor level. autog's edge routing inside that level determines the bend points; we splice the path through cluster boundaries (entering at the cluster's bbox edge nearest the segment).
4. **Emit** — once all positions are absolute, emit a `Cluster` item per subgraph (backdrop + title), `Shape` items for nodes, `Edge` items for edges. Cluster `Role` defaults to `RoleSubgraph`; nested clusters get the same role unless the parser tagged them differently.

This logic lives in `autog/cluster.go` (the bookkeeping is autog-agnostic; only the inner layout calls touch autog). `flowchart/` and `state/` delegate to it.

### Sequence Diagram Layout (no autog)

Two-pass column/row layout:

1. **Columns:** assign each actor a column index in declaration order. Compute column X positions: `X[i] = X[i-1] + maxLabelWidth(actor[i-1], actor[i]) + sequenceMessageSpacing`.
2. **Rows:** walk the message timeline. Each message, note, or block-divider gets a row Y. Block frames span from their start row to end row. Activations are rectangles overlaid on lifelines spanning their open-message range.
3. **Emit:**
   - Actor headers: `Shape{Kind: rect, Role: RoleActorBox}` + `Text{Role: RoleActorTitle}`.
   - Lifelines: `Edge{Points: [(X, headerBottom), (X, footerTop)], LineStyle: dashed, Role: RoleLifeline}`.
   - Activations: `Shape{Kind: rect, Role: RoleActivation}`.
   - Messages: `Edge` between actor lifelines + `Text{Role: RoleMessageLabel}` above the line.
   - Notes: `Cluster{Role: RoleSequenceNote, Title: ""}` + `Text{Role: RoleNoteText}`.
   - Blocks: `Cluster{Role: RoleLoopBlock|RoleAltBlock|...}` enclosing their span, with divider Edges for `else`/`and`.

### Class Diagram Layout (autog-based)

1. Each class becomes one autog node sized to fit its name + member list. Member list height is computed by the Measurer.
2. Relationships become autog edges; arrow-marker kind is recorded on the Edge for the emitter.
3. Cardinality strings (`"1" --> "*"`) become small `Text` items pinned near each endpoint.
4. Annotations `<<interface>>` are rendered as a small italic line above the class name.
5. Namespaces are clusters (same machinery as flowchart subgraphs).

### ER Diagram Layout (autog-based)

1. Each entity becomes one autog node; size = max(entity name width, attribute list width) × (1 + attribute count).
2. Relationships become autog edges. Cardinality glyphs at each endpoint are rendered as `Marker` items based on the parsed cardinality (`||`, `|o`, `}|`, etc.).
3. Identifying vs non-identifying styling drives `Edge.LineStyle` (solid vs dashed).

### State Diagram Layout (autog-based, with composite states)

Treated as a flowchart variant: states become nodes, transitions become edges, composite states become clusters (recursive layout). Pseudostates (`[*]`) get special `ShapeKind` (filled bullet for start, bullseye for end).

## Error Handling

- **Parser errors** are real errors: malformed Mermaid → `mermaid.ErrParse` wrapping a positional message (line, column, fragment). Caller decides whether to surface to user or fall back.
- **Layout errors** are rare but possible (e.g., cyclic edge in a tree-only structure, autog internal failures). Wrapped in `mermaid.ErrLayout` with the offending diagram-type name. Layout/parser code wraps panics from third-party libraries (autog) into errors so a malformed input doesn't crash the caller. Programmer-error panics (`runtime.Error`: nil deref, index OOB, …) are deliberately re-raised so real bugs surface — same pattern as `lib/go/executor/libs/pdf/latex.go:layoutLatex`.
- **Emitter errors** propagate fpdf/canvas errors verbatim wrapped with context.
- **`ErrUnknownDiagram`** when source's first non-comment line doesn't match a known type. Caller can match-and-fallback (e.g., render a `[Mermaid: <source>]` placeholder).

## Testing Strategy

### Unit + golden snapshots (Approach A)

- **Parser tests** per diagram type: `parser_test.go` with table-driven inputs and golden JSON ASTs in `testdata/<type>/parse/`. Update with a `-update` flag.
- **Layout tests** per diagram type: take a parsed AST, run layout with a deterministic Measurer (fixed-width 7px-per-char), assert golden DisplayList JSON in `testdata/<type>/layout/`.
- **Emitter integration tests**: render each canonical input through `pdf.DrawMermaid` and `canvasr.RenderPNG`. Assert: PDF parses, has expected object count; PNG decodes, has expected dimensions.

### Property tests (Approach C)

- **No-panic property:** for a generator producing random valid Mermaid (pulled from a fuzz corpus seeded with mermaid-js examples), `ParseAndLayout` must not panic.
- **Bbox invariants:** for any generated DisplayList, every `Shape.BBox` lies within `[0, Width] × [0, Height]`.
- **Cluster containment:** every node assigned to a cluster has its bbox fully inside the cluster's bbox.
- **Edge endpoint correctness:** every `Edge.Points[0]` and `Edge.Points[-1]` lies on or within the bbox of its source/target node.

### Visual review (manual)

A `cmd/render` binary lets contributors visually inspect output against `testdata/corpus/` examples during development. No automated pixel-diff in CI.

## Implementation Order

The work decomposes into five phases. Each is a roughly self-contained PR:

1. **Foundation** — module init, `displaylist/`, `fontmetrics/` (default measurer + embedded Inter metrics), `fonts/` (embedded TTFs), `autog/` adapter without clusters, top-level `mermaid.go` skeleton with diagram-type detection. Test harness scaffolding.
2. **Flowchart, no clusters** — `flowchart/parser.go`, `flowchart/ast.go`, `flowchart/layout.go` using the autog adapter. All shapes, all edge styles. Golden tests. Subgraphs parsed but flattened in layout (placeholder).
3. **Cluster recursion** — `autog/cluster.go` recursive layout, retrofitted into `flowchart`. Re-record golden tests where they change.
4. **Sequence, class, ER, state** — each their own package, each with parser + layout + golden tests. Class and ER use the autog adapter (no clusters needed for ER; class uses clusters for namespaces). State uses the cluster-aware autog adapter.
5. **Emitters** — `pdf/` (DrawInto + DrawMermaid + EmbedDefaults). `canvasr/` (RenderPNG + RenderInto). `cmd/parse`, `cmd/render`. Integration tests.

The platform integration (`lib/go/executor/libs/pdf/mermaid.go` + `parse_markdown.go` ` ```mermaid ` block parsing) is a follow-up step in the platform repo, not part of go-mermaid.

## Open Questions

None blocking — see Implementation Order above.

## References

- mermaigo: <https://github.com/a-kaibu/mermaigo> — the reference Pure-Go Mermaid implementation we're matching for parity (does not use autog; uses goccy/go-graphviz).
- nulab/autog: <https://github.com/nulab/autog> — the pure-Go layout engine we depend on.
- go-tex: `/Users/jaakkolukkari/Development/luo/go-tex` — sibling library; same role for LaTeX. Don't blindly mirror its structure (it's a port; it has port-specific shapes); do mirror its emitter integration pattern (`DrawInto` + role-driven styling).
- mermaid.js: <https://mermaid.js.org> — syntax reference. We do *not* aim for byte parity with its output.
- nulab/autog functional options: <https://pkg.go.dev/github.com/nulab/autog>.
