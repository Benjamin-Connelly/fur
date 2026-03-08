package index

import (
	"fmt"
	"sort"
	"strings"
)

// ToDOT generates a DOT-format directed graph string from the link graph.
// Nodes represent files (labeled with relative paths), edges represent links.
// Broken links use dashed red edges.
func (g *LinkGraph) ToDOT() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var b strings.Builder
	b.WriteString("digraph links {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, style=rounded, fontname=\"Helvetica\"];\n")
	b.WriteString("  edge [fontname=\"Helvetica\", fontsize=10];\n\n")

	// Collect all unique nodes and assign stable IDs
	nodeSet := make(map[string]bool)
	for source, links := range g.forward {
		nodeSet[source] = true
		for _, link := range links {
			nodeSet[link.Target] = true
		}
	}

	// Sort for deterministic output
	nodes := make([]string, 0, len(nodeSet))
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)

	nodeID := make(map[string]string, len(nodes))
	for i, n := range nodes {
		id := fmt.Sprintf("n%d", i)
		nodeID[n] = id
		b.WriteString(fmt.Sprintf("  %s [label=%q];\n", id, n))
	}

	if len(nodes) > 0 {
		b.WriteString("\n")
	}

	// Sort sources for deterministic edge order
	sources := make([]string, 0, len(g.forward))
	for source := range g.forward {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	for _, source := range sources {
		for _, link := range g.forward[source] {
			srcID := nodeID[source]
			tgtID := nodeID[link.Target]
			if link.Broken {
				b.WriteString(fmt.Sprintf("  %s -> %s [style=dashed, color=red];\n", srcID, tgtID))
			} else {
				b.WriteString(fmt.Sprintf("  %s -> %s;\n", srcID, tgtID))
			}
		}
	}

	b.WriteString("}\n")
	return b.String()
}
