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

	for i, line := range t.Lines {
		w := pdf.GetStringWidth(line)
		var lx float64
		switch t.Align {
		case displaylist.AlignLeft:
			lx = x
		case displaylist.AlignRight:
			lx = x - w
		default:
			lx = x - w/2
		}
		pdf.Text(lx, startY+float64(i)*lineH, line)
	}
}
