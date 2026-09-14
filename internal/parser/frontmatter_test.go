package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPeekFrontmatter(t *testing.T) {
	tempDir := t.TempDir()

	validFile := filepath.Join(tempDir, "valid.md")
	validContent := `---
type: concept
title: "Valid Node"
description: "A test concept note."
status: stable
tags:
  - test
  - okf
---
# Header

This is the markdown body.
`
	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := PeekFrontmatter(validFile, 16384)
	if err != nil {
		t.Fatalf("PeekFrontmatter failed: %v", err)
	}

	if result.Doc.Title != "Valid Node" {
		t.Errorf("expected title 'Valid Node', got '%s'", result.Doc.Title)
	}
	if result.Doc.Type != "concept" {
		t.Errorf("expected type 'concept', got '%s'", result.Doc.Type)
	}
	if result.Doc.Status != "stable" {
		t.Errorf("expected status 'stable', got '%s'", result.Doc.Status)
	}
	if len(result.Doc.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Doc.Tags))
	}
	if result.BodyOffset <= 0 {
		t.Errorf("expected positive BodyOffset, got %d", result.BodyOffset)
	}

	// Verify seeking to BodyOffset yields the body
	f, _ := os.Open(validFile)
	f.Seek(result.BodyOffset, 0)
	bodyBytes := make([]byte, 100)
	n, _ := f.Read(bodyBytes)
	f.Close()
	if !strings.Contains(string(bodyBytes[:n]), "# Header") {
		t.Errorf("BodyOffset did not point to body start: %s", string(bodyBytes[:n]))
	}
}

func TestPeekFrontmatterMissing(t *testing.T) {
	tempDir := t.TempDir()
	noFrontmatterFile := filepath.Join(tempDir, "no_fm.md")
	content := `# Just a regular markdown file
Without frontmatter.
`
	os.WriteFile(noFrontmatterFile, []byte(content), 0644)

	_, err := PeekFrontmatter(noFrontmatterFile, 16384)
	if err != ErrMissingFrontmatter {
		t.Errorf("expected ErrMissingFrontmatter, got %v", err)
	}
}

func TestPeekFrontmatterUnterminated(t *testing.T) {
	tempDir := t.TempDir()
	unterminatedFile := filepath.Join(tempDir, "unterminated.md")
	content := `---
type: concept
title: "Unterminated"
`
	os.WriteFile(unterminatedFile, []byte(content), 0644)

	_, err := PeekFrontmatter(unterminatedFile, 16384)
	if err != ErrUnterminatedFrontmatter {
		t.Errorf("expected ErrUnterminatedFrontmatter, got %v", err)
	}
}

func TestPeekFrontmatterBoundedLimit(t *testing.T) {
	tempDir := t.TempDir()
	hugeFile := filepath.Join(tempDir, "huge.md")
	// Frontmatter larger than 500 bytes when limit is 500
	var sb strings.Builder
	sb.WriteString("---\ntype: concept\ntitle: \"Huge\"\nlong_text: \"")
	for i := 0; i < 1000; i++ {
		sb.WriteString("A")
	}
	sb.WriteString("\"\n---\nBody")
	os.WriteFile(hugeFile, []byte(sb.String()), 0644)

	_, err := PeekFrontmatter(hugeFile, 500)
	if err != ErrUnterminatedFrontmatter {
		t.Errorf("expected ErrUnterminatedFrontmatter on bounded limit exceed, got %v", err)
	}
}
