# go-mermaid Phase 1 (Foundation) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the go-mermaid module with all foundational packages — `displaylist`, `fontmetrics`, `fonts`, `autog` adapter, top-level `mermaid` package — and scaffold the test harness. After this phase the module compiles, all unit tests pass, and `mermaid.ParseAndLayout` recognises diagram-type keywords but returns `ErrUnknownDiagram` for every type. Real parsers/layouts land in Phase 2+.

**Architecture:** Per-diagram-type packages will sit alongside the foundation packages introduced here. Phase 1 produces only the cross-cutting infrastructure: a style-neutral `DisplayList` IR, a default `Measurer` backed by embedded Inter font metrics, embedded Inter TTFs for the canvas rasterizer, and a thin adapter over `github.com/nulab/autog` that the future layout packages will call.

**Tech Stack:** Go 1.25, `github.com/nulab/autog` (pure-Go Sugiyama layout), `golang.org/x/text` for Unicode width approximation in the default measurer.

---

## Spec Reference

- Spec: `docs/superpowers/specs/2026-05-08-go-mermaid-design.md`
- Phase 1 corresponds to the "Foundation" entry in *Implementation Order*.

## File Structure (Phase 1)

```
go-mermaid/
├── go.mod                           # module github.com/luo-studio/go-mermaid
├── go.sum
├── .gitignore                       # exists
├── LICENSE                          # add (MIT)
├── README.md                        # extend
├── mermaid.go                       # Measurer, LayoutOptions, ParseAndLayout, ErrUnknownDiagram
├── mermaid_test.go                  # diagram-type detection tests
├── displaylist/
│   ├── displaylist.go               # DisplayList, Item, Shape, Edge, Text, Cluster, Marker, Rect, Point
│   ├── role.go                      # Role + standard role constants
│   ├── kind.go                      # ShapeKind, LineStyle, MarkerKind, Align, VAlign enums
│   ├── json.go                      # MarshalJSON/UnmarshalJSON for Item interface
│   └── displaylist_test.go
├── fontmetrics/
│   ├── inter.go                     # embedded Inter metrics tables (//go:embed inter_*.csv)
│   ├── inter_regular.csv            # generated, advance widths per codepoint
│   ├── inter_bold.csv               # generated
│   ├── inter_italic.csv             # generated
│   ├── default.go                   # DefaultMeasurer implementation
│   └── fontmetrics_test.go
├── fonts/
│   ├── inter.go                     # //go:embed Inter-Regular.ttf, Inter-Bold.ttf, Inter-Italic.ttf
│   ├── Inter-Regular.ttf            # ~300 KB
│   ├── Inter-Bold.ttf
│   ├── Inter-Italic.ttf
│   ├── loader.go                    # public API: Bytes(style) → []byte; Face(style, size) → font.Face
│   └── fonts_test.go
├── autog/
│   ├── adapter.go                   # public Layout func; Node, Edge, Result types
│   ├── source.go                    # graph.Source implementation (private edgeSource)
│   └── adapter_test.go
└── docs/superpowers/                # already exists
```

## Tasks

### Task 1: Initialise the Go module

**Files:**
- Create: `go.mod`
- Create: `LICENSE`
- Modify: `README.md` (replace stub)

- [ ] **Step 1.1: Create `go.mod`**

```bash
cd /Users/jaakkolukkari/Development/luo/go-mermaid
go mod init github.com/luo-studio/go-mermaid
```

Verify the file looks like:
```
module github.com/luo-studio/go-mermaid

go 1.25
```

- [ ] **Step 1.2: Add Inter TTFs**

Inter is OFL-1.1. Download the three files we need and place them in `fonts/`:
```bash
mkdir -p fonts
curl -L -o fonts/Inter-Regular.ttf https://github.com/rsms/inter/raw/v4.0/docs/font-files/Inter-Regular.ttf
curl -L -o fonts/Inter-Bold.ttf    https://github.com/rsms/inter/raw/v4.0/docs/font-files/Inter-Bold.ttf
curl -L -o fonts/Inter-Italic.ttf  https://github.com/rsms/inter/raw/v4.0/docs/font-files/Inter-Italic.ttf
ls -lh fonts/*.ttf
```
Expected: three files, each ~300 KB. (If the URL changes upstream, fetch from a current Inter release tag and update this command.)

- [ ] **Step 1.3: Add MIT license + Inter OFL note**

Create `LICENSE` with the standard MIT license text (copyright "2026 Luo Studio Oy"). Inter is included under OFL-1.1; record this in `fonts/LICENSE-Inter.txt` by downloading <https://github.com/rsms/inter/raw/v4.0/LICENSE.txt>.

- [ ] **Step 1.4: Replace README.md stub**

Replace the contents of `README.md` with:
```markdown
# go-mermaid

Pure-Go [Mermaid](https://mermaid.js.org) diagram parser, layout engine,
and renderer. Produces a style-neutral `DisplayList` IR that downstream
emitters draw into PDFs (via [fpdf](https://codeberg.org/go-pdf/fpdf))
or rasterize via [tdewolff/canvas](https://github.com/tdewolff/canvas).

Built as the Mermaid counterpart to
[`go-tex`](https://github.com/luo-studio/go-tex). Layout uses
[nulab/autog](https://github.com/nulab/autog) — no Graphviz, no wasm,
no CGO.

## Status

Phase 1 (foundation) — module compiles, infrastructure packages in
place. No diagram parsing yet. See
`docs/superpowers/specs/2026-05-08-go-mermaid-design.md` for the full
design.
```

- [ ] **Step 1.5: Verify the module compiles**

```bash
go build ./...
```
Expected: succeeds, no output.

- [ ] **Step 1.6: Commit**

```bash
git add go.mod LICENSE README.md fonts/Inter-Regular.ttf fonts/Inter-Bold.ttf fonts/Inter-Italic.ttf fonts/LICENSE-Inter.txt
git commit -m "Init module + bundled Inter TTFs"
```

---

### Task 2: `displaylist` types and enums

**Files:**
- Create: `displaylist/displaylist.go`
- Create: `displaylist/kind.go`
- Create: `displaylist/role.go`
- Create: `displaylist/displaylist_test.go`

- [ ] **Step 2.1: Write the failing test**

Create `displaylist/displaylist_test.go`:
```go
package displaylist

import "testing"

func TestShapeImplementsItem(t *testing.T) {
	var _ Item = Shape{}
	var _ Item = Edge{}
	var _ Item = Text{}
	var _ Item = Cluster{}
	var _ Item = Marker{}
}

func TestRectContains(t *testing.T) {
	r := Rect{X: 10, Y: 10, W: 20, H: 20}
	if !r.Contains(Point{X: 15, Y: 15}) {
		t.Fatal("Rect should contain its interior point")
	}
	if r.Contains(Point{X: 5, Y: 15}) {
		t.Fatal("Rect should not contain a point left of it")
	}
	if r.Contains(Point{X: 30, Y: 15}) {
		t.Fatal("Rect should not contain a point at its right edge (half-open)")
	}
}

func TestDisplayListZeroValue(t *testing.T) {
	var dl DisplayList
	if len(dl.Items) != 0 {
		t.Fatal("zero-value DisplayList should have no items")
	}
	if dl.Width != 0 || dl.Height != 0 {
		t.Fatal("zero-value DisplayList should have zero dimensions")
	}
}
```

- [ ] **Step 2.2: Run the test to verify it fails**

```bash
go test ./displaylist/...
```
Expected: FAIL — `displaylist` package does not exist yet.

- [ ] **Step 2.3: Create `displaylist/kind.go`**

```go
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
	MarkerNone           MarkerKind = ""
	MarkerArrow          MarkerKind = "arrow"          // filled triangle
	MarkerArrowOpen      MarkerKind = "arrowOpen"      // hollow triangle
	MarkerDiamondFilled  MarkerKind = "diamondFilled"  // composition
	MarkerDiamondOpen    MarkerKind = "diamondOpen"    // aggregation
	MarkerTriangleOpen   MarkerKind = "triangleOpen"   // inheritance
	MarkerCross          MarkerKind = "cross"          // sequence "lost"
	MarkerCircleOpen     MarkerKind = "circleOpen"     // sequence "open"
	MarkerCardinalityOne MarkerKind = "cardOne"        // ER ||
	MarkerCardinalityZeroOrOne MarkerKind = "cardZeroOrOne" // ER |o or o|
	MarkerCardinalityOneOrMore MarkerKind = "cardOneOrMore" // ER }| or |{
	MarkerCardinalityZeroOrMore MarkerKind = "cardZeroOrMore" // ER }o or o{
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
```

- [ ] **Step 2.4: Create `displaylist/role.go`**

```go
package displaylist

// Role is a semantic tag attached to every Item. Emitters look up
// colors, fonts, and line widths in a caller-supplied per-Role style
// map. Standard roles are defined as constants below; callers may use
// custom Role values, in which case emitters fall back to the default
// style.
type Role string

const (
	RoleUnknown Role = ""

	// Flowchart / state
	RoleNode             Role = "node"
	RoleEdge             Role = "edge"
	RoleEdgeLabel        Role = "edgeLabel"
	RoleSubgraph         Role = "subgraph"
	RoleClusterTitle     Role = "clusterTitle"
	RolePseudostateStart Role = "pseudostateStart"
	RolePseudostateEnd   Role = "pseudostateEnd"
	RoleStateBox         Role = "stateBox"
	RoleStateComposite   Role = "stateComposite"

	// Sequence
	RoleActorBox       Role = "actorBox"
	RoleActorTitle     Role = "actorTitle"
	RoleLifeline       Role = "lifeline"
	RoleActivation     Role = "activation"
	RoleMessageLabel   Role = "messageLabel"
	RoleNoteText       Role = "noteText"
	RoleSequenceNote   Role = "sequenceNote"
	RoleLoopBlock      Role = "loopBlock"
	RoleAltBlock       Role = "altBlock"
	RoleOptBlock       Role = "optBlock"
	RoleParBlock       Role = "parBlock"
	RoleCriticalBlock  Role = "criticalBlock"
	RoleBreakBlock     Role = "breakBlock"
	RoleRectBlock      Role = "rectBlock"

	// Class
	RoleClassBox        Role = "classBox"
	RoleClassMember     Role = "classMember"
	RoleClassAnnotation Role = "classAnnotation"

	// ER
	RoleEntityBox       Role = "entityBox"
	RoleEntityAttribute Role = "entityAttribute"
)
```

- [ ] **Step 2.5: Create `displaylist/displaylist.go`**

```go
package displaylist

// Point is a 2D coordinate in DisplayList units (consistent with
// LayoutOptions.Padding etc — typically points or pixels depending on
// caller).
type Point struct {
	X, Y float64
}

// Rect is an axis-aligned rectangle. (X, Y) is the top-left corner;
// W and H are width and height. Coordinates grow right and down.
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

// Item is the closed sum type of DisplayList items. Implementations:
// Shape, Edge, Text, Cluster, Marker.
type Item interface {
	itemKind() string
}

// Shape is a node-like geometry. Common kinds are drawn natively by
// emitters; ShapeKindCustom carries an explicit polygon Path.
type Shape struct {
	Kind ShapeKind
	BBox Rect
	Path []Point // populated iff Kind == ShapeKindCustom
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

// Text is a rendered string. Lines is the wrapped multi-line content;
// emitters concatenate or stack as the role implies.
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
// In most cases markers ride on an Edge instead.
type Marker struct {
	Pos   Point
	Angle float64
	Kind  MarkerKind
	Role  Role
}

func (Marker) itemKind() string { return "marker" }
```

- [ ] **Step 2.6: Run the test to verify it passes**

```bash
go test ./displaylist/...
```
Expected: PASS.

- [ ] **Step 2.7: Commit**

```bash
git add displaylist/
git commit -m "displaylist: add DisplayList IR types and standard roles"
```

---

### Task 3: `displaylist` JSON serialisation

Used for golden-snapshot tests. The Item interface needs a discriminator on the wire.

**Files:**
- Create: `displaylist/json.go`
- Modify: `displaylist/displaylist_test.go` (add tests)

- [ ] **Step 3.1: Write the failing test**

Append to `displaylist/displaylist_test.go`:
```go
import (
	"encoding/json"
)

func TestDisplayListJSONRoundTrip(t *testing.T) {
	in := DisplayList{
		Width:  100,
		Height: 50,
		Items: []Item{
			Shape{Kind: ShapeKindRect, BBox: Rect{X: 0, Y: 0, W: 30, H: 20}, Role: RoleNode},
			Edge{
				Points:     []Point{{X: 30, Y: 10}, {X: 60, Y: 10}},
				LineStyle:  LineStyleSolid,
				ArrowEnd:   MarkerArrow,
				Role:       RoleEdge,
			},
			Text{Pos: Point{X: 15, Y: 10}, Lines: []string{"A"}, Align: AlignCenter, VAlign: VAlignMiddle, Role: RoleNode},
			Cluster{BBox: Rect{X: 0, Y: 0, W: 100, H: 50}, Title: "outer", Role: RoleSubgraph},
			Marker{Pos: Point{X: 60, Y: 10}, Angle: 0, Kind: MarkerArrow, Role: RoleEdge},
		},
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out DisplayList
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Width != in.Width || out.Height != in.Height {
		t.Fatalf("dims: got %v×%v want %v×%v", out.Width, out.Height, in.Width, in.Height)
	}
	if len(out.Items) != len(in.Items) {
		t.Fatalf("item count: got %d want %d", len(out.Items), len(in.Items))
	}
	for i := range in.Items {
		if in.Items[i].itemKind() != out.Items[i].itemKind() {
			t.Fatalf("item %d kind: got %s want %s", i, out.Items[i].itemKind(), in.Items[i].itemKind())
		}
	}
}
```

- [ ] **Step 3.2: Run the test to verify it fails**

```bash
go test ./displaylist/ -run TestDisplayListJSONRoundTrip
```
Expected: FAIL — Item is an interface; default unmarshal can't pick the concrete type.

- [ ] **Step 3.3: Implement `displaylist/json.go`**

```go
package displaylist

import (
	"encoding/json"
	"fmt"
)

// DisplayList serialises Items with a "kind" discriminator so the
// closed sum type round-trips through JSON. Kept distinct from the
// inline Item structs so callers don't see the wire shape.

type wireItem struct {
	Kind string          `json:"kind"`
	Body json.RawMessage `json:"body"`
}

type wireDisplayList struct {
	Width  float64    `json:"width"`
	Height float64    `json:"height"`
	Items  []wireItem `json:"items"`
}

// MarshalJSON encodes a DisplayList as {width, height, items: [{kind,
// body}]}. Each item's body is the standard struct encoding of its
// concrete type.
func (dl DisplayList) MarshalJSON() ([]byte, error) {
	w := wireDisplayList{Width: dl.Width, Height: dl.Height}
	for _, it := range dl.Items {
		body, err := json.Marshal(it)
		if err != nil {
			return nil, fmt.Errorf("displaylist: marshal %s item: %w", it.itemKind(), err)
		}
		w.Items = append(w.Items, wireItem{Kind: it.itemKind(), Body: body})
	}
	return json.Marshal(w)
}

// UnmarshalJSON decodes the discriminated form back into typed Items.
func (dl *DisplayList) UnmarshalJSON(data []byte) error {
	var w wireDisplayList
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	dl.Width = w.Width
	dl.Height = w.Height
	dl.Items = nil
	for i, wi := range w.Items {
		var it Item
		switch wi.Kind {
		case "shape":
			var s Shape
			if err := json.Unmarshal(wi.Body, &s); err != nil {
				return fmt.Errorf("displaylist: item %d shape body: %w", i, err)
			}
			it = s
		case "edge":
			var e Edge
			if err := json.Unmarshal(wi.Body, &e); err != nil {
				return fmt.Errorf("displaylist: item %d edge body: %w", i, err)
			}
			it = e
		case "text":
			var x Text
			if err := json.Unmarshal(wi.Body, &x); err != nil {
				return fmt.Errorf("displaylist: item %d text body: %w", i, err)
			}
			it = x
		case "cluster":
			var c Cluster
			if err := json.Unmarshal(wi.Body, &c); err != nil {
				return fmt.Errorf("displaylist: item %d cluster body: %w", i, err)
			}
			it = c
		case "marker":
			var m Marker
			if err := json.Unmarshal(wi.Body, &m); err != nil {
				return fmt.Errorf("displaylist: item %d marker body: %w", i, err)
			}
			it = m
		default:
			return fmt.Errorf("displaylist: unknown item kind %q at index %d", wi.Kind, i)
		}
		dl.Items = append(dl.Items, it)
	}
	return nil
}
```

- [ ] **Step 3.4: Run the test to verify it passes**

```bash
go test ./displaylist/...
```
Expected: PASS.

- [ ] **Step 3.5: Commit**

```bash
git add displaylist/json.go displaylist/displaylist_test.go
git commit -m "displaylist: JSON marshal/unmarshal with kind discriminator"
```

---

### Task 4: `fontmetrics` — embedded Inter metrics tables

The default Measurer needs glyph advance widths. We extract advance widths from each Inter TTF once at package init using `golang.org/x/image/font/sfnt`, then cache them.

**Files:**
- Create: `fontmetrics/inter.go`
- Create: `fontmetrics/default.go`
- Create: `fontmetrics/fontmetrics_test.go`

- [ ] **Step 4.1: Add the sfnt dep**

```bash
go get golang.org/x/image@latest
```
Expected: `go.sum` updated.

- [ ] **Step 4.2: Write the failing test**

Create `fontmetrics/fontmetrics_test.go`:
```go
package fontmetrics

import (
	"math"
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func TestDefaultMeasurerNonZero(t *testing.T) {
	m := NewDefault(14)
	w, h := m.Measure("Hello", displaylist.RoleNode)
	if w <= 0 {
		t.Fatalf("width should be positive, got %v", w)
	}
	if h <= 0 {
		t.Fatalf("height should be positive, got %v", h)
	}
}

func TestDefaultMeasurerScalesWithLength(t *testing.T) {
	m := NewDefault(14)
	short, _ := m.Measure("A", displaylist.RoleNode)
	longText, _ := m.Measure("AAAAAAAAAA", displaylist.RoleNode)
	// 10 As should be approximately 10× wider than 1 A within a wide tolerance
	if longText < short*8 || longText > short*12 {
		t.Fatalf("scaling: got short=%v long=%v ratio=%v want 8..12", short, longText, longText/short)
	}
}

func TestDefaultMeasurerLineHeightStable(t *testing.T) {
	m := NewDefault(14)
	_, h1 := m.Measure("A", displaylist.RoleNode)
	_, h2 := m.Measure("Beethoven", displaylist.RoleNode)
	if math.Abs(h1-h2) > 0.001 {
		t.Fatalf("line height should not depend on text content: %v vs %v", h1, h2)
	}
}

func TestDefaultMeasurerBoldRole(t *testing.T) {
	m := NewDefault(14)
	regular, _ := m.Measure("Hello", displaylist.RoleNode)
	bold, _ := m.Measure("Hello", displaylist.RoleClusterTitle) // bold role
	if bold <= regular {
		t.Fatalf("bold should be wider than regular for the same text: bold=%v regular=%v", bold, regular)
	}
}
```

- [ ] **Step 4.3: Run the test to verify it fails**

```bash
go test ./fontmetrics/...
```
Expected: FAIL — `fontmetrics` package doesn't exist.

- [ ] **Step 4.4: Implement `fontmetrics/inter.go`**

```go
// Package fontmetrics provides the default Measurer used by the
// layout stage when the caller doesn't supply one. The measurer is
// backed by Inter font metrics extracted from the embedded TTFs.
package fontmetrics

import (
	"sync"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/luo-studio/go-mermaid/fonts"
)

// faceStyle picks one of the embedded Inter variants.
type faceStyle int

const (
	faceRegular faceStyle = iota
	faceBold
	faceItalic
)

// metrics caches a parsed sfnt font and its overall vertical metrics.
type metrics struct {
	font       *sfnt.Font
	unitsPerEm int
	ascent     int // in font units
	descent    int // in font units, positive
}

var (
	loadOnce sync.Once
	loadErr  error
	cache    [3]*metrics
)

func ensureLoaded() error {
	loadOnce.Do(func() {
		for i, style := range []faceStyle{faceRegular, faceBold, faceItalic} {
			data, err := fonts.Bytes(fontStyleFor(style))
			if err != nil {
				loadErr = err
				return
			}
			f, err := sfnt.Parse(data)
			if err != nil {
				loadErr = err
				return
			}
			var b sfnt.Buffer
			upm := int(f.UnitsPerEm())
			fm, err := f.Metrics(&b, fixed.I(upm), 0)
			if err != nil {
				loadErr = err
				return
			}
			cache[i] = &metrics{
				font:       f,
				unitsPerEm: upm,
				ascent:     fm.Ascent.Ceil(),
				descent:    fm.Descent.Ceil(),
			}
		}
	})
	return loadErr
}

// advance returns the advance width of r in font units. If r is not
// in the font, half an em is returned as a fallback.
func (m *metrics) advance(r rune) int {
	var b sfnt.Buffer
	idx, err := m.font.GlyphIndex(&b, r)
	if err != nil || idx == 0 {
		return m.unitsPerEm / 2
	}
	adv, err := m.font.GlyphAdvance(&b, idx, fixed.I(m.unitsPerEm), 0)
	if err != nil {
		return m.unitsPerEm / 2
	}
	return adv.Ceil()
}

// fontStyleFor maps the local style enum to fonts.Style.
func fontStyleFor(s faceStyle) fonts.Style {
	switch s {
	case faceBold:
		return fonts.StyleBold
	case faceItalic:
		return fonts.StyleItalic
	default:
		return fonts.StyleRegular
	}
}
```

- [ ] **Step 4.5: Implement `fontmetrics/default.go`**

```go
package fontmetrics

import (
	"github.com/luo-studio/go-mermaid/displaylist"
)

// DefaultMeasurer measures text using the embedded Inter metrics.
//
// Bold is selected for roles that are typically bold in Mermaid
// diagrams (cluster titles, class names, entity names, actor titles).
// Italic is selected for class annotations.
//
// Heights are computed once per font size from ascent + descent and
// cached.
type DefaultMeasurer struct {
	fontSize float64
	regularLineHeight float64
	boldLineHeight    float64
	italicLineHeight  float64
}

// NewDefault returns a Measurer using the embedded Inter at the given
// point size. Panics if the embedded fonts can't be loaded — that
// signals a build / embedding bug.
func NewDefault(fontSize float64) *DefaultMeasurer {
	if err := ensureLoaded(); err != nil {
		panic("fontmetrics: cannot load embedded Inter: " + err.Error())
	}
	if fontSize <= 0 {
		fontSize = 14
	}
	d := &DefaultMeasurer{fontSize: fontSize}
	d.regularLineHeight = lineHeightFor(cache[faceRegular], fontSize)
	d.boldLineHeight = lineHeightFor(cache[faceBold], fontSize)
	d.italicLineHeight = lineHeightFor(cache[faceItalic], fontSize)
	return d
}

func lineHeightFor(m *metrics, fontSize float64) float64 {
	scale := fontSize / float64(m.unitsPerEm)
	return float64(m.ascent+m.descent) * scale
}

// Measure returns the rendered width and height of text in the
// Measurer's font at the requested role.
func (d *DefaultMeasurer) Measure(text string, role displaylist.Role) (w, h float64) {
	style := styleForRole(role)
	m := cache[style]
	scale := d.fontSize / float64(m.unitsPerEm)

	for _, r := range text {
		w += float64(m.advance(r)) * scale
	}
	switch style {
	case faceBold:
		h = d.boldLineHeight
	case faceItalic:
		h = d.italicLineHeight
	default:
		h = d.regularLineHeight
	}
	return
}

func styleForRole(role displaylist.Role) faceStyle {
	switch role {
	case displaylist.RoleClusterTitle,
		displaylist.RoleActorTitle,
		displaylist.RoleClassBox,
		displaylist.RoleEntityBox,
		displaylist.RoleStateBox:
		return faceBold
	case displaylist.RoleClassAnnotation:
		return faceItalic
	default:
		return faceRegular
	}
}
```

- [ ] **Step 4.6: Run the test to verify it passes**

```bash
go test ./fontmetrics/...
```
Expected: PASS, all four tests.

- [ ] **Step 4.7: Commit**

```bash
git add fontmetrics/ go.mod go.sum
git commit -m "fontmetrics: default Measurer using embedded Inter"
```

---

### Task 5: `fonts` — embed and expose Inter TTFs

**Files:**
- Create: `fonts/inter.go`
- Create: `fonts/loader.go`
- Create: `fonts/fonts_test.go`

- [ ] **Step 5.1: Write the failing test**

Create `fonts/fonts_test.go`:
```go
package fonts

import "testing"

func TestBytesNonEmpty(t *testing.T) {
	for _, st := range []Style{StyleRegular, StyleBold, StyleItalic} {
		b, err := Bytes(st)
		if err != nil {
			t.Fatalf("Bytes(%v): %v", st, err)
		}
		if len(b) < 1000 {
			t.Fatalf("Bytes(%v) too small: %d bytes (expected ~300 KB TTF)", st, len(b))
		}
		// TTF magic: 0x00010000 or "OTTO"
		magic := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		if magic != 0x00010000 && string(b[0:4]) != "OTTO" && string(b[0:4]) != "true" {
			t.Fatalf("Bytes(%v) does not look like a TrueType font: magic=%08x", st, magic)
		}
	}
}

func TestUnknownStyle(t *testing.T) {
	if _, err := Bytes(Style(99)); err == nil {
		t.Fatal("expected error for unknown style")
	}
}
```

- [ ] **Step 5.2: Run the test to verify it fails**

```bash
go test ./fonts/...
```
Expected: FAIL — package doesn't exist.

- [ ] **Step 5.3: Create `fonts/inter.go`**

```go
// Package fonts holds the embedded Inter TTFs that go-mermaid uses by
// default for the canvas rasterizer and font-metrics measurer.
//
// Inter is licensed under SIL Open Font License 1.1 (see
// LICENSE-Inter.txt). Embedding here keeps go-mermaid usable in
// containers without a system font cache.
package fonts

import _ "embed"

//go:embed Inter-Regular.ttf
var interRegular []byte

//go:embed Inter-Bold.ttf
var interBold []byte

//go:embed Inter-Italic.ttf
var interItalic []byte
```

- [ ] **Step 5.4: Create `fonts/loader.go`**

```go
package fonts

import "fmt"

// Style identifies an embedded font variant.
type Style int

const (
	StyleRegular Style = iota
	StyleBold
	StyleItalic
)

// Bytes returns the raw TTF bytes for the requested style. The
// returned slice MUST NOT be mutated — it backs the embedded constant.
func Bytes(s Style) ([]byte, error) {
	switch s {
	case StyleRegular:
		return interRegular, nil
	case StyleBold:
		return interBold, nil
	case StyleItalic:
		return interItalic, nil
	default:
		return nil, fmt.Errorf("fonts: unknown style %d", s)
	}
}
```

- [ ] **Step 5.5: Run the test to verify it passes**

```bash
go test ./fonts/...
```
Expected: PASS.

- [ ] **Step 5.6: Commit**

```bash
git add fonts/inter.go fonts/loader.go fonts/fonts_test.go
git commit -m "fonts: embed Inter Regular/Bold/Italic + Style enum"
```

---

### Task 6: `autog` adapter

A thin wrapper over `github.com/nulab/autog` that takes a list of nodes (id + size) and edges, runs autog's default pipeline via `graph.EdgeSlice`, and returns positions and edge waypoints. No cluster handling here — that lands in Phase 3.

**Files:**
- Create: `autog/adapter.go`
- Create: `autog/adapter_test.go`

- [ ] **Step 6.1: Add the autog dep**

```bash
go get github.com/nulab/autog@latest
```
Expected: `go.mod` and `go.sum` updated.

- [ ] **Step 6.2: Write the failing test**

Create `autog/adapter_test.go`:
```go
package autog

import "testing"

func TestLayoutChainOfThree(t *testing.T) {
	in := Input{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
			{ID: "C", Width: 30, Height: 20},
		},
		Edges: []Edge{
			{FromID: "A", ToID: "B"},
			{FromID: "B", ToID: "C"},
		},
		Direction:    DirectionTB,
		NodeSpacing:  10,
		LayerSpacing: 30,
	}

	out, err := Layout(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.Nodes) != 3 {
		t.Fatalf("expected 3 positioned nodes, got %d", len(out.Nodes))
	}
	if len(out.Edges) != 2 {
		t.Fatalf("expected 2 positioned edges, got %d", len(out.Edges))
	}

	// In top-bottom layout, A.Y < B.Y < C.Y
	posOf := func(id string) Node {
		for _, n := range out.Nodes {
			if n.ID == id {
				return n
			}
		}
		t.Fatalf("node %q not in layout", id)
		return Node{}
	}
	a, b, c := posOf("A"), posOf("B"), posOf("C")
	if !(a.Y < b.Y && b.Y < c.Y) {
		t.Fatalf("expected A.Y < B.Y < C.Y, got %v %v %v", a.Y, b.Y, c.Y)
	}

	// Bbox is non-zero
	if out.Width <= 0 || out.Height <= 0 {
		t.Fatalf("expected positive bbox, got %vx%v", out.Width, out.Height)
	}
}

func TestLayoutTwoNodesOneEdge(t *testing.T) {
	// Smallest non-degenerate case. Single-node-no-edges is deferred —
	// see Open Questions.
	in := Input{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
		},
		Edges: []Edge{{FromID: "A", ToID: "B"}},
	}
	out, err := Layout(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(out.Nodes))
	}
}

func TestLayoutEmpty(t *testing.T) {
	out, err := Layout(Input{})
	if err != nil {
		t.Fatalf("Layout(empty): %v", err)
	}
	if len(out.Nodes) != 0 || len(out.Edges) != 0 {
		t.Fatalf("expected empty layout, got %v", out)
	}
	if out.Width != 0 || out.Height != 0 {
		t.Fatalf("expected zero bbox, got %vx%v", out.Width, out.Height)
	}
}
```

- [ ] **Step 6.3: Run the test to verify it fails**

```bash
go test ./autog/...
```
Expected: FAIL — `autog` package doesn't exist.

- [ ] **Step 6.4: Create `autog/adapter.go`**

```go
// Package autog wraps github.com/nulab/autog with a smaller surface
// suited to go-mermaid's layout stage. It is a thin adapter — no
// cluster recursion (added in Phase 3), no per-diagram-type
// knowledge.
package autog

import (
	"fmt"
	"runtime"

	"github.com/nulab/autog"
	autoggraph "github.com/nulab/autog/graph"
)

// Direction matches Mermaid's flowchart direction.
type Direction int

const (
	DirectionTB Direction = iota // top to bottom (graphviz "TB"/"TD")
	DirectionBT                  // bottom to top
	DirectionLR                  // left to right
	DirectionRL                  // right to left
)

// Node is a layout input node. Width and Height are in DisplayList
// units; the returned Node has X,Y filled in and the same dimensions.
type Node struct {
	ID            string
	Width, Height float64
	X, Y          float64
}

// Edge is a layout input edge. After Layout, Points is the
// polyline waypoints in DisplayList units.
type Edge struct {
	FromID string
	ToID   string
	Points [][2]float64
}

// Input is the cumulative layout request.
type Input struct {
	Nodes        []Node
	Edges        []Edge
	Direction    Direction
	NodeSpacing  float64 // px between sibling nodes; default 24
	LayerSpacing float64 // px between layers; default 40
	Padding      float64 // outer padding; default 0
}

// Output carries the positioned graph.
type Output struct {
	Width, Height float64
	Nodes         []Node
	Edges         []Edge
}

// Layout runs the autog pipeline. Returns an empty Output if Input
// has no nodes. Wraps autog panics into errors so a malformed input
// doesn't crash the caller; runtime panics (nil deref, OOB) are
// re-raised so real bugs surface.
func Layout(in Input) (out Output, err error) {
	if len(in.Nodes) == 0 {
		return Output{}, nil
	}

	defer func() {
		if r := recover(); r != nil {
			if _, isRuntime := r.(runtime.Error); isRuntime {
				panic(r)
			}
			err = fmt.Errorf("autog: layout panic: %v", r)
		}
	}()

	nodeSpacing := in.NodeSpacing
	if nodeSpacing == 0 {
		nodeSpacing = 24
	}
	layerSpacing := in.LayerSpacing
	if layerSpacing == 0 {
		layerSpacing = 40
	}

	// Build the edge list as the [][]string adjacency form autog's
	// graph.EdgeSlice accepts.
	adj := make([][]string, len(in.Edges))
	for i, e := range in.Edges {
		adj[i] = []string{e.FromID, e.ToID}
	}
	src := autoggraph.EdgeSlice(adj)

	// Provide per-node sizes via WithNodeSize. (Isolated nodes — i.e.
	// nodes with no edges — won't appear in the EdgeSlice; for Phase 1
	// the caller is responsible for ensuring every Node ID also
	// appears as an endpoint of some Edge. Phase 2's flowchart parser
	// always emits at least one self-loop or sentinel edge per node;
	// see the layout package's docstring for the full contract.)
	autogSizes := make(map[string]autoggraph.Size, len(in.Nodes))
	for _, n := range in.Nodes {
		autogSizes[n.ID] = autoggraph.Size{W: n.Width, H: n.Height}
	}

	layout := autog.Layout(
		src,
		autog.WithNodeSize(autogSizes),
		autog.WithNodeSpacing(nodeSpacing),
		autog.WithLayerSpacing(layerSpacing),
	)

	// autog's coordinate system: depending on direction, we may want to
	// rotate. Phase 1 supports DirectionTB only — other directions land
	// in Phase 2 once the flowchart parser exposes them.
	_ = in.Direction

	// Map autog output back into our types.
	out.Nodes = make([]Node, 0, len(layout.Nodes))
	maxX, maxY := 0.0, 0.0
	for _, n := range layout.Nodes {
		nn := Node{
			ID:     n.ID,
			Width:  n.Size.W,
			Height: n.Size.H,
			X:      n.X,
			Y:      n.Y,
		}
		out.Nodes = append(out.Nodes, nn)
		if rx := nn.X + nn.Width; rx > maxX {
			maxX = rx
		}
		if ry := nn.Y + nn.Height; ry > maxY {
			maxY = ry
		}
	}
	out.Edges = make([]Edge, 0, len(layout.Edges))
	for _, e := range layout.Edges {
		ee := Edge{FromID: e.FromID, ToID: e.ToID, Points: append([][2]float64{}, e.Points...)}
		out.Edges = append(out.Edges, ee)
		for _, p := range ee.Points {
			if p[0] > maxX {
				maxX = p[0]
			}
			if p[1] > maxY {
				maxY = p[1]
			}
		}
	}
	out.Width = maxX + in.Padding*2
	out.Height = maxY + in.Padding*2
	return out, nil
}
```

> Note: the exact `autog.WithNodeSize` and `autog.WithNodeSpacing`
> functional-option names match nulab/autog's public API. Verify with
> `go doc github.com/nulab/autog` before writing — if a name has
> changed, adapt the call but keep the wrapper signature unchanged.

- [ ] **Step 6.6: Run the test to verify it passes**

```bash
go test ./autog/...
```
Expected: PASS, all three tests.

- [ ] **Step 6.7: Commit**

```bash
git add autog/ go.mod go.sum
git commit -m "autog: thin adapter over nulab/autog (no clusters)"
```

---

### Task 7: Top-level `mermaid` package — `Measurer`, `LayoutOptions`, `ParseAndLayout`

This is the public entry point. In Phase 1 it recognizes diagram-type keywords and dispatches, but every dispatch arm returns `ErrUnknownDiagram` because no parsers exist yet. Phase 2 will replace the flowchart arm with a real call.

**Files:**
- Create: `mermaid.go`
- Create: `mermaid_test.go`

- [ ] **Step 7.1: Write the failing test**

Create `mermaid_test.go`:
```go
package mermaid

import (
	"errors"
	"testing"
)

func TestDetectType(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want diagramType
	}{
		{"flowchart-TB", "flowchart TB\n  A --> B\n", typeFlowchart},
		{"graph-LR", "graph LR\n  A --> B\n", typeFlowchart},
		{"sequence", "sequenceDiagram\n  A->>B: hi\n", typeSequence},
		{"class", "classDiagram\n  A <|-- B\n", typeClass},
		{"er", "erDiagram\n  A ||--o{ B : has\n", typeER},
		{"state-v2", "stateDiagram-v2\n  [*] --> S\n", typeState},
		{"state", "stateDiagram\n  [*] --> S\n", typeState},
		{"unknown", "garbage\n", typeUnknown},
		{"empty", "", typeUnknown},
		{"comments-then-flowchart", "%% comment\nflowchart TB\nA --> B", typeFlowchart},
		{"leading-blank-lines", "\n\nflowchart TB\nA --> B", typeFlowchart},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectType(tc.src); got != tc.want {
				t.Errorf("detectType(%q) = %v, want %v", tc.src, got, tc.want)
			}
		})
	}
}

func TestParseAndLayoutUnknownDiagram(t *testing.T) {
	_, err := ParseAndLayout("not a diagram", LayoutOptions{})
	if !errors.Is(err, ErrUnknownDiagram) {
		t.Fatalf("expected ErrUnknownDiagram, got %v", err)
	}
}

func TestParseAndLayoutKnownButUnimplemented(t *testing.T) {
	// Phase 1: every diagram type returns ErrNotImplemented (or wraps it).
	_, err := ParseAndLayout("flowchart TB\nA --> B\n", LayoutOptions{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestLayoutOptionsZeroValueValid(t *testing.T) {
	// Zero-value LayoutOptions must be a valid input — Measurer nil →
	// default; spacings 0 → defaults inside layout.
	var opts LayoutOptions
	if opts.Measurer != nil {
		t.Fatal("zero LayoutOptions should have nil Measurer")
	}
}
```

- [ ] **Step 7.2: Run the test to verify it fails**

```bash
go test .
```
Expected: FAIL — `mermaid` package not yet present.

- [ ] **Step 7.3: Implement `mermaid.go`**

```go
// Package mermaid is the top-level entry point for go-mermaid. It
// detects the diagram type from source, dispatches to the
// appropriate per-type package, and returns a style-neutral
// DisplayList.
//
// Per-diagram-type packages (flowchart, sequence, class, er, state)
// expose their own Parse/Layout pairs for callers that want lower-
// level access; ParseAndLayout is the convenience entry.
//
// Phase 1 status: detection works; every dispatch arm returns
// ErrNotImplemented. Real parsers ship in Phase 2+.
package mermaid

import (
	"errors"
	"strings"

	"github.com/luo-studio/go-mermaid/displaylist"
)

// Measurer reports the rendered width and height of a string in the
// caller's font for the given semantic Role. Implementations are
// expected to be deterministic for the same input.
type Measurer interface {
	Measure(text string, role displaylist.Role) (w, h float64)
}

// LayoutOptions are common knobs shared across diagram types. Per-
// type packages may consume additional fields here (Sequence*).
type LayoutOptions struct {
	// Measurer measures rendered text. If nil, layout uses the
	// embedded Inter metrics measurer at FontSize.
	Measurer Measurer

	// FontSize used by the default Measurer in DisplayList units
	// (typically points). Default: 14.
	FontSize float64

	// Padding around the diagram's bbox.
	Padding float64

	// NodeSpacing is the horizontal/sibling spacing autog uses.
	// Default: 24.
	NodeSpacing float64

	// LayerSpacing is the vertical/cross-layer spacing autog uses.
	// Default: 40.
	LayerSpacing float64

	// SequenceActorSpacing/SequenceMessageSpacing are sequence-
	// specific. Ignored for non-sequence diagrams.
	SequenceActorSpacing   float64
	SequenceMessageSpacing float64
}

// Errors.
var (
	ErrUnknownDiagram = errors.New("mermaid: unrecognized diagram type")
	ErrNotImplemented = errors.New("mermaid: diagram type recognized but not implemented in this build")
)

// ParseAndLayout detects the diagram type from src, runs the parser
// and layout for that type, and returns the resulting DisplayList.
func ParseAndLayout(src string, opts LayoutOptions) (*displaylist.DisplayList, error) {
	switch detectType(src) {
	case typeFlowchart, typeSequence, typeClass, typeER, typeState:
		return nil, ErrNotImplemented
	default:
		return nil, ErrUnknownDiagram
	}
}

// diagramType is an internal enum used by detectType.
type diagramType int

const (
	typeUnknown diagramType = iota
	typeFlowchart
	typeSequence
	typeClass
	typeER
	typeState
)

// detectType returns the diagram type implied by src's first non-
// blank, non-comment line. Mirrors mermaid.js's detection.
func detectType(src string) diagramType {
	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "%%") {
			continue
		}
		head := strings.ToLower(strings.Fields(line)[0])
		switch {
		case head == "flowchart" || head == "graph":
			return typeFlowchart
		case head == "sequencediagram":
			return typeSequence
		case head == "classdiagram":
			return typeClass
		case head == "erdiagram":
			return typeER
		case head == "statediagram" || head == "statediagram-v2":
			return typeState
		default:
			return typeUnknown
		}
	}
	return typeUnknown
}
```

- [ ] **Step 7.4: Run the test to verify it passes**

```bash
go test .
```
Expected: PASS, all four tests.

- [ ] **Step 7.5: Commit**

```bash
git add mermaid.go mermaid_test.go
git commit -m "mermaid: top-level ParseAndLayout, Measurer, diagram-type detection"
```

---

### Task 8: Wire the default Measurer into ParseAndLayout

The `LayoutOptions.Measurer` field can be nil; layout code needs a way to obtain the default. Add a small helper. This isn't visible until Phase 2 dispatch arms call layout functions, but we land it now so the option semantics are nailed down.

**Files:**
- Modify: `mermaid.go` (add `defaultMeasurer` helper)
- Modify: `mermaid_test.go` (assert default is used)

- [ ] **Step 8.1: Write the failing test**

Append to `mermaid_test.go`:
```go
func TestLayoutOptionsDefaultMeasurerNotNil(t *testing.T) {
	var opts LayoutOptions
	m := opts.measurer()
	if m == nil {
		t.Fatal("opts.measurer() must never return nil")
	}
	w, h := m.Measure("Hello", "")
	if w <= 0 || h <= 0 {
		t.Fatalf("default measurer should report positive metrics, got %vx%v", w, h)
	}
}

func TestLayoutOptionsCustomMeasurerWins(t *testing.T) {
	called := false
	custom := measurerFunc(func(text string, role displaylist.Role) (float64, float64) {
		called = true
		return 42, 42
	})
	opts := LayoutOptions{Measurer: custom}
	w, h := opts.measurer().Measure("anything", "")
	if !called || w != 42 || h != 42 {
		t.Fatalf("custom measurer was not preferred: called=%v w=%v h=%v", called, w, h)
	}
}

// measurerFunc lets a plain func satisfy Measurer in tests.
type measurerFunc func(text string, role displaylist.Role) (float64, float64)

func (f measurerFunc) Measure(text string, role displaylist.Role) (float64, float64) {
	return f(text, role)
}
```

(`displaylist` import needs to be present in `mermaid_test.go`; add `"github.com/luo-studio/go-mermaid/displaylist"` to its imports.)

- [ ] **Step 8.2: Run the test to verify it fails**

```bash
go test .
```
Expected: FAIL — `opts.measurer()` does not exist.

- [ ] **Step 8.3: Implement the helper**

Append to `mermaid.go`:
```go
import "github.com/luo-studio/go-mermaid/fontmetrics"

// measurer returns the Measurer to use: the explicit one if set,
// otherwise a default backed by the embedded Inter metrics.
func (o LayoutOptions) measurer() Measurer {
	if o.Measurer != nil {
		return o.Measurer
	}
	fs := o.FontSize
	if fs <= 0 {
		fs = 14
	}
	return fontmetrics.NewDefault(fs)
}
```

(Adjust the existing `import` block to merge — Go won't allow two `import` blocks; fold the new import into the existing one.)

Also: the `fontmetrics.DefaultMeasurer` returned by `NewDefault` must satisfy `mermaid.Measurer`. Verify the method signature matches: `Measure(text string, role displaylist.Role) (w, h float64)`. If the existing `fontmetrics.DefaultMeasurer.Measure` doesn't return named results, that's still fine — Go accepts the implementation regardless of named returns.

- [ ] **Step 8.4: Run the test to verify it passes**

```bash
go test .
```
Expected: PASS.

- [ ] **Step 8.5: Commit**

```bash
git add mermaid.go mermaid_test.go
git commit -m "mermaid: default Measurer fallback in LayoutOptions"
```

---

### Task 9: Module-wide vet / lint sanity check + final integration test

Make sure the whole module builds, runs all tests, and no package leaks an unintended panic in normal use.

**Files:**
- Create: `mermaid_smoke_test.go`

- [ ] **Step 9.1: Add the smoke test**

Create `mermaid_smoke_test.go` (yes, in addition to `mermaid_test.go` — keeps the smoke test discoverable):
```go
package mermaid

import (
	"errors"
	"testing"
)

// Smoke: the Phase 1 happy path is "library compiles, recognises a
// diagram, returns ErrNotImplemented without panicking, and the
// default measurer works end-to-end."
func TestPhase1Smoke(t *testing.T) {
	src := "flowchart TB\n  A --> B\n"

	dl, err := ParseAndLayout(src, LayoutOptions{})
	if dl != nil {
		t.Fatal("expected nil DisplayList until Phase 2")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	// Default Measurer end-to-end.
	w, h := (LayoutOptions{}).measurer().Measure("Hello", "")
	if w <= 0 || h <= 0 {
		t.Fatalf("default measurer broke: %vx%v", w, h)
	}
}
```

- [ ] **Step 9.2: Run the whole test suite**

```bash
go test ./...
```
Expected: all packages PASS.

- [ ] **Step 9.3: Vet**

```bash
go vet ./...
```
Expected: no output (clean).

- [ ] **Step 9.4: Commit**

```bash
git add mermaid_smoke_test.go
git commit -m "mermaid: phase 1 smoke test"
```

- [ ] **Step 9.5: Push the branch**

```bash
git push -u origin spec/initial-design
```

(or wait for the user to direct merge / PR creation — see "Finishing" below.)

---

## Self-Review

After all tasks pass, run a final pass against the spec:

| Spec section | Phase 1 task |
|---|---|
| Module path `github.com/luo-studio/go-mermaid` | Task 1 |
| `displaylist` types (Shape/Edge/Text/Cluster/Marker, Role/Kind enums) | Tasks 2, 3 |
| Embedded Inter Regular/Bold/Italic | Tasks 1, 5 |
| Default Measurer using embedded Inter | Task 4 |
| Hybrid measurement (caller can override) | Tasks 7, 8 |
| `autog` adapter, no clusters | Task 6 |
| Top-level `ParseAndLayout` with diagram-type detection | Task 7 |
| Style-neutral DisplayList primitives | Tasks 2, 3 |
| Standard role set | Task 2 (`role.go`) |
| Error model: `ErrUnknownDiagram`, parse/layout panic recovery | Tasks 6, 7 (autog adapter recovers panics; `ErrUnknownDiagram` and `ErrNotImplemented` defined) |
| Test scaffolding | Tasks 2, 3, 4, 5, 6, 7, 8, 9 (every task ships tests) |

**Items deferred to subsequent phases:**

| Spec area | Phase |
|---|---|
| Flowchart parser, AST, layout (no clusters) | Phase 2 |
| Cluster recursion in `autog` adapter | Phase 3 |
| Sequence parser + hand-rolled layout | Phase 4 |
| Class parser + autog-based layout | Phase 4 |
| ER parser + autog-based layout | Phase 4 |
| State parser + autog-based layout | Phase 4 |
| `pdf` emitter (DrawInto, DrawMermaid) + Style/RoleStyle | Phase 5 |
| `canvasr` emitter (RenderPNG, RenderInto) | Phase 5 |
| `cmd/parse`, `cmd/render` | Phase 5 |
| Property tests (no-panic, bbox containment, cluster containment) | Phases 2-4 (added with each diagram type) |

## Finishing

After Task 9, the branch `spec/initial-design` has 8-9 commits. Open
a PR titled "Phase 1: Foundation packages" against `main`. Once
reviewed and merged, write the Phase 2 plan
(`docs/superpowers/plans/<date>-phase2-flowchart.md`) before starting
implementation.

## Open Questions / Risks

- **autog functional options**: the `WithNodeSize` / `WithNodeSpacing`
  / `WithLayerSpacing` names in Task 6 are best-effort. The
  implementer should verify against `go doc github.com/nulab/autog`
  and adjust if the upstream API has different names. The Input/Output
  types stay stable regardless.
- **Inter sfnt parsing cost**: parsing three TTFs at first use takes a
  few ms. `sync.Once` ensures it happens once per process — fine for
  PDF rendering loops; may be observable in microbenchmarks.
- **No emitter in Phase 1**: there's no end-to-end "render → bytes"
  test. The smoke test only confirms detection + default measurer.
  This is intentional — emitters need real DisplayList content to
  exercise, which arrives in Phase 2.
- **Isolated nodes in `autog.Layout`**: `graph.EdgeSlice` discovers
  nodes via edge endpoints, so a node with no incident edges is
  invisible to autog. Phase 1 declines to address this — every test
  case has at least one edge per node. If Phase 2's flowchart parser
  produces real diagrams with isolated nodes, the adapter will need a
  fallback (e.g., place isolated nodes in a row below the autog'd
  graph). Track this when Phase 2 lands.
