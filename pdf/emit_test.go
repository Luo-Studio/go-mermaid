package mermaidpdf

import (
	"bytes"
	"testing"

	"codeberg.org/go-pdf/fpdf"
)

func TestDrawMermaidSimpleFlowchart(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)

	src := "flowchart TB\nA --> B\nB --> C\n"
	if err := DrawMermaid(pdf, src, 10, 30, EmbedDefaults()); err != nil {
		t.Fatalf("DrawMermaid: %v", err)
	}
	if pdf.Err() {
		t.Fatalf("fpdf error: %v", pdf.Error())
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("Output: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
	if buf.Len() < 1000 {
		t.Fatalf("PDF unexpectedly small: %d bytes", buf.Len())
	}
}

func TestDrawMermaidUnknownDiagram(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	if err := DrawMermaid(pdf, "garbage\n", 10, 10, EmbedDefaults()); err == nil {
		t.Fatal("expected error for non-diagram input")
	}
}

func TestDrawMermaidAllShapes(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)

	src := `flowchart TB
A[Rect]
B(Round)
C([Stadium])
D{Diamond}
E((Circle))
F[(Cylinder)]
G{{Hex}}
A --> B
B --> C
C --> D
D --> E
E --> F
F --> G
`
	opts := EmbedDefaults()
	opts.MaxWidth = 180
	if err := DrawMermaid(pdf, src, 10, 30, opts); err != nil {
		t.Fatalf("DrawMermaid: %v", err)
	}
	if pdf.Err() {
		t.Fatalf("fpdf error: %v", pdf.Error())
	}
}
