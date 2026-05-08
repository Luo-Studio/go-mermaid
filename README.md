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
