package autog

import (
	"fmt"
	"strings"

	"github.com/luo-studio/go-mermaid/displaylist"
)

// ClusterInput describes a graph with nested clusters.
type ClusterInput struct {
	Direction    Direction
	NodeSpacing  float64
	LayerSpacing float64
	Padding      float64

	Nodes    []Node
	Edges    []Edge
	Clusters []*Cluster
}

// Cluster is a nestable group of nodes.
type Cluster struct {
	ID       string
	Title    string
	NodeIDs  []string
	Children []*Cluster
}

// ClusterLayout is the output. Nodes/Edges have absolute positions in
// DisplayList space; ClusterRects describes each cluster's backdrop
// rectangle (with the same nesting structure as input).
type ClusterLayout struct {
	Width, Height float64
	Nodes         []Node
	Edges         []Edge
	ClusterRects  []ClusterRect
}

// ClusterRect is a positioned cluster.
type ClusterRect struct {
	ID       string
	Title    string
	BBox     displaylist.Rect
	Children []ClusterRect
}

const (
	clusterPad      = 5.0
	clusterTitleBar = 7.0
	clusterPrefix   = "__cluster__"
)

// isClusterID reports whether id was assigned as a cluster super-node.
func isClusterID(id string) bool { return strings.HasPrefix(id, clusterPrefix) }

// LayoutClusters runs the recursive cluster layout. Each cluster
// (bottom-up) is laid out as a flat graph of its direct members and
// child-cluster super-nodes; the parent splices the child results
// into its own coordinate frame.
func LayoutClusters(in ClusterInput) (ClusterLayout, error) {
	nodeByID := map[string]Node{}
	for _, n := range in.Nodes {
		nodeByID[n.ID] = n
	}

	// Build node→cluster, cluster→parent maps. Validate references.
	nodeCluster := map[string]string{}
	parentCluster := map[string]string{}
	clusterByID := map[string]*Cluster{}

	var visit func(c *Cluster, parent string) error
	visit = func(c *Cluster, parent string) error {
		if _, dup := clusterByID[c.ID]; dup {
			return fmt.Errorf("autog: duplicate cluster ID %q", c.ID)
		}
		clusterByID[c.ID] = c
		parentCluster[c.ID] = parent
		for _, id := range c.NodeIDs {
			if _, ok := nodeByID[id]; !ok {
				return fmt.Errorf("autog: cluster %q references unknown node %q", c.ID, id)
			}
			if existing, exists := nodeCluster[id]; exists {
				return fmt.Errorf("autog: node %q is in multiple clusters (%s and %s)", id, existing, c.ID)
			}
			nodeCluster[id] = c.ID
		}
		for _, child := range c.Children {
			if err := visit(child, c.ID); err != nil {
				return err
			}
		}
		return nil
	}
	for _, c := range in.Clusters {
		if err := visit(c, ""); err != nil {
			return ClusterLayout{}, err
		}
	}

	// Compute ancestor chains (from deepest to root) per node.
	nodeAncestors := map[string][]string{}
	for id := range nodeByID {
		chain := []string{}
		cur := nodeCluster[id]
		for cur != "" {
			chain = append(chain, cur)
			cur = parentCluster[cur]
		}
		nodeAncestors[id] = chain
	}

	// LCA per edge: the deepest shared cluster (or "" for root).
	edgeLCA := make([]string, len(in.Edges))
	for i, e := range in.Edges {
		src := nodeAncestors[e.FromID]
		tgt := nodeAncestors[e.ToID]
		lca := ""
		si, ti := len(src)-1, len(tgt)-1
		for si >= 0 && ti >= 0 && src[si] == tgt[ti] {
			lca = src[si]
			si--
			ti--
		}
		edgeLCA[i] = lca
	}

	// immediateChild returns the cluster ID directly under `clusterID`
	// on the path to `nodeID`, or "" if the node is a direct member of
	// `clusterID`.
	immediateChild := func(clusterID, nodeID string) string {
		chain := nodeAncestors[nodeID]
		if clusterID == "" {
			if len(chain) == 0 {
				return ""
			}
			return chain[len(chain)-1]
		}
		for i, c := range chain {
			if c == clusterID {
				if i == 0 {
					return ""
				}
				return chain[i-1]
			}
		}
		return ""
	}

	type clusterResult struct {
		nodes  []Node
		edges  []Edge
		rects  []ClusterRect
		width  float64
		height float64
	}
	var layout func(c *Cluster, isRoot bool) (clusterResult, error)

	layout = func(c *Cluster, isRoot bool) (clusterResult, error) {
		childResults := map[string]clusterResult{}
		for _, child := range c.Children {
			r, err := layout(child, false)
			if err != nil {
				return clusterResult{}, err
			}
			childResults[child.ID] = r
		}

		// Build autog input: direct nodes + child super-nodes.
		var autogNodes []Node
		for _, id := range c.NodeIDs {
			autogNodes = append(autogNodes, nodeByID[id])
		}
		for _, child := range c.Children {
			r := childResults[child.ID]
			autogNodes = append(autogNodes, Node{
				ID:     clusterPrefix + child.ID,
				Width:  r.width,
				Height: r.height,
			})
		}

		// Pending edges at this level. Track autog IDs (translated to
		// super-nodes when needed) and original endpoints so we can
		// re-attach metadata after autog returns.
		type pendingEdge struct {
			origFrom, origTo  string
			autogFrom, autogTo string
			consumed          bool
		}
		var pending []pendingEdge
		for i, e := range in.Edges {
			if edgeLCA[i] != c.ID {
				continue
			}
			af := e.FromID
			if ic := immediateChild(c.ID, e.FromID); ic != "" {
				af = clusterPrefix + ic
			}
			at := e.ToID
			if ic := immediateChild(c.ID, e.ToID); ic != "" {
				at = clusterPrefix + ic
			}
			pending = append(pending, pendingEdge{
				origFrom:  e.FromID,
				origTo:    e.ToID,
				autogFrom: af,
				autogTo:   at,
			})
		}

		var autogEdges []Edge
		for _, p := range pending {
			autogEdges = append(autogEdges, Edge{FromID: p.autogFrom, ToID: p.autogTo})
		}

		out, err := Layout(Input{
			Nodes:        autogNodes,
			Edges:        autogEdges,
			Direction:    in.Direction,
			NodeSpacing:  in.NodeSpacing,
			LayerSpacing: in.LayerSpacing,
		})
		if err != nil {
			return clusterResult{}, fmt.Errorf("autog: cluster %q layout: %w", c.ID, err)
		}

		// Compute padding/title bar for non-root clusters.
		pad := 0.0
		titleBar := 0.0
		if !isRoot {
			pad = clusterPad
			if c.Title != "" {
				titleBar = clusterTitleBar
			}
		}
		contentX := pad
		contentY := pad + titleBar
		var (
			result      clusterResult
			contentMaxX float64
			contentMaxY float64
		)

		// First pass: bookkeep super-node positions for splicing.
		superPos := map[string]Node{}
		for _, n := range out.Nodes {
			if isClusterID(n.ID) {
				superPos[n.ID[len(clusterPrefix):]] = n
			}
		}

		// Direct nodes: emit at content offset.
		for _, n := range out.Nodes {
			if isClusterID(n.ID) {
				continue
			}
			nn := Node{
				ID:     n.ID,
				Width:  n.Width,
				Height: n.Height,
				X:      contentX + n.X,
				Y:      contentY + n.Y,
			}
			result.nodes = append(result.nodes, nn)
			if rx := nn.X + nn.Width; rx > contentMaxX {
				contentMaxX = rx
			}
			if ry := nn.Y + nn.Height; ry > contentMaxY {
				contentMaxY = ry
			}
		}

		// Splice each child cluster's interior at the super-node
		// position autog assigned.
		for _, child := range c.Children {
			r := childResults[child.ID]
			sn, ok := superPos[child.ID]
			if !ok {
				// Shouldn't happen unless autog dropped the cluster
				// entirely (e.g., empty child). Skip.
				continue
			}
			ox := contentX + sn.X
			oy := contentY + sn.Y
			for _, n := range r.nodes {
				result.nodes = append(result.nodes, Node{
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
				result.edges = append(result.edges, Edge{
					FromID: e.FromID,
					ToID:   e.ToID,
					Points: pts,
				})
			}
			rect := ClusterRect{
				ID:    child.ID,
				Title: child.Title,
				BBox: displaylist.Rect{
					X: ox, Y: oy, W: r.width, H: r.height,
				},
				Children: translateRects(r.rects, ox, oy),
			}
			result.rects = append(result.rects, rect)
			if rx := ox + r.width; rx > contentMaxX {
				contentMaxX = rx
			}
			if ry := oy + r.height; ry > contentMaxY {
				contentMaxY = ry
			}
		}

		// Edges at this level: match autog output back to pending.
		for _, oe := range out.Edges {
			for i := range pending {
				if pending[i].consumed {
					continue
				}
				if pending[i].autogFrom == oe.FromID && pending[i].autogTo == oe.ToID {
					pts := make([][2]float64, len(oe.Points))
					for j, p := range oe.Points {
						pts[j] = [2]float64{contentX + p[0], contentY + p[1]}
					}
					result.edges = append(result.edges, Edge{
						FromID: pending[i].origFrom,
						ToID:   pending[i].origTo,
						Points: pts,
					})
					pending[i].consumed = true
					for _, p := range pts {
						if p[0] > contentMaxX {
							contentMaxX = p[0]
						}
						if p[1] > contentMaxY {
							contentMaxY = p[1]
						}
					}
					break
				}
			}
		}

		// Cluster total size.
		if isRoot {
			result.width = contentMaxX
			result.height = contentMaxY
		} else {
			result.width = contentMaxX + pad
			result.height = contentMaxY + pad
		}
		return result, nil
	}

	// Synthesize a virtual root cluster containing all unclustered
	// nodes plus the supplied root clusters.
	var unclusteredIDs []string
	for _, n := range in.Nodes {
		if _, inCluster := nodeCluster[n.ID]; !inCluster {
			unclusteredIDs = append(unclusteredIDs, n.ID)
		}
	}
	root := &Cluster{
		ID:       "",
		NodeIDs:  unclusteredIDs,
		Children: in.Clusters,
	}
	res, err := layout(root, true)
	if err != nil {
		return ClusterLayout{}, err
	}
	return ClusterLayout{
		Width:        res.width,
		Height:       res.height,
		Nodes:        res.nodes,
		Edges:        res.edges,
		ClusterRects: res.rects,
	}, nil
}

func translateRects(in []ClusterRect, dx, dy float64) []ClusterRect {
	if len(in) == 0 {
		return nil
	}
	out := make([]ClusterRect, len(in))
	for i, r := range in {
		out[i] = ClusterRect{
			ID:    r.ID,
			Title: r.Title,
			BBox: displaylist.Rect{
				X: r.BBox.X + dx,
				Y: r.BBox.Y + dy,
				W: r.BBox.W,
				H: r.BBox.H,
			},
			Children: translateRects(r.Children, dx, dy),
		}
	}
	return out
}
