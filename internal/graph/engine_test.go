package graph

import (
	"os"
	"path/filepath"
	"testing"

	"okf/internal/scanner"
)

func TestBuildGraphAndNeighborhood(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"node_a.md": `---
type: concept
title: "Node A"
description: "First node."
relations:
  - type: depends_on
    target: "[[node_b]]"
  - type: related_to
    target: "non_existent_note"
---
# Node A
Links to [[node_c|Third Node]] and [Node B](node_b.md).
`,
		"node_b.md": `---
type: concept
title: "Node B"
description: "Second node."
---
# Node B
Backlink source for Node C: [[node_c]]
`,
		"node_c.md": `---
type: concept
title: "Node C"
description: "Third node."
---
# Node C
Leaf node.
`,
	}

	for rel, content := range files {
		fullPath := filepath.Join(tempDir, rel)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cat, err := scanner.ScanVault(tempDir, nil)
	if err != nil {
		t.Fatalf("ScanVault failed: %v", err)
	}

	g, err := BuildGraph(cat, true)
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// Check dead link
	if len(g.DeadEdges) != 1 || g.DeadEdges[0].RawTarget != "non_existent_note" {
		t.Errorf("expected 1 dead edge for non_existent_note, got: %+v", g.DeadEdges)
	}

	// Check node_c neighborhood
	nodeC := cat.FindNode("node_c")
	if nodeC == nil {
		t.Fatal("node_c not found")
	}

	neighC := g.GetNeighborhood(nodeC)
	if len(neighC.Backlinks) != 2 {
		t.Fatalf("expected 2 backlinks to node_c (from node_a and node_b), got %d: %+v", len(neighC.Backlinks), neighC.Backlinks)
	}

	// Check node_a neighborhood
	nodeA := cat.FindNode("node_a")
	neighA := g.GetNeighborhood(nodeA)
	if len(neighA.DeclaredRelations) != 2 {
		t.Errorf("expected 2 declared relations on node_a, got %d", len(neighA.DeclaredRelations))
	}
	if len(neighA.BodyLinks) != 2 {
		t.Errorf("expected 2 body links on node_a, got %d", len(neighA.BodyLinks))
	}
}
