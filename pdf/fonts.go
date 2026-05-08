package mermaidpdf

import (
	"os"
	"strings"
	"sync"
	"unicode"

	"codeberg.org/go-pdf/fpdf"
	"golang.org/x/image/font/sfnt"

	"github.com/luo-studio/go-mermaid/fonts"
)

// EmojiFontFamily is the family name registered with fpdf for the
// system NotoColorEmoji font when found at one of the standard
// install paths. Empty in the registry if no emoji font was located
// on this host — drawText then strips emoji from labels rather than
// rendering "NO GLYPH" tofu boxes.
const EmojiFontFamily = "go-mermaid-emoji"

// notoEmojiPaths is searched at first DrawInto for an emoji TTF.
// Mirrors the platform's PDF lib (lib/go/executor/libs/pdf/render.go).
var notoEmojiPaths = []string{
	"/usr/share/fonts/noto/NotoColorEmoji.ttf",                // Alpine
	"/usr/share/fonts/truetype/noto/NotoColorEmoji.ttf",       // Debian/Ubuntu
	"/usr/share/fonts/google-noto-emoji/NotoColorEmoji.ttf",   // Fedora
	"/Library/Fonts/Apple Color Emoji.ttc",                    // macOS
	"/System/Library/Fonts/Apple Color Emoji.ttc",             // macOS (system)
	"/tmp/fonts/NotoColorEmoji.ttf",                           // local testing
}

// emojiRegDone tracks per-fpdf-doc whether we've attempted to load
// the emoji font. The boolean value is true iff the font was loaded
// successfully (and is therefore usable in drawText).
var (
	emojiRegMu   sync.Mutex
	emojiRegDone = map[*fpdf.Fpdf]bool{}
)

// interSfnt holds the parsed Inter Regular face used to test glyph
// presence. Initialized on first call to ensureInterFont so we don't
// pay the parse cost when only canvasr is in use.
var (
	interSfntOnce sync.Once
	interSfnt     *sfnt.Font
)

// inInterFont reports whether the embedded Inter Regular face has a
// glyph for r. Callers use this to filter codepoints that would
// otherwise render as fpdf's "NO GLYPH" placeholder box.
func inInterFont(r rune) bool {
	loadInterSfnt()
	if interSfnt == nil {
		return true // be permissive if we couldn't parse the font
	}
	var b sfnt.Buffer
	idx, err := interSfnt.GlyphIndex(&b, r)
	return err == nil && idx != 0
}

func loadInterSfnt() {
	interSfntOnce.Do(func() {
		data, err := fonts.Bytes(fonts.StyleRegular)
		if err != nil {
			return
		}
		f, err := sfnt.Parse(data)
		if err != nil {
			return
		}
		interSfnt = f
	})
}

// ensureEmojiFont registers a system color-emoji TTF (NotoColorEmoji
// on Linux, Apple Color Emoji on macOS) if one can be found at a
// well-known path. Returns true iff the font is now registered as
// EmojiFontFamily on pdf. Idempotent per *Fpdf.
func ensureEmojiFont(pdf *fpdf.Fpdf) bool {
	emojiRegMu.Lock()
	defer emojiRegMu.Unlock()
	if loaded, ok := emojiRegDone[pdf]; ok {
		return loaded
	}
	for _, path := range notoEmojiPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// fpdf only supports plain TrueType, not TTC collections —
		// the upstream metrics extractor returns "not supported" for
		// .ttc files (e.g. macOS's Apple Color Emoji.ttc). Validate
		// with sfnt.Parse first so we register only fonts fpdf can
		// actually render with, then fall back to stripping if no
		// usable emoji font exists.
		if _, err := sfnt.Parse(data); err != nil {
			continue
		}
		pdf.AddUTF8FontFromBytes(EmojiFontFamily, "", data)
		if pdf.Err() {
			pdf.ClearError()
			continue
		}
		emojiRegDone[pdf] = true
		return true
	}
	emojiRegDone[pdf] = false
	return false
}

// isEmoji reports whether r is in a Unicode block typically rendered
// by an emoji/symbol font rather than a standard text font. Mirrors
// the same heuristic the platform's PDF lib uses (kept in sync with
// services/platform/.../pdf/render.go isEmoji).
func isEmoji(r rune) bool {
	switch {
	case r >= 0x2600 && r <= 0x27BF:
		return true
	case r >= 0x2B50 && r <= 0x2B55:
		return true
	case r >= 0x2934 && r <= 0x2935:
		return true
	case r >= 0x3297 && r <= 0x3299:
		return true
	case r >= 0xFE00 && r <= 0xFE0F:
		return true
	case r >= 0x1F000 && r <= 0x1FAFF:
		return true
	case r == 0x200D:
		return true
	case r == 0x20E3:
		return true
	case r >= 0x2190 && r <= 0x21FF:
		return !unicode.IsLetter(r)
	case r >= 0x2300 && r <= 0x23FF:
		return true
	case r >= 0x25A0 && r <= 0x25FF:
		return true
	case r >= 0x2700 && r <= 0x27BF:
		return true
	case r >= 0x2900 && r <= 0x297F:
		return true
	case r == 0x00A9 || r == 0x00AE:
		return true
	case r == 0x2122:
		return true
	}
	return false
}

// isZWJ reports whether r is a zero-width joiner or variation
// selector that links emoji into a sequence (e.g. 👨‍👩‍👧).
func isZWJ(r rune) bool {
	return r == 0x200D || (r >= 0xFE00 && r <= 0xFE0F)
}

// textRun is one segment of a label that uses a single font face.
type textRun struct {
	text  string
	emoji bool
}

// splitEmojiRuns splits s into alternating runs of body text and
// emoji. ZWJ sequences stay together as a single emoji run.
func splitEmojiRuns(s string) []textRun {
	if s == "" {
		return nil
	}
	var out []textRun
	var cur strings.Builder
	curEmoji := false
	first := true
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, textRun{text: cur.String(), emoji: curEmoji})
		cur.Reset()
	}
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if isZWJ(r) && curEmoji {
			cur.WriteRune(r)
			continue
		}
		e := isEmoji(r)
		if first {
			curEmoji = e
			first = false
		}
		if e != curEmoji {
			flush()
			curEmoji = e
		}
		cur.WriteRune(r)
	}
	flush()
	return out
}

// stripUnsupportedGlyphs removes runes that Inter doesn't have so
// fpdf's UTF-8 renderer doesn't draw the "NO GLYPH" placeholder box.
// ASCII passes through unconditionally.
func stripUnsupportedGlyphs(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x80 {
			b.WriteRune(r)
			continue
		}
		if inInterFont(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

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
