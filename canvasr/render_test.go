package mermaidcanvasr

import (
	"bytes"
	"image/png"
	"testing"
)

func TestRenderPNGBasic(t *testing.T) {
	src := "flowchart TB\nA --> B\nB --> C\n"
	out, err := RenderPNG(src, RenderOptions{})
	if err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	if !bytes.HasPrefix(out, []byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatal("output is not a PNG")
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		t.Fatalf("PNG dimensions: %v", b)
	}
}

func TestRenderPNGAllDiagramTypes(t *testing.T) {
	cases := map[string]string{
		"flowchart": "flowchart TB\nA --> B\n",
		"sequence":  "sequenceDiagram\nparticipant A\nparticipant B\nA->>B: hi\n",
		"class":     "classDiagram\nclass A\nclass B\nA <|-- B\n",
		"er":        "erDiagram\nA ||--o{ B : has\n",
		"state":     "stateDiagram-v2\n[*] --> S\nS --> [*]\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := RenderPNG(src, RenderOptions{})
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if len(out) < 100 {
				t.Fatalf("%s: PNG too small (%d bytes)", name, len(out))
			}
		})
	}
}
