# go-mermaid Phase 8 — Canvas Rasterizer + CLI Binaries

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development.

**Goal:** Final phase — add the `canvasr` package (DisplayList → PNG via tdewolff/canvas, mirroring the role-driven `pdf` emitter) and the two CLI binaries `cmd/parse` (stdin .mmd → JSON AST) and `cmd/render` (stdin .mmd → PDF/PNG). After this phase the library is feature-complete per the spec.

**Architecture:** `canvasr` parallels `pdf`: same `Style`/`RoleStyle` types, same `DrawInto`/`RenderPNG`/`RenderInto` shape, but draws into a `tdewolff/canvas` surface using the embedded Inter TTFs from Phase 1's `fonts` package. The CLI binaries are thin wrappers — `cmd/parse` writes a JSON AST (one of `flowchart.Diagram`, `sequence.Diagram`, etc., wrapped in a discriminator); `cmd/render` writes either PDF or PNG depending on `-format`.

**Depends on:** Phases 1–7 (all diagram types must work for `cmd/render` to exercise them).

---

## Spec Reference

Spec section "Canvas rasterizer (`canvasr/`)" + "Implementation Order — Phase 5 (Emitters)".

## File Structure

```
go-mermaid/
├── canvasr/
│   ├── render.go           # RenderPNG, RenderInto, RenderOptions, Style alias
│   ├── draw_shape.go       # per ShapeKind
│   ├── draw_edge.go
│   ├── draw_text.go
│   ├── style.go            # canvasr-specific defaults (re-using displaylist.Role)
│   └── render_test.go
├── cmd/
│   ├── parse/
│   │   └── main.go         # stdin .mmd → stdout JSON AST
│   └── render/
│       └── main.go         # stdin .mmd → stdout PDF or PNG
└── README.md               # MODIFIED: usage examples
```

## Tasks

### Task 1: canvasr — Style, RenderOptions, RenderInto skeleton

**Files:** Create `canvasr/style.go`, `canvasr/render.go`, `canvasr/render_test.go`.

- [ ] **Step 1.1: Add tdewolff/canvas dep**

```bash
go get github.com/tdewolff/canvas
```

- [ ] **Step 1.2: Implement `canvasr/style.go`**

Mirror `pdf/style.go` but with `canvas.RGBA` colors and `font.Face` references:

```go
package mermaidcanvasr

import (
	"github.com/tdewolff/canvas"
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/fonts"
)

type RoleStyle struct {
	Stroke      canvas.Paint    // canvas.Transparent for "no stroke"
	StrokeWidth float64         // mm
	DashPattern []float64
	Fill        canvas.Paint
	TextPaint   canvas.Paint
	FontFamily  *canvas.FontFamily
	FontStyle   canvas.FontStyle
	FontSize    float64         // pt
}

type Style struct {
	Roles   map[displaylist.Role]RoleStyle
	Default RoleStyle
}

func (s Style) lookup(role displaylist.Role) RoleStyle {
	if r, ok := s.Roles[role]; ok { return r }
	return s.Default
}

// DefaultStyle uses embedded Inter for all roles, black on white.
func DefaultStyle() (Style, error) {
	regBytes, _ := fonts.Bytes(fonts.StyleRegular)
	boldBytes, _ := fonts.Bytes(fonts.StyleBold)
	italicBytes, _ := fonts.Bytes(fonts.StyleItalic)
	fam := canvas.NewFontFamily("Inter")
	if err := fam.LoadFontFile("Inter-Regular", regBytes, canvas.FontRegular); err != nil { return Style{}, err }
	if err := fam.LoadFontFile("Inter-Bold", boldBytes, canvas.FontBold); err != nil { return Style{}, err }
	if err := fam.LoadFontFile("Inter-Italic", italicBytes, canvas.FontItalic); err != nil { return Style{}, err }
	body := RoleStyle{
		Stroke: canvas.Black, StrokeWidth: 0.3,
		Fill: canvas.Transparent,
		TextPaint: canvas.Black,
		FontFamily: fam, FontStyle: canvas.FontRegular, FontSize: 10,
	}
	bold := body
	bold.FontStyle = canvas.FontBold
	muted := body
	muted.TextPaint = canvas.RGBA{R: 100, G: 100, B: 100, A: 255}
	return Style{
		Default: body,
		Roles: map[displaylist.Role]RoleStyle{
			displaylist.RoleNode: body,
			displaylist.RoleEdge: body,
			displaylist.RoleEdgeLabel: muted,
			displaylist.RoleClusterTitle: bold,
			displaylist.RoleSubgraph: body,
			displaylist.RoleActorBox: body,
			displaylist.RoleActorTitle: bold,
			displaylist.RoleLifeline: muted,
			displaylist.RoleClassBox: body,
			displaylist.RoleClassMember: muted,
			displaylist.RoleEntityBox: bold,
			displaylist.RoleEntityAttribute: muted,
			displaylist.RoleStateBox: body,
		},
	}, nil
}
```

- [ ] **Step 1.3: Implement `canvasr/render.go` skeleton**

```go
package mermaidcanvasr

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/tdewolff/canvas"
	canvasrenderers "github.com/tdewolff/canvas/renderers"

	mermaid "github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/displaylist"
)

type RenderOptions struct {
	Style    Style
	Layout   mermaid.LayoutOptions
	DPI      float64 // for PNG; default 192
	MaxWidth float64 // pixels; 0 = no cap
	Padding  float64 // mm
}

func defaultOptions() RenderOptions {
	st, _ := DefaultStyle()
	return RenderOptions{Style: st, DPI: 192}
}

func RenderPNG(src string, opts RenderOptions) ([]byte, error) {
	dl, err := mermaid.ParseAndLayout(src, opts.Layout)
	if err != nil { return nil, err }
	if dl == nil { return nil, fmt.Errorf("canvasr: empty diagram") }
	return RenderDisplayListPNG(dl, opts)
}

func RenderDisplayListPNG(dl *displaylist.DisplayList, opts RenderOptions) ([]byte, error) {
	style := opts.Style
	if style.Default == (RoleStyle{}) && len(style.Roles) == 0 {
		var err error
		style, err = DefaultStyle()
		if err != nil { return nil, err }
	}
	dpi := opts.DPI
	if dpi <= 0 { dpi = 192 }
	pad := opts.Padding
	c := canvas.New(dl.Width+pad*2, dl.Height+pad*2)
	ctx := canvas.NewContext(c)
	ctx.SetCoordSystem(canvas.CartesianIV) // (0,0) at top-left, y grows down — matches DisplayList
	ctx.Translate(pad, pad)

	for _, it := range dl.Items {
		switch v := it.(type) {
		case displaylist.Cluster:
			drawClusterCanvas(ctx, v, style.lookup(v.Role))
		}
	}
	for _, it := range dl.Items {
		switch v := it.(type) {
		case displaylist.Shape:
			drawShapeCanvas(ctx, v, style.lookup(v.Role))
		case displaylist.Edge:
			drawEdgeCanvas(ctx, v, style.lookup(v.Role))
		case displaylist.Text:
			drawTextCanvas(ctx, v, style.lookup(v.Role))
		case displaylist.Marker:
			// inline; rare
		}
	}

	var buf bytes.Buffer
	img := canvasrenderers.PNG(canvas.DPMM(dpi/25.4))
	if err := img(&buf, c); err != nil { return nil, err }
	// (depending on tdewolff/canvas version, the rasterizer API may
	// be `c.WriteFile` or via `renderers.PNG(c, w)` — adapt at impl time)
	_ = png.Encoder{}
	return buf.Bytes(), nil
}
```

- [ ] **Step 1.4: Implement `canvasr/draw_shape.go`, `draw_edge.go`, `draw_text.go`** mirroring the PDF emitter's logic but using `*canvas.Context`'s `DrawPath` / `DrawText` / `Rectangle` / `Circle` etc. For each ShapeKind, build a `canvas.Path` and call `ctx.DrawPath(0, 0, path)` after applying paint/stroke. Refer to tdewolff/canvas docs for the exact paint API.
- [ ] **Step 1.5: Test** — round-trip a small flowchart through `RenderPNG`, decode the PNG, confirm dimensions are non-zero. Commit: `git commit -m "canvasr: PNG rendering pipeline"`

---

### Task 2: cmd/parse

**Files:** Create `cmd/parse/main.go`.

```go
// Command parse reads a Mermaid diagram from stdin and writes its
// AST as JSON to stdout. Used for debugging and for piping into
// downstream tools.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/luo-studio/go-mermaid"
	"github.com/luo-studio/go-mermaid/class"
	"github.com/luo-studio/go-mermaid/er"
	"github.com/luo-studio/go-mermaid/flowchart"
	"github.com/luo-studio/go-mermaid/sequence"
	"github.com/luo-studio/go-mermaid/state"
)

type wireOutput struct {
	Type string      `json:"type"`
	AST  interface{} `json:"ast"`
}

func main() {
	src, err := io.ReadAll(os.Stdin)
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

	t := mermaid.DetectTypeForCmd(string(src)) // expose as exported helper
	var ast interface{}
	switch t {
	case "flowchart":
		ast, err = flowchart.Parse(string(src))
	case "sequence":
		ast, err = sequence.Parse(string(src))
	case "class":
		ast, err = class.Parse(string(src))
	case "er":
		ast, err = er.Parse(string(src))
	case "state":
		ast, err = state.Parse(string(src))
	default:
		fmt.Fprintln(os.Stderr, "mermaid: unrecognised diagram type")
		os.Exit(1)
	}
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(wireOutput{Type: t, AST: ast}); err != nil {
		fmt.Fprintln(os.Stderr, err); os.Exit(1)
	}
}
```

(`mermaid.DetectTypeForCmd` is an exported wrapper around the existing `detectType` so cmd code can reach it without exposing the internal enum.)

- [ ] Test by piping a fixture: `cat testdata/flowchart/parse/simple-tb.mmd | go run ./cmd/parse | jq .type` → "flowchart". Commit: `git commit -m "cmd/parse: stdin .mmd → JSON AST"`

---

### Task 3: cmd/render

**Files:** Create `cmd/render/main.go`.

```go
// Command render reads a Mermaid diagram from stdin and writes a
// rendered PDF or PNG to stdout.
//
// Usage:
//   render -format pdf < diagram.mmd > diagram.pdf
//   render -format png < diagram.mmd > diagram.png
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"codeberg.org/go-pdf/fpdf"

	mermaid "github.com/luo-studio/go-mermaid"
	mermaidcanvasr "github.com/luo-studio/go-mermaid/canvasr"
	mermaidpdf "github.com/luo-studio/go-mermaid/pdf"
)

func main() {
	format := flag.String("format", "pdf", "output format: pdf|png")
	flag.Parse()

	src, err := io.ReadAll(os.Stdin)
	if err != nil { fail(err) }

	switch *format {
	case "pdf":
		pdf := fpdf.New("P", "mm", "A4", "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.AddPage()
		opts := mermaidpdf.EmbedDefaults()
		if err := mermaidpdf.DrawMermaid(pdf, string(src), 10, 10, opts); err != nil { fail(err) }
		if err := pdf.Output(os.Stdout); err != nil { fail(err) }
	case "png":
		opts := mermaidcanvasr.RenderOptions{}
		out, err := mermaidcanvasr.RenderPNG(string(src), opts)
		if err != nil { fail(err) }
		if _, err := io.Copy(os.Stdout, bytes.NewReader(out)); err != nil { fail(err) }
	default:
		fail(fmt.Errorf("unknown format %q (want pdf|png)", *format))
	}
	_ = mermaid.LayoutOptions{}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```

- [ ] Test:
```bash
cat testdata/flowchart/parse/simple-tb.mmd | go run ./cmd/render -format pdf > /tmp/out.pdf
cat testdata/flowchart/parse/simple-tb.mmd | go run ./cmd/render -format png > /tmp/out.png
file /tmp/out.pdf /tmp/out.png
```
Both files should be reported as PDF / PNG image data. Commit: `git commit -m "cmd/render: stdin .mmd → PDF/PNG"`

---

### Task 4: Integration tests covering all diagram types

**Files:** Create `cmd/render/main_test.go`.

```go
package main

// Build-time integration test: the cmd binary compiles. Behavioral
// tests live in the per-diagram-type packages.
import "testing"

func TestCompiles(t *testing.T) {
	_ = main // referenced so go test ./... links the binary
}
```

A more thorough test runs each canonical fixture through both `pdf` and `canvasr` paths and asserts non-empty output. Add to a `cmd/render_integration_test.go`:

```go
// +build integration

package main_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestRenderEachDiagramType(t *testing.T) {
	cases := map[string]string{
		"flowchart": "flowchart TB\nA --> B\n",
		"sequence":  "sequenceDiagram\nA->>B: hi\n",
		"class":     "classDiagram\nA <|-- B\n",
		"er":        "erDiagram\nA ||--o{ B : has\n",
		"state":     "stateDiagram-v2\n[*] --> S\n",
	}
	for kind, src := range cases {
		for _, fmt := range []string{"pdf", "png"} {
			t.Run(kind+"/"+fmt, func(t *testing.T) {
				cmd := exec.Command("go", "run", "./cmd/render", "-format", fmt)
				cmd.Stdin = strings.NewReader(src)
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("%s/%s: %v", kind, fmt, err)
				}
				if len(out) < 100 {
					t.Fatalf("%s/%s: output too small (%d bytes)", kind, fmt, len(out))
				}
				if fmt == "pdf" && !bytes.HasPrefix(out, []byte("%PDF-")) {
					t.Fatalf("%s/pdf: not a PDF", kind)
				}
				if fmt == "png" && !bytes.HasPrefix(out, []byte{0x89, 0x50, 0x4e, 0x47}) {
					t.Fatalf("%s/png: not a PNG", kind)
				}
			})
		}
	}
}
```

Run with `go test -tags=integration ./cmd/render/...`.

- [ ] Commit: `git commit -m "cmd/render: cross-diagram integration tests"`

---

### Task 5: README + usage docs

**Files:** Modify `README.md`.

Add usage examples:
```markdown
## Usage

```go
import (
    "codeberg.org/go-pdf/fpdf"
    mermaidpdf "github.com/luo-studio/go-mermaid/pdf"
)

pdf := fpdf.New("P", "mm", "A4", "")
pdf.AddPage()
err := mermaidpdf.DrawMermaid(pdf, mmdSource, 10, 10, mermaidpdf.EmbedDefaults())
```

## Status

All 5 diagram types supported (flowchart, sequence, class, ER, state)
at mermaigo parity. PDF and PNG output via fpdf and tdewolff/canvas.
```

- [ ] Commit: `git commit -m "README: usage + status"`

---

### Task 6: Phase 8 smoke test

**Files:** `mermaid_smoke_test.go` (final replacement).

```go
package mermaid

import (
	"errors"
	"testing"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func TestPhase8AllDiagramTypes(t *testing.T) {
	cases := map[string]string{
		"flowchart": "flowchart TB\nA --> B\n",
		"sequence":  "sequenceDiagram\nA->>B: hi\n",
		"class":     "classDiagram\nclass A\nclass B\nA <|-- B\n",
		"er":        "erDiagram\nA ||--o{ B : has\n",
		"state":     "stateDiagram-v2\n[*] --> S\nS --> [*]\n",
	}
	for kind, src := range cases {
		t.Run(kind, func(t *testing.T) {
			dl, err := ParseAndLayout(src, LayoutOptions{})
			if err != nil { t.Fatalf("ParseAndLayout: %v", err) }
			if dl == nil || len(dl.Items) == 0 {
				t.Fatal("empty DisplayList")
			}
		})
	}

	if _, err := ParseAndLayout("garbage", LayoutOptions{}); !errors.Is(err, ErrUnknownDiagram) {
		t.Fatalf("unknown: expected ErrUnknownDiagram, got %v", err)
	}
	_ = displaylist.RoleNode
}
```

- [ ] `go test ./...` clean. `go vet ./...` clean. Commit: `git commit -m "mermaid: phase 8 cross-diagram smoke test"`

---

## Self-Review

| Spec | Phase 8 task |
|---|---|
| canvasr.RenderPNG / RenderInto | Task 1 |
| Style/RoleStyle parity with pdf | Task 1 |
| cmd/parse | Task 2 |
| cmd/render with PDF + PNG | Task 3 |
| All-diagram integration tests | Task 4, 6 |
| README docs | Task 5 |

## Open Questions / Risks

- **tdewolff/canvas API drift**: the rasterizer API has shifted across recent versions (`canvas.Renderer`, `renderers.PNG`, `c.WriteFile` — different shapes). Pin a known-good version in `go.mod` (the same version go-tex uses) and adapt code to that version's API.
- **PDF emitter output via canvasr**: tdewolff/canvas can also emit PDF. The `pdf` package emits via fpdf; `canvasr` emits via tdewolff. Keep them parallel — same role-driven Style, slightly different paint primitives. fpdf is the primary PDF path because it integrates with the platform's existing fpdf usage.
- **`DetectTypeForCmd` exposure**: don't make `detectType` public if we can avoid it. Have `cmd/parse` call `mermaid.ParseAndLayout` and inspect the returned types via reflection or an exposed `mermaid.DiagramType()` helper that returns a string.
