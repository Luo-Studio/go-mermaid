// Command render reads a Mermaid diagram from stdin and writes a
// rendered PDF, PNG, or SVG to stdout.
//
// Usage:
//   render -format pdf < diagram.mmd > diagram.pdf
//   render -format png < diagram.mmd > diagram.png
//   render -format svg < diagram.mmd > diagram.svg
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"codeberg.org/go-pdf/fpdf"

	mermaidcanvasr "github.com/luo-studio/go-mermaid/canvasr"
	mermaidpdf "github.com/luo-studio/go-mermaid/pdf"
)

func main() {
	format := flag.String("format", "pdf", "output format: pdf|png|svg")
	flag.Parse()

	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(err)
	}

	switch *format {
	case "pdf":
		pdf := fpdf.New("P", "mm", "A4", "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.AddPage()
		opts := mermaidpdf.EmbedDefaults()
		if err := mermaidpdf.DrawMermaid(pdf, string(src), 10, 10, opts); err != nil {
			fail(err)
		}
		if pdf.Err() {
			fail(pdf.Error())
		}
		if err := pdf.Output(os.Stdout); err != nil {
			fail(err)
		}
	case "png":
		out, err := mermaidcanvasr.RenderPNG(string(src), mermaidcanvasr.RenderOptions{})
		if err != nil {
			fail(err)
		}
		if _, err := io.Copy(os.Stdout, bytes.NewReader(out)); err != nil {
			fail(err)
		}
	case "svg":
		out, err := mermaidcanvasr.RenderSVG(string(src), mermaidcanvasr.RenderOptions{})
		if err != nil {
			fail(err)
		}
		if _, err := io.Copy(os.Stdout, bytes.NewReader(out)); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("unknown format %q (want pdf|png|svg)", *format))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
