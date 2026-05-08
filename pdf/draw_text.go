package mermaidpdf

import (
	"codeberg.org/go-pdf/fpdf"

	"github.com/luo-studio/go-mermaid/displaylist"
)

func drawText(pdf *fpdf.Fpdf, t displaylist.Text, tx func(displaylist.Point) (float64, float64), rs RoleStyle, emojiFont string, scale float64) {
	if len(t.Lines) == 0 {
		return
	}
	if rs.Font == "" {
		rs.Font = "Helvetica"
	}
	if rs.FontSize <= 0 {
		rs.FontSize = 10
	}
	if scale <= 0 {
		scale = 1
	}
	// Scale FontSize so labels shrink in lockstep with the diagram's
	// geometry — without this, page-fit shrinking leaves text at full
	// size while boxes shrink, and text overflows its box.
	rs.FontSize *= scale
	pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
	pdf.SetTextColor(int(rs.TextR), int(rs.TextG), int(rs.TextB))

	x, y := tx(t.Pos)
	// fpdf's text origin is the baseline. 1pt ≈ 0.353 mm, so a
	// FontSize-pt line takes ~0.353 mm at 1pt = 0.353 mm. We use 0.36
	// for tight per-line spacing in multi-line labels — the previous
	// 0.4 gave noticeable extra gap between lines.
	lineH := rs.FontSize * 0.36
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

	emojiAvailable := emojiFont != ""

	// Edge labels (and similar overlay text on top of lines) get a
	// white fill behind them so an arrow passing through doesn't make
	// the text unreadable. The fill colour is the page background;
	// for now hardcode white — themes that need a tinted background
	// can override per-Role styling later.
	overlayBackground := t.Role == displaylist.RoleEdgeLabel || t.Role == displaylist.RoleMessageLabel

	// measureRuns returns the total rendered width of runs by
	// switching fonts before each GetStringWidth call.
	measureRuns := func(runs []textRun) float64 {
		var w float64
		for _, r := range runs {
			if r.emoji {
				if !emojiAvailable {
					continue
				}
				pdf.SetFont(emojiFont, "", rs.FontSize)
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
		// Cell positions text by its top-left corner; baselineY was
		// computed for pdf.Text (baseline-positioned). Convert back
		// to top-left by subtracting the ascent portion (≈0.7*lineH).
		topY := startY + float64(i)*lineH - lineH*0.7

		if overlayBackground {
			// Paint a white rect behind the line to mask any edge or
			// arrowhead passing through. Slight horizontal padding so
			// the mask covers ascenders/descenders without touching
			// the next character.
			savedFillR, savedFillG, savedFillB := pdf.GetFillColor()
			pdf.SetFillColor(255, 255, 255)
			pdf.Rect(lx-0.5, topY, w+1.0, lineH, "F")
			pdf.SetFillColor(savedFillR, savedFillG, savedFillB)
		}

		// Save the cell margin and zero it so text runs sit flush —
		// fpdf's default cell margin would add an asymmetric pad.
		savedMargin := pdf.GetCellMargin()
		pdf.SetCellMargin(0)
		for _, r := range runs {
			if r.emoji {
				pdf.SetFont(emojiFont, "", rs.FontSize)
			} else {
				pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
			}
			runW := pdf.GetStringWidth(r.text)
			pdf.SetXY(curX, topY)
			// Cell draws the text in a (w, h) box. Bitmap-based emoji
			// fonts (NotoColorEmoji) embed glyphs as images via Cell;
			// pdf.Text doesn't trigger that path.
			pdf.Cell(runW, lineH, r.text)
			curX += runW
		}
		pdf.SetCellMargin(savedMargin)
		// Leave the body font active for subsequent draws.
		pdf.SetFont(rs.Font, rs.FontStyle, rs.FontSize)
	}
}
