package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func createTestVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"concepts/raft.md": `---
type: concept
title: "Raft"
description: "Raft consensus protocol."
status: stable
tags: [consensus, distributed]
relations:
  - type: depends_on
    target: "[[paxos]]"
---
# Raft
Links to [Paxos](paxos.md) and [[gossip]].
`,
		"concepts/paxos.md": `---
type: concept
title: "Paxos"
description: "Classic consensus protocol."
status: stable
tags: [consensus]
---
# Paxos
Referenced by Raft.
`,
		"concepts/gossip.md": `---
type: concept
title: "Gossip Protocol"
description: "Epidemic broadcast protocol."
status: draft
tags: [networking]
---
# Gossip
`,
		"index.md": `---
title: "Index"
description: "Main vault index."
---
# Index
`,
	}

	for rel, content := range files {
		fullPath := filepath.Join(dir, rel)
		_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", rel, err)
		}
	}

	return dir
}

func executeCommand(args ...string) (string, error) {
	// Reset all flags across command tree
	resetFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func resetFlags() {
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, sub := range rootCmd.Commands() {
		sub.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
}

func TestCLI_List(t *testing.T) {
	vaultDir := createTestVault(t)

	// 1. Human output
	out, err := executeCommand("list", "--vault", vaultDir)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(out, "raft") || !strings.Contains(out, "paxos") {
		t.Errorf("list output missing expected notes: %s", out)
	}

	// 2. Filter by status
	outDraft, err := executeCommand("list", "--vault", vaultDir, "--status", "draft")
	if err != nil {
		t.Fatalf("list --status draft failed: %v", err)
	}
	if !strings.Contains(outDraft, "concepts/gossip.md") || strings.Contains(outDraft, "concepts/raft.md") {
		t.Errorf("list --status draft filtered incorrectly: %s", outDraft)
	}

	// 3. JSON output
	jsonOut, err := executeCommand("list", "--vault", vaultDir, "--json")
	if err != nil {
		t.Fatalf("list --json failed: %v", err)
	}
	var docs []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &docs); err != nil {
		t.Fatalf("failed to parse list --json: %v (raw: %s)", err, jsonOut)
	}
	if len(docs) != 4 {
		t.Errorf("expected 4 docs, got %d", len(docs))
	}
}

func TestCLI_Get(t *testing.T) {
	vaultDir := createTestVault(t)

	// 1. Human output
	out, err := executeCommand("get", "raft", "--vault", vaultDir)
	if err != nil {
		t.Fatalf("get raft failed: %v", err)
	}
	if !strings.Contains(out, "Raft consensus protocol") || !strings.Contains(out, "depends_on") {
		t.Errorf("get raft output missing expected fields: %s", out)
	}

	// 2. JSON output
	jsonOut, err := executeCommand("get", "raft", "--vault", vaultDir, "--json")
	if err != nil {
		t.Fatalf("get raft --json failed: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &doc); err != nil {
		t.Fatalf("failed to parse get --json: %v", err)
	}
	if doc["title"] != "Raft" || doc["type"] != "concept" {
		t.Errorf("unexpected get JSON: %+v", doc)
	}

	// 3. Not found
	_, errNotFound := executeCommand("get", "non_existent", "--vault", vaultDir)
	if errNotFound == nil {
		t.Errorf("expected error for non_existent note, got nil")
	}
}

func TestCLI_Relations(t *testing.T) {
	vaultDir := createTestVault(t)

	// Check neighborhood of raft
	out, err := executeCommand("relations", "raft", "--vault", vaultDir)
	if err != nil {
		t.Fatalf("relations failed: %v", err)
	}
	if !strings.Contains(out, "Declared Relations") || !strings.Contains(out, "Outgoing Body Links") {
		t.Errorf("relations output missing expected sections: %s", out)
	}

	// Check backlinks of paxos
	paxosOut, err := executeCommand("relations", "paxos", "--vault", vaultDir)
	if err != nil {
		t.Fatalf("relations paxos failed: %v", err)
	}
	if !strings.Contains(paxosOut, "Incoming Backlinks") || !strings.Contains(paxosOut, "raft") {
		t.Errorf("paxos backlinks missing raft: %s", paxosOut)
	}

	// Check JSON
	jsonOut, err := executeCommand("relations", "paxos", "--vault", vaultDir, "--json")
	if err != nil {
		t.Fatalf("relations paxos --json failed: %v", err)
	}
	var neigh map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &neigh); err != nil {
		t.Fatalf("failed to parse relations JSON: %v", err)
	}
	if neigh["node"] == nil || neigh["backlinks"] == nil {
		t.Errorf("invalid relations JSON structure: %+v", neigh)
	}
}

func TestCLI_Audit(t *testing.T) {
	vaultDir := createTestVault(t)

	// In createTestVault, all notes have frontmatter, description, and links resolve to paxos and gossip.
	// However, index.md has kind "index" so it doesn't require type.
	// So createTestVault is clean and audit should succeed.
	out, err := executeCommand("audit", "--vault", vaultDir)
	if err != nil {
		t.Fatalf("audit failed on valid vault: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "Errors:              0") {
		t.Errorf("expected 0 errors in audit: %s", out)
	}

	// Test JSON audit
	jsonOut, err := executeCommand("audit", "--vault", vaultDir, "--json")
	if err != nil {
		t.Fatalf("audit --json failed: %v", err)
	}
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &report); err != nil {
		t.Fatalf("failed to parse audit JSON: %v", err)
	}
	if report["passed"] != true {
		t.Errorf("expected audit passed = true, got %+v", report)
	}
}

func TestCLI_Export(t *testing.T) {
	vaultDir := createTestVault(t)

	jsonOut, err := executeCommand("export", "--vault", vaultDir)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	var exportData struct {
		VaultPath string                   `json:"vault_path"`
		Nodes     []map[string]interface{} `json:"nodes"`
		Edges     []map[string]interface{} `json:"edges"`
	}

	if err := json.Unmarshal([]byte(jsonOut), &exportData); err != nil {
		t.Fatalf("failed to parse export JSON: %v (raw: %s)", err, jsonOut)
	}

	if len(exportData.Nodes) != 4 {
		t.Errorf("expected 4 nodes in export, got %d", len(exportData.Nodes))
	}
	if len(exportData.Edges) == 0 {
		t.Errorf("expected edges in export, got 0")
	}
}
