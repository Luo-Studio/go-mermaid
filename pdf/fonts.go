package mermaidpdf

import (
	"sync"

	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/fonts"
)

// FontFamily is the family name registered with fpdf for the embedded
// Inter face. Default styles in DefaultStyle()/StyleFromTheme() use it.
const FontFamily = "go-mermaid-inter"

// regOnce gates Inter registration to once per fpdf document so
// AddUTF8FontFromBytes only runs the first time it sees a given doc.
var (
	regMu   sync.Mutex
	regDone = map[*fpdf.Fpdf]bool{}
)

// ensureInterFont registers the embedded Inter Regular/Bold/Italic
// TTFs with pdf as a UTF-8 font family so non-Latin-1 characters
// (emoji, CJK, etc.) render correctly. Idempotent per *Fpdf — fpdf
// itself rejects duplicate registrations, but we gate as well so we
// don't load the TTF bytes more than once per doc.
//
// Errors during registration are returned. They typically only
// surface if the embedded TTF bytes are malformed (a build-time bug).
func ensureInterFont(pdf *fpdf.Fpdf) error {
	regMu.Lock()
	defer regMu.Unlock()
	if regDone[pdf] {
		return nil
	}
	for _, v := range []struct {
		style fonts.Style
		key   string
	}{
		{fonts.StyleRegular, ""},
		{fonts.StyleBold, "B"},
		{fonts.StyleItalic, "I"},
	} {
		data, err := fonts.Bytes(v.style)
		if err != nil {
			return err
		}
		pdf.AddUTF8FontFromBytes(FontFamily, v.key, data)
		if pdf.Err() {
			return pdf.Error()
		}
	}
	regDone[pdf] = true
	return nil
}
