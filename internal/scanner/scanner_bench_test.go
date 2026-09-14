package scanner_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"okf/internal/graph"
	"okf/internal/scanner"
)

func createBenchmarkVault(b *testing.B, numFiles int) string {
	b.Helper()
	dir := b.TempDir()

	for i := 0; i < numFiles; i++ {
		targetIdx1 := (i + 1) % numFiles
		targetIdx2 := (i + 2) % numFiles
		content := fmt.Sprintf(`---
type: concept
title: "Concept %d"
description: "High-performance distributed system concept %d."
status: stable
tags: [benchmark, okf, v02]
relations:
  - type: depends_on
    target: "[[concept_%d]]"
generated:
  by: "agent:benchmark"
  at: "2026-09-14T00:00:00Z"
verified:
  by: "human:auditor"
  at: "2026-09-14T01:00:00Z"
sources:
  - "https://example.com/source/%d"
---
# Concept %d

This concept links to [[concept_%d|Next Concept]] and [Another](concept_%d.md).
Body content with some paragraphs and context.
`, i, i, targetIdx1, i, i, targetIdx1, targetIdx2)

		subDir := filepath.Join(dir, fmt.Sprintf("folder_%d", i%10))
		_ = os.MkdirAll(subDir, 0755)
		filePath := filepath.Join(subDir, fmt.Sprintf("concept_%d.md", i))
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("failed to write benchmark file: %v", err)
		}
	}

	return dir
}

func BenchmarkScanVault1000Docs(b *testing.B) {
	vaultDir := createBenchmarkVault(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cat, err := scanner.ScanVault(vaultDir, nil)
		if err != nil {
			b.Fatalf("ScanVault failed: %v", err)
		}
		if len(cat.Nodes) != 1000 {
			b.Fatalf("expected 1000 nodes, got %d", len(cat.Nodes))
		}
	}
}

func BenchmarkScanAndBuildGraph1000Docs(b *testing.B) {
	vaultDir := createBenchmarkVault(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cat, err := scanner.ScanVault(vaultDir, nil)
		if err != nil {
			b.Fatalf("ScanVault failed: %v", err)
		}
		g, err := graph.BuildGraph(cat, true)
		if err != nil {
			b.Fatalf("BuildGraph failed: %v", err)
		}
		if len(g.ForwardEdges) == 0 {
			b.Fatal("expected forward edges")
		}
	}
}
