# go-mermaid

Pure-Go [Mermaid](https://mermaid.js.org) diagram parser, layout engine,
and renderer. Produces a style-neutral `DisplayList` IR that downstream
emitters draw into PDFs (via [fpdf](https://codeberg.org/go-pdf/fpdf))
or rasterize via [tdewolff/canvas](https://github.com/tdewolff/canvas)
(PNG / SVG).

Built as the Mermaid counterpart to
[`go-tex`](https://github.com/luo-studio/go-tex). Layout uses
[nulab/autog](https://github.com/nulab/autog) — no Graphviz, no wasm,
no CGO.

## Status

All five Mermaid diagram types are supported at mermaigo parity:

| Type | Source keyword | Layout | Notes |
|------|----------------|--------|-------|
| Flowchart | `flowchart`, `graph` | autog + recursive subgraph clusters | All shapes, all edge styles, classDef, subgraphs |
| Sequence | `sequenceDiagram` | hand-rolled column/row | Messages, notes, all block types, activations |
| Class | `classDiagram` | autog (+ namespaces as clusters) | All relationship variants, cardinality, annotations |
| ER | `erDiagram` | autog | All cardinality combos, PK/FK/UK markers |
| State | `stateDiagram` / `-v2` | autog (+ composite states as clusters) | Pseudostates, composite states, transitions |

PDF and PNG/SVG output via fpdf and tdewolff/canvas, with role-keyed
styling so callers can override colors, fonts, and line widths
without forking the library.

## Usage

### Library

```go
import (
    "codeberg.org/go-pdf/fpdf"
    mermaidpdf "github.com/luo-studio/go-mermaid/pdf"
)

pdf := fpdf.New("P", "mm", "A4", "")
pdf.SetFont("Helvetica", "", 10)
pdf.AddPage()

src := `flowchart TB
A[Start] --> B{Decide}
B -- yes --> C[Do]
B -- no --> D[Skip]`

err := mermaidpdf.DrawMermaid(pdf, src, 10, 10, mermaidpdf.EmbedDefaults())
```

For PNG output:

```go
import mermaidcanvasr "github.com/luo-studio/go-mermaid/canvasr"

png, err := mermaidcanvasr.RenderPNG(src, mermaidcanvasr.RenderOptions{})
```

For low-level access (parse + layout, then inspect the DisplayList):

```go
import mermaid "github.com/luo-studio/go-mermaid"

dl, err := mermaid.ParseAndLayout(src, mermaid.LayoutOptions{})
// dl.Items is a []displaylist.Item — Shape, Edge, Text, Cluster, Marker.
```

### CLI

```bash
# Parse to JSON AST (debugging)
echo 'flowchart TB
A --> B' | go run ./cmd/parse

# Render
echo 'flowchart TB
A --> B' | go run ./cmd/render -format pdf > diagram.pdf
echo 'flowchart TB
A --> B' | go run ./cmd/render -format png > diagram.png
echo 'flowchart TB
A --> B' | go run ./cmd/render -format svg > diagram.svg
```

## Architecture

See `docs/superpowers/specs/2026-05-08-go-mermaid-design.md` for the
full design. Briefly:

- **`displaylist/`** — style-neutral IR (`Shape`, `Edge`, `Text`,
  `Cluster`, `Marker`) with a closed set of standard `Role` constants
  for emitter styling lookups.
- **`layoutopts/`** — shared layout knobs (`Measurer`, spacings,
  padding) used by every per-diagram-type package.
- **`fonts/`** — embedded Inter Regular/Bold/Italic TTFs.
- **`fontmetrics/`** — default `Measurer` using sfnt-parsed Inter.
- **`autog/`** — thin adapter over `nulab/autog` plus a recursive
  cluster engine for subgraph nesting.
- **`flowchart/`, `sequence/`, `class/`, `er/`, `state/`** — one
  package per diagram type. Each owns its parser, AST, and layout.
- **`pdf/`** — `DrawMermaid` and `DrawInto` for fpdf documents.
- **`canvasr/`** — `RenderPNG` / `RenderSVG` via tdewolff/canvas.
- **`cmd/parse`, `cmd/render`** — thin CLI wrappers.

## License

go-mermaid: MIT (see `LICENSE`).
Inter font: SIL Open Font License 1.1 (see `fonts/LICENSE-Inter.txt`).
