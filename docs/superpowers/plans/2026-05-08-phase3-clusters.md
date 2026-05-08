# go-mermaid Phase 3 — Subgraph Cluster Recursion

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development.

**Goal:** Add recursive cluster (subgraph) layout to the flowchart pipeline. After this phase, nested `subgraph X ... end` blocks render as titled backdrop rectangles enclosing their member nodes.

**Architecture:** A new `autog/cluster.go` lays out clusters bottom-up: leaf clusters first (their members fed to autog as a flat graph), then each parent cluster treats its children as super-nodes sized to the child layout's bbox. After the top-level pass, child layouts are translated into their parent's coordinate space. Cross-cluster edges are routed at the lowest common ancestor's level. The flowchart layout consumes this and emits `displaylist.Cluster` items per subgraph alongside the existing Shape/Edge/Text items.

**Tech Stack:** Phase 1 + Phase 2 packages, no new external deps.

**Depends on:** Phase 2 plan complete (flowchart parser + layout + PDF emitter shipped).

---

## Spec Reference

Spec section "Cluster (Subgraph) Layout".

## File Structure

```
go-mermaid/
├── autog/
│   ├── cluster.go                   # ClusterInput, ClusterLayout, recursive engine
│   └── cluster_test.go
├── flowchart/
│   ├── layout.go                    # MODIFIED: build ClusterInput, consume ClusterLayout
│   └── layout_cluster_test.go
└── testdata/flowchart/clusters/
    ├── single-subgraph.mmd / .golden.json
    ├── nested-subgraphs.mmd / .golden.json
    └── cross-cluster-edges.mmd / .golden.json
```

## Tasks

### Task 1: ClusterInput / ClusterLayout types

**Files:** Create `autog/cluster.go` (types only — engine in Task 3).

- [ ] **Step 1.1: Implement types**

```go
package autog

import "github.com/luo-studio/go-mermaid/displaylist"

// ClusterInput describes a graph with nested clusters. Clusters
// hold node IDs and child clusters; leaf clusters have no children.
type ClusterInput struct {
	Direction    Direction
	NodeSpacing  float64
	LayerSpacing float64
	Padding      float64

	// All nodes (flat) referenced anywhere in the graph.
	Nodes []Node

	// Edges connect nodes by ID. Edges may cross cluster boundaries.
	Edges []Edge

	// Root clusters at the top level. Each may have nested children.
	Clusters []*Cluster
}

// Cluster is a nestable group of nodes. ID is the cluster identifier
// from the source (`subgraph X`); Title is the human label.
type Cluster struct {
	ID       string
	Title    string
	NodeIDs  []string
	Children []*Cluster
}

// ClusterLayout is the output. Nodes/Edges have absolute positions
// in DisplayList space; ClusterRects describes each cluster's
// backdrop rectangle (with the same nesting structure as input).
type ClusterLayout struct {
	Width, Height float64
	Nodes         []Node
	Edges         []Edge
	ClusterRects  []ClusterRect
}

// ClusterRect is a positioned cluster. Children mirror Cluster.Children.
type ClusterRect struct {
	ID       string
	Title    string
	BBox     displaylist.Rect
	Children []ClusterRect
}
```

- [ ] **Step 1.2: Commit** — `git commit -m "autog: ClusterInput/ClusterLayout types"`

---

### Task 2: Cluster layout engine — leaf clusters

Lay out each leaf cluster (no nested children) as an independent flat graph. Cache the resulting bbox + interior coordinates.

**Files:** Modify `autog/cluster.go`, create `autog/cluster_test.go`.

- [ ] **Step 2.1: Write the failing test**

```go
package autog

import "testing"

func TestClusterLayoutSingleLeafCluster(t *testing.T) {
	in := ClusterInput{
		Nodes: []Node{
			{ID: "A", Width: 30, Height: 20},
			{ID: "B", Width: 30, Height: 20},
		},
		Edges:    []Edge{{FromID: "A", ToID: "B"}},
		Clusters: []*Cluster{{ID: "G1", Title: "Group 1", NodeIDs: []string{"A", "B"}}},
	}
	out, err := LayoutClusters(in)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(out.ClusterRects) != 1 {
		t.Fatalf("expected 1 cluster rect, got %d", len(out.ClusterRects))
	}
	cr := out.ClusterRects[0]
	if cr.BBox.W <= 0 || cr.BBox.H <= 0 {
		t.Fatalf("cluster bbox empty: %+v", cr)
	}
	// Both nodes lie inside the cluster bbox.
	for _, n := range out.Nodes {
		if !cr.BBox.Contains(displaylist.Point{X: n.X + n.Width/2, Y: n.Y + n.Height/2}) {
			t.Errorf("node %s outside cluster bbox: node=%+v cluster=%+v", n.ID, n, cr)
		}
	}
}
```

- [ ] **Step 2.2: Run, expect FAIL** (LayoutClusters undefined).
- [ ] **Step 2.3: Implement leaf-only `LayoutClusters`**

```go
package autog

import (
	"fmt"

	"github.com/luo-studio/go-mermaid/displaylist"
)

// LayoutClusters runs the recursive cluster layout. Single-pass
// implementation: each cluster (bottom-up) is laid out as a flat
// graph; the result becomes a "super-node" at the parent level.
func LayoutClusters(in ClusterInput) (ClusterLayout, error) {
	// Validate: every node ID in NodeIDs must exist in Nodes.
	known := map[string]Node{}
	for _, n := range in.Nodes {
		known[n.ID] = n
	}
	// Compute the set of IDs that belong to any cluster (recursively).
	memberOf := map[string]string{} // node ID → cluster ID
	var collectMembers func(c *Cluster) error
	collectMembers = func(c *Cluster) error {
		for _, id := range c.NodeIDs {
			if _, ok := known[id]; !ok {
				return fmt.Errorf("autog: cluster %q references unknown node %q", c.ID, id)
			}
			if existing, exists := memberOf[id]; exists {
				return fmt.Errorf("autog: node %q is in multiple clusters (%s and %s)", id, existing, c.ID)
			}
			memberOf[id] = c.ID
		}
		for _, child := range c.Children {
			if err := collectMembers(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, c := range in.Clusters {
		if err := collectMembers(c); err != nil {
			return ClusterLayout{}, err
		}
	}

	// Phase 3 v1: assume *only leaf clusters* (no nesting). Multi-
	// level recursion lands in Task 4.
	for _, c := range in.Clusters {
		if len(c.Children) > 0 {
			return ClusterLayout{}, fmt.Errorf("autog: nested clusters not yet supported; cluster %q has %d children", c.ID, len(c.Children))
		}
	}

	clusterPad := 12.0

	// Lay out each leaf cluster's interior as a flat subgraph (only
	// edges among its members).
	type clusterResult struct {
		id     string
		title  string
		nodes  []Node // in cluster-local coordinates
		edges  []Edge
		width  float64
		height float64
	}
	var results []clusterResult
	for _, c := range in.Clusters {
		nodeSet := map[string]bool{}
		for _, id := range c.NodeIDs {
			nodeSet[id] = true
		}
		var clusterNodes []Node
		for _, id := range c.NodeIDs {
			clusterNodes = append(clusterNodes, known[id])
		}
		var clusterEdges []Edge
		for _, e := range in.Edges {
			if nodeSet[e.FromID] && nodeSet[e.ToID] {
				clusterEdges = append(clusterEdges, e)
			}
		}
		out, err := Layout(Input{
			Nodes:        clusterNodes,
			Edges:        clusterEdges,
			Direction:    in.Direction,
			NodeSpacing:  in.NodeSpacing,
			LayerSpacing: in.LayerSpacing,
		})
		if err != nil {
			return ClusterLayout{}, fmt.Errorf("autog: cluster %q interior: %w", c.ID, err)
		}
		results = append(results, clusterResult{
			id:     c.ID,
			title:  c.Title,
			nodes:  out.Nodes,
			edges:  out.Edges,
			width:  out.Width + clusterPad*2,
			height: out.Height + clusterPad*2 + 14, // title bar
		})
	}

	// Top-level: lay out the cluster super-nodes plus any non-clustered
	// nodes as a single flat graph. Edges between clusters/non-clustered
	// nodes use the cluster ID (or node ID for non-clustered).
	var topNodes []Node
	for _, r := range results {
		topNodes = append(topNodes, Node{ID: "__cluster__" + r.id, Width: r.width, Height: r.height})
	}
	for _, n := range in.Nodes {
		if _, inCluster := memberOf[n.ID]; !inCluster {
			topNodes = append(topNodes, n)
		}
	}
	// Top-level edges: collapse cluster-to-cluster edges, drop intra-cluster.
	var topEdges []Edge
	for _, e := range in.Edges {
		fromCluster := memberOf[e.FromID]
		toCluster := memberOf[e.ToID]
		if fromCluster != "" && toCluster != "" && fromCluster == toCluster {
			continue // intra-cluster, already handled inside cluster
		}
		from := e.FromID
		if fromCluster != "" {
			from = "__cluster__" + fromCluster
		}
		to := e.ToID
		if toCluster != "" {
			to = "__cluster__" + toCluster
		}
		topEdges = append(topEdges, Edge{FromID: from, ToID: to})
	}
	topOut, err := Layout(Input{
		Nodes:        topNodes,
		Edges:        topEdges,
		Direction:    in.Direction,
		NodeSpacing:  in.NodeSpacing,
		LayerSpacing: in.LayerSpacing,
	})
	if err != nil {
		return ClusterLayout{}, fmt.Errorf("autog: top-level layout: %w", err)
	}

	// Splice cluster interiors into the top-level layout.
	posByID := map[string]Node{}
	for _, n := range topOut.Nodes {
		posByID[n.ID] = n
	}

	final := ClusterLayout{Width: topOut.Width, Height: topOut.Height}
	for _, r := range results {
		clusterNode := posByID["__cluster__"+r.id]
		// Cluster interior origin = cluster super-node origin + (clusterPad, clusterPad + titleBar).
		ox := clusterNode.X + clusterPad
		oy := clusterNode.Y + clusterPad + 14
		for _, n := range r.nodes {
			final.Nodes = append(final.Nodes, Node{
				ID:     n.ID,
				Width:  n.Width,
				Height: n.Height,
				X:      ox + n.X,
				Y:      oy + n.Y,
			})
		}
		for _, e := range r.edges {
			pts := make([][2]float64, len(e.Points))
			for i, p := range e.Points {
				pts[i] = [2]float64{ox + p[0], oy + p[1]}
			}
			final.Edges = append(final.Edges, Edge{FromID: e.FromID, ToID: e.ToID, Points: pts})
		}
		final.ClusterRects = append(final.ClusterRects, ClusterRect{
			ID:    r.id,
			Title: r.title,
			BBox:  displaylist.Rect{X: clusterNode.X, Y: clusterNode.Y, W: r.width, H: r.height},
		})
	}
	for _, n := range topOut.Nodes {
		if !isClusterID(n.ID) {
			final.Nodes = append(final.Nodes, n)
		}
	}
	// Top-level edges need their cluster-collapsed endpoints replaced
	// with the original node IDs and their points clipped to the
	// cluster boundary. For Phase 3 v1 we keep the collapsed points
	// as-is and let the splice translation handle most cases.
	for _, e := range topOut.Edges {
		final.Edges = append(final.Edges, e)
	}
	return final, nil
}

func isClusterID(id string) bool {
	return len(id) > 11 && id[:11] == "__cluster__"
}
```

- [ ] **Step 2.4: Run test, expect PASS.**
- [ ] **Step 2.5: Commit** — `git commit -m "autog: leaf-cluster layout (single level)"`

---

### Task 3: Cluster layout engine — nested clusters (recursive)

Replace the leaf-only restriction with full recursion.

**Files:** Modify `autog/cluster.go`, extend `autog/cluster_test.go`.

- [ ] **Step 3.1: Write the failing test** — a 2-level nested cluster setup. Expect both inner and outer cluster rects in the output, with the inner contained in the outer.

- [ ] **Step 3.2: Implement recursion.** Refactor `LayoutClusters` so the per-cluster layout step is a recursive helper that, for each cluster:
  1. If `c.Children` is empty: same as Task 2's leaf path.
  2. Otherwise: recursively layout each child first, treat each child as a super-node sized to its returned bbox, layout this cluster's super-nodes plus its directly-contained leaf nodes, then splice child interiors into the result.

The recursive helper returns `(localNodes, localEdges, localRects, w, h)` — local to the cluster's coordinate frame (origin 0,0). The caller adds an offset when splicing.

- [ ] **Step 3.3: Run, expect PASS** for both leaf and nested fixtures.
- [ ] **Step 3.4: Commit** — `git commit -m "autog: recursive nested-cluster layout"`

---

### Task 4: Wire clusters into flowchart layout

**Files:** Modify `flowchart/layout.go`, add `flowchart/layout_cluster_test.go`.

- [ ] **Step 4.1: Update Layout to use LayoutClusters** when the AST has subgraphs. Keep the flat path for diagrams without subgraphs (slight optimization + matches Phase 2 golden tests).

```go
func Layout(d *Diagram, opts mermaid.LayoutOptions) *displaylist.DisplayList {
	// ... measure nodes ...
	if len(d.Subgraphs) == 0 {
		return layoutFlat(d, autogNodes, autogEdges, opts)
	}
	return layoutWithClusters(d, autogNodes, autogEdges, opts)
}

func layoutWithClusters(d *Diagram, autogNodes []autog.Node, autogEdges []autog.Edge, opts mermaid.LayoutOptions) *displaylist.DisplayList {
	clusters := convertSubgraphs(d.Subgraphs)
	in := autog.ClusterInput{
		Direction:    autogDir(d.Direction),
		NodeSpacing:  opts.NodeSpacing,
		LayerSpacing: opts.LayerSpacing,
		Nodes:        autogNodes,
		Edges:        autogEdges,
		Clusters:     clusters,
	}
	out, err := autog.LayoutClusters(in)
	if err != nil {
		return &displaylist.DisplayList{}
	}
	dl := &displaylist.DisplayList{Width: out.Width, Height: out.Height}
	emitClusterRects(dl, out.ClusterRects)
	emitNodesAndEdges(dl, out.Nodes, out.Edges, d)
	return dl
}

func convertSubgraphs(sgs []*Subgraph) []*autog.Cluster {
	out := make([]*autog.Cluster, len(sgs))
	for i, sg := range sgs {
		out[i] = &autog.Cluster{
			ID:       sg.ID,
			Title:    sg.Label,
			NodeIDs:  sg.NodeIDs,
			Children: convertSubgraphs(sg.Children),
		}
	}
	return out
}

func emitClusterRects(dl *displaylist.DisplayList, rects []autog.ClusterRect) {
	for _, r := range rects {
		dl.Items = append(dl.Items, displaylist.Cluster{
			BBox:  r.BBox,
			Title: r.Title,
			Role:  displaylist.RoleSubgraph,
		})
		emitClusterRects(dl, r.Children)
	}
}
```

(Refactor the existing Phase 2 `Layout` body into a private `layoutFlat` helper that takes the same inputs.)

- [ ] **Step 4.2: Add fixtures and golden tests** for clustered flowcharts. Re-run `go test ./flowchart/... -update` to seed.
- [ ] **Step 4.3: Commit** — `git commit -m "flowchart: use cluster-aware layout when subgraphs present"`

---

### Task 5: PDF emitter — render Cluster items

**Files:** Modify `pdf/emit.go`, `pdf/draw_shape.go` (new helper).

- [ ] **Step 5.1: Add a Cluster handler to `DrawInto`'s switch** that draws a backdrop rect (with optional dashed stroke) plus the cluster title at the top.

```go
case displaylist.Cluster:
	x, y, w, h := tr(v.BBox)
	rs := style.lookup(v.Role)
	applyStroke(pdf, rs)
	pdf.Rect(x, y, w, h, "D")
	if v.Title != "" {
		titleStyle := style.lookup(displaylist.RoleClusterTitle)
		pdf.SetFont(titleStyle.Font, titleStyle.FontStyle, titleStyle.FontSize)
		pdf.SetTextColor(int(titleStyle.TextR), int(titleStyle.TextG), int(titleStyle.TextB))
		pdf.Text(x+3, y+4, v.Title)
	}
```

(Place this BEFORE the Shape case so cluster rects render behind nodes.)

- [ ] **Step 5.2: Add a clustered-fixture PDF integration test** ensuring the PDF output contains roughly proportional sizing.
- [ ] **Step 5.3: Commit** — `git commit -m "mermaidpdf: render Cluster backdrops"`

---

### Task 6: Property test — cluster containment

**Files:** Modify `flowchart/property_test.go`.

- [ ] **Step 6.1: Add a generator branch that emits subgraphs** (single-level only for property test simplicity).
- [ ] **Step 6.2: Add an invariant**: for every Shape whose AST node is a member of a subgraph, the Shape's bbox must lie within the cluster's bbox.

```go
// inside the existing TestPropertyNoPanic loop, after Layout:
clusterRectByID := map[string]displaylist.Rect{}
for _, it := range dl.Items {
	if c, ok := it.(displaylist.Cluster); ok {
		clusterRectByID[c.Title] = c.BBox // using Title as ID-ish for test purposes
	}
}
// (Use a more careful index — keep the cluster ID in the AST so we
// can map node IDs back to cluster IDs and look up by that.)
```

- [ ] **Step 6.3: Commit** — `git commit -m "flowchart: cluster containment property test"`

---

### Task 7: Phase 3 smoke test

**Files:** Modify `mermaid_smoke_test.go`.

- [ ] **Step 7.1: Add subgraph case**

```go
func TestPhase3Smoke(t *testing.T) {
	src := `flowchart TB
subgraph G1 [Outer]
  A --> B
  subgraph G2 [Inner]
    C --> D
  end
  B --> C
end
`
	dl, err := ParseAndLayout(src, LayoutOptions{})
	if err != nil {
		t.Fatalf("ParseAndLayout: %v", err)
	}
	clusters := 0
	for _, it := range dl.Items {
		if _, ok := it.(displaylist.Cluster); ok {
			clusters++
		}
	}
	if clusters < 2 {
		t.Fatalf("expected ≥2 clusters in nested subgraph, got %d", clusters)
	}
}
```

- [ ] **Step 7.2: `go test ./...` clean.**
- [ ] **Step 7.3: Commit** — `git commit -m "mermaid: phase 3 smoke test"`

---

## Self-Review

| Spec section | Phase 3 task |
|---|---|
| Recursive per-cluster layout | Tasks 1, 2, 3 |
| Cross-cluster edges (intra-cluster vs spanning) | Task 2 (intra-cluster), Task 3 (spanning) |
| Cluster backdrop with title | Task 4 (emit), Task 5 (render) |
| Nested subgraphs | Task 3 |
| `displaylist.Cluster` items | Task 4 |
| Containment invariant | Task 6 |

## Open Questions / Risks

- **Edge endpoint clipping at cluster boundaries**: when a top-level edge connects two clusters, autog's polyline runs between the two cluster super-nodes. For visual clarity we may want to clip the polyline at the cluster bbox edge (so the line emerges from the cluster border, not the centre). Not wired in this phase; revisit if cross-cluster edges look bad in real samples.
- **Direction inheritance**: `subgraph X direction LR` lets a child cluster override its parent's direction. Phase 3 v1 ignores `direction` overrides; the cluster always inherits the diagram-level direction. Add in a follow-up if real samples need it.
