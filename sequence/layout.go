package sequence

import (
	"github.com/luo-studio/go-mermaid/displaylist"
	"github.com/luo-studio/go-mermaid/layoutopts"
)

// Layout positions a sequence diagram and returns a DisplayList.
// Hand-rolled column/row layout — no autog.
func Layout(d *Diagram, opts layoutopts.Options) *displaylist.DisplayList {
	if d == nil || len(d.Actors) == 0 {
		return &displaylist.DisplayList{}
	}
	measurer := opts.ResolveMeasurer()
	dl := &displaylist.DisplayList{}

	const (
		actorBoxH      = 30.0
		actorPadH      = 16.0
		messagePadY    = 24.0
		messagePadTop  = 50.0
		notePadX       = 8.0
		notePadY       = 6.0
		blockHeader    = 20.0
		blockPadY      = 10.0
		blockPadX      = 6.0
		activationW    = 8.0
		activationStep = 4.0
	)

	actorSpacing := opts.SequenceActorSpacing
	if actorSpacing <= 0 {
		actorSpacing = 30
	}
	messageSpacing := opts.SequenceMessageSpacing
	if messageSpacing <= 0 {
		messageSpacing = messagePadY
	}

	// Compute actor column centres and box widths.
	type actorPos struct {
		id     string
		label  string
		kind   ActorKind
		boxW   float64
		boxX   float64
		boxY   float64
		centre float64
	}
	actors := make([]actorPos, len(d.Actors))
	cursorX := 0.0
	for i, a := range d.Actors {
		labelW, _ := measurer.Measure(a.Label, displaylist.RoleActorTitle)
		boxW := labelW + actorPadH*2
		actors[i] = actorPos{
			id:     a.ID,
			label:  a.Label,
			kind:   a.Kind,
			boxW:   boxW,
			boxX:   cursorX,
			centre: cursorX + boxW/2,
			boxY:   0,
		}
		cursorX += boxW + actorSpacing
	}
	totalWidth := cursorX - actorSpacing
	if totalWidth < 0 {
		totalWidth = 0
	}
	headerHeight := actorBoxH

	idx := map[string]int{}
	for i, a := range actors {
		idx[a.id] = i
	}

	// Activation tracking. Each actor has a stack of open activations,
	// each remembered by its starting Y. The activation rect is emitted
	// when the activation closes (deactivate, or end of diagram).
	type openAct struct {
		actor int
		startY float64
		level  int
	}
	openActs := map[int][]openAct{}

	closeActivation := func(actorIdx int, atY float64) (openAct, bool) {
		stack := openActs[actorIdx]
		if len(stack) == 0 {
			return openAct{}, false
		}
		top := stack[len(stack)-1]
		openActs[actorIdx] = stack[:len(stack)-1]
		// Emit the rect.
		x := actors[actorIdx].centre - activationW/2 + float64(top.level)*activationStep
		dl.Items = append(dl.Items, displaylist.Shape{
			Kind: displaylist.ShapeKindRect,
			BBox: displaylist.Rect{X: x, Y: top.startY, W: activationW, H: atY - top.startY},
			Role: displaylist.RoleActivation,
		})
		_ = top
		return top, true
	}
	openActivation := func(actorIdx int, atY float64) {
		level := len(openActs[actorIdx])
		openActs[actorIdx] = append(openActs[actorIdx], openAct{actor: actorIdx, startY: atY, level: level})
	}

	// Walk items recursively. Each item advances cursorY by its height.
	cursorY := messagePadTop + headerHeight

	var walk func(items []Item)

	emitMessage := func(m *Message) {
		from, fok := idx[m.From]
		to, tok := idx[m.To]
		if !fok || !tok {
			return
		}
		// Activation behavior:
		//   `+`: open activation on target at this message's Y
		//   `-`: close activation on source after this message
		// Standard mermaid puts the activation start at the same Y as
		// the message's arrival.
		// Allocate label height first.
		labelH := 0.0
		labelW := 0.0
		if m.Label != "" {
			labelW, labelH = measurer.Measure(m.Label, displaylist.RoleMessageLabel)
		}
		topGap := labelH + 4
		if topGap < messageSpacing*0.5 {
			topGap = messageSpacing * 0.5
		}
		cursorY += topGap

		x0 := actors[from].centre
		x1 := actors[to].centre
		// Adjust message endpoints for any active activations on each side.
		if lvl := len(openActs[from]); lvl > 0 {
			off := float64(lvl-1)*activationStep + activationW/2
			if x0 < x1 {
				x0 += off
			} else {
				x0 -= off
			}
		}
		if lvl := len(openActs[to]); lvl > 0 {
			off := float64(lvl-1)*activationStep + activationW/2
			if x0 < x1 {
				x1 -= off
			} else {
				x1 += off
			}
		}

		points := []displaylist.Point{
			{X: x0, Y: cursorY},
			{X: x1, Y: cursorY},
		}
		dl.Items = append(dl.Items, displaylist.Edge{
			Points:     points,
			LineStyle:  arrowLineStyle(m.Arrow),
			ArrowEnd:   arrowEnd(m.Arrow),
			Role:       displaylist.RoleEdge,
		})
		if m.Label != "" {
			midX := (x0 + x1) / 2
			dl.Items = append(dl.Items, displaylist.Text{
				Pos:    displaylist.Point{X: midX, Y: cursorY - 3},
				Lines:  []string{m.Label},
				Align:  displaylist.AlignCenter,
				VAlign: displaylist.VAlignBottom,
				Role:   displaylist.RoleMessageLabel,
			})
			if labelW > 0 {
				_ = labelW
			}
		}
		// Activations.
		if m.Activate {
			openActivation(to, cursorY)
		}
		if m.Deactivate {
			closeActivation(from, cursorY)
		}
		cursorY += messageSpacing * 0.5
	}

	emitNote := func(n *Note) {
		text := n.Text
		tw, th := measurer.Measure(text, displaylist.RoleNoteText)
		w := tw + notePadX*2
		h := th + notePadY*2
		var x float64
		switch n.Side {
		case NoteLeftOf:
			if len(n.Actors) == 0 {
				return
			}
			ai, ok := idx[n.Actors[0]]
			if !ok {
				return
			}
			x = actors[ai].boxX - w - 8
			if x < 0 {
				x = 4
			}
		case NoteRightOf:
			if len(n.Actors) == 0 {
				return
			}
			ai, ok := idx[n.Actors[0]]
			if !ok {
				return
			}
			x = actors[ai].boxX + actors[ai].boxW + 8
		case NoteOver:
			if len(n.Actors) == 0 {
				return
			}
			minX, maxX := float64(1e18), -1.0
			for _, id := range n.Actors {
				ai, ok := idx[id]
				if !ok {
					continue
				}
				if actors[ai].boxX < minX {
					minX = actors[ai].boxX
				}
				if r := actors[ai].boxX + actors[ai].boxW; r > maxX {
					maxX = r
				}
			}
			if maxX > minX {
				x = minX - notePadX
				w = maxX - minX + notePadX*2
			} else {
				if len(n.Actors) > 0 {
					ai := idx[n.Actors[0]]
					x = actors[ai].centre - w/2
				}
			}
		}
		cursorY += 6
		dl.Items = append(dl.Items, displaylist.Cluster{
			BBox:  displaylist.Rect{X: x, Y: cursorY, W: w, H: h},
			Title: "",
			Role:  displaylist.RoleSequenceNote,
		})
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: x + w/2, Y: cursorY + h/2},
			Lines:  []string{text},
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignMiddle,
			Role:   displaylist.RoleNoteText,
		})
		cursorY += h + 4
	}

	emitBlock := func(b *Block) {
		// Block frame spans from cursorY to a Y after recursing into
		// children (and Else branches).
		startY := cursorY
		cursorY += blockHeader
		// Main branch.
		walk(b.Children)
		// Else / and / option branches: divider edge then recurse.
		for _, branch := range b.Else {
			cursorY += 4
			// Divider line will be drawn after we know the X span; for
			// now just record the Y and the label position.
			dividerY := cursorY
			labelText := branch.Label
			labelRole := displaylist.RoleEdgeLabel
			dl.Items = append(dl.Items, displaylist.Edge{
				Points:    []displaylist.Point{{X: 0, Y: dividerY}, {X: totalWidth, Y: dividerY}},
				LineStyle: displaylist.LineStyleDashed,
				Role:      displaylist.RoleEdge,
			})
			if labelText != "" {
				dl.Items = append(dl.Items, displaylist.Text{
					Pos:    displaylist.Point{X: 8, Y: dividerY + 8},
					Lines:  []string{labelText},
					Align:  displaylist.AlignLeft,
					VAlign: displaylist.VAlignTop,
					Role:   labelRole,
				})
			}
			cursorY += 12
			walk(branch.Children)
		}
		endY := cursorY + 8
		role := blockRole(b.Type)
		title := blockKindLabel(b.Type)
		if b.Label != "" {
			title = title + ": " + b.Label
		}
		dl.Items = append(dl.Items, displaylist.Cluster{
			BBox:  displaylist.Rect{X: -blockPadX, Y: startY, W: totalWidth + blockPadX*2, H: endY - startY},
			Title: title,
			Role:  role,
		})
		cursorY = endY + 4
	}

	walk = func(items []Item) {
		for _, it := range items {
			switch v := it.(type) {
			case *Message:
				emitMessage(v)
			case *Note:
				emitNote(v)
			case *Block:
				emitBlock(v)
			case *Activate:
				if ai, ok := idx[v.Actor]; ok {
					openActivation(ai, cursorY)
				}
			case *Deactivate:
				if ai, ok := idx[v.Actor]; ok {
					closeActivation(ai, cursorY)
				}
			}
		}
	}

	walk(d.Items)

	// Close any remaining activations at the diagram's bottom.
	for ai := range actors {
		for len(openActs[ai]) > 0 {
			closeActivation(ai, cursorY)
		}
	}

	footerY := cursorY + 8

	// Emit lifelines (dashed line per actor from header bottom to footer top).
	for i := range actors {
		dl.Items = append(dl.Items, displaylist.Edge{
			Points: []displaylist.Point{
				{X: actors[i].centre, Y: actors[i].boxY + actorBoxH},
				{X: actors[i].centre, Y: footerY},
			},
			LineStyle: displaylist.LineStyleDashed,
			Role:      displaylist.RoleLifeline,
		})
	}

	// Emit actor headers (top row) and footers (mirror at bottom).
	for i, a := range actors {
		dl.Items = append(dl.Items, displaylist.Shape{
			Kind: displaylist.ShapeKindRect,
			BBox: displaylist.Rect{X: a.boxX, Y: a.boxY, W: a.boxW, H: actorBoxH},
			Role: displaylist.RoleActorBox,
		})
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: a.centre, Y: a.boxY + actorBoxH/2},
			Lines:  []string{a.label},
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignMiddle,
			Role:   displaylist.RoleActorTitle,
		})
		// Footer mirror.
		dl.Items = append(dl.Items, displaylist.Shape{
			Kind: displaylist.ShapeKindRect,
			BBox: displaylist.Rect{X: a.boxX, Y: footerY, W: a.boxW, H: actorBoxH},
			Role: displaylist.RoleActorBox,
		})
		dl.Items = append(dl.Items, displaylist.Text{
			Pos:    displaylist.Point{X: a.centre, Y: footerY + actorBoxH/2},
			Lines:  []string{a.label},
			Align:  displaylist.AlignCenter,
			VAlign: displaylist.VAlignMiddle,
			Role:   displaylist.RoleActorTitle,
		})
		_ = i
	}

	dl.Width = totalWidth
	dl.Height = footerY + actorBoxH
	return dl
}

func arrowLineStyle(a ArrowKind) displaylist.LineStyle {
	switch a {
	case ArrowReply, ArrowOpenDashed, ArrowDashed, ArrowDashedCross:
		return displaylist.LineStyleDashed
	default:
		return displaylist.LineStyleSolid
	}
}

func arrowEnd(a ArrowKind) displaylist.MarkerKind {
	switch a {
	case ArrowSync, ArrowReply:
		return displaylist.MarkerArrow
	case ArrowOpen, ArrowOpenDashed:
		return displaylist.MarkerArrowOpen
	case ArrowCross, ArrowDashedCross:
		return displaylist.MarkerCross
	case ArrowSolid, ArrowDashed:
		return displaylist.MarkerArrowOpen
	}
	return displaylist.MarkerArrow
}

func blockRole(t BlockType) displaylist.Role {
	switch t {
	case BlockLoop:
		return displaylist.RoleLoopBlock
	case BlockAlt:
		return displaylist.RoleAltBlock
	case BlockOpt:
		return displaylist.RoleOptBlock
	case BlockPar:
		return displaylist.RoleParBlock
	case BlockCritical:
		return displaylist.RoleCriticalBlock
	case BlockBreak:
		return displaylist.RoleBreakBlock
	case BlockRect:
		return displaylist.RoleRectBlock
	}
	return displaylist.RoleLoopBlock
}

func blockKindLabel(t BlockType) string {
	switch t {
	case BlockLoop:
		return "loop"
	case BlockAlt:
		return "alt"
	case BlockOpt:
		return "opt"
	case BlockPar:
		return "par"
	case BlockCritical:
		return "critical"
	case BlockBreak:
		return "break"
	case BlockRect:
		return "rect"
	}
	return ""
}
