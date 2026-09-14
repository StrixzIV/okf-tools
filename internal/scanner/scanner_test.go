package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanVault(t *testing.T) {
	tempDir := t.TempDir()

	// Create structure:
	// /concepts/consensus.md
	// /concepts/raft.md
	// /.obsidian/app.json (hidden dir - should be skipped)
	// /index.md
	// /log.md
	// /unparseable.md (no frontmatter)

	if err := os.MkdirAll(filepath.Join(tempDir, "concepts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, ".obsidian"), 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"concepts/consensus.md": `---
type: concept
title: "Consensus"
description: "General consensus mechanisms."
status: stable
relations:
  - type: depends_on
    target: "[[raft]]"
---
# Consensus
`,
		"concepts/raft.md": `---
type: concept
title: "Raft"
description: "Raft consensus protocol."
status: stable
---
# Raft
`,
		".obsidian/ignored.md": `# Should be ignored`,
		"index.md": `---
title: "Vault Index"
description: "Root index."
---
# Index
`,
		"log.md": `---
title: "Changelog"
---
# Log
`,
		"unparseable.md": `# No frontmatter at all`,
	}

	for rel, content := range files {
		fullPath := filepath.Join(tempDir, rel)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", rel, err)
		}
	}

	catalog, err := ScanVault(tempDir, nil)
	if err != nil {
		t.Fatalf("ScanVault failed: %v", err)
	}

	// Should have 5 files scanned (.obsidian ignored)
	if len(catalog.Nodes) != 5 {
		t.Fatalf("expected 5 scanned nodes, got %d", len(catalog.Nodes))
	}

	// Test find node
	node := catalog.FindNode("Raft")
	if node == nil || node.Doc == nil || node.Doc.Title != "Raft" {
		t.Fatalf("FindNode('Raft') failed: %+v", node)
	}

	// Test find node by relative path
	nodeRel := catalog.FindNode("concepts/consensus.md")
	if nodeRel == nil || nodeRel.Doc == nil || nodeRel.Doc.Title != "Consensus" {
		t.Fatalf("FindNode('concepts/consensus.md') failed: %+v", nodeRel)
	}

	// Check unparseable node recorded
	unparseable := catalog.FindNode("unparseable")
	if unparseable == nil {
		t.Fatal("expected to find unparseable node")
	}
	if unparseable.Err == nil {
		t.Fatal("expected error on unparseable node")
	}
}
