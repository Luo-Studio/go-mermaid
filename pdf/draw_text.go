package mermaidpdf

import (
	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func drawText(pdf *fpdf.Fpdf, t displaylist.Text, tx func(displaylist.Point) (float64, float64), rs RoleStyle) {
	if len(t.Lines) == 0 {
		return
	}
	if rs.Font == "" {
		rs.Font = "Helvetica"
	}
	if rs.FontSize <= 0 {
		rs.FontSize = 10
	}
	pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
	pdf.SetTextColor(int(rs.TextR), int(rs.TextG), int(rs.TextB))

	x, y := tx(t.Pos)
	// fpdf's text origin is the baseline. 1pt ≈ 0.353 mm, so a
	// FontSize-pt line takes ~0.4 * FontSize mm.
	lineH := rs.FontSize * 0.4
	totalH := lineH * float64(len(t.Lines))

	var startY float64
	switch t.VAlign {
	case displaylist.VAlignTop:
		startY = y + lineH*0.7
	case displaylist.VAlignBottom:
		startY = y - totalH + lineH*0.7
	default:
		startY = y - totalH/2 + lineH*0.7
	}

	emojiAvailable := ensureEmojiFont(pdf)

	// measureRuns returns the total rendered width of runs by
	// switching fonts before each GetStringWidth call.
	measureRuns := func(runs []textRun) float64 {
		var w float64
		for _, r := range runs {
			if r.emoji {
				if !emojiAvailable {
					continue
				}
				pdf.SetFont(EmojiFontFamily, "", rs.FontSize)
			} else {
				pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
			}
			w += pdf.GetStringWidth(r.text)
		}
		// Restore the body font; caller relies on this being the
		// active font after drawText returns.
		pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
		return w
	}

	for i, line := range t.Lines {
		runs := splitEmojiRuns(line)
		if !emojiAvailable {
			// Drop emoji from the line so fpdf doesn't render its
			// "NO GLYPH" placeholder box.
			filtered := runs[:0]
			for _, r := range runs {
				if r.emoji {
					continue
				}
				r.text = stripUnsupportedGlyphs(r.text)
				if r.text != "" {
					filtered = append(filtered, r)
				}
			}
			runs = filtered
		}
		if len(runs) == 0 {
			continue
		}
		w := measureRuns(runs)
		var lx float64
		switch t.Align {
		case displaylist.AlignLeft:
			lx = x
		case displaylist.AlignRight:
			lx = x - w
		default:
			lx = x - w/2
		}
		curX := lx
		baselineY := startY + float64(i)*lineH
		for _, r := range runs {
			if r.emoji {
				pdf.SetFont(EmojiFontFamily, "", rs.FontSize)
			} else {
				pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
			}
			pdf.Text(curX, baselineY, r.text)
			curX += pdf.GetStringWidth(r.text)
		}
		// Leave the body font active for subsequent draws.
		pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
	}
}
