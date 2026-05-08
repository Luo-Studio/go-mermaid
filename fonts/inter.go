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
