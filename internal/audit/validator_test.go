package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"okf/internal/graph"
	"okf/internal/scanner"
)

func TestRunAudit(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"valid_concept.md": `---
type: concept
title: "Valid Concept"
description: "Has all required fields."
status: stable
stale_after: "2030-01-01T00:00:00Z"
---
# Valid
`,
		"missing_type.md": `---
title: "No Type"
description: "Missing type."
status: draft
---
# No Type
`,
		"missing_desc.md": `---
type: concept
title: "No Description"
status: draft
---
# No Desc
`,
		"stale.md": `---
type: concept
title: "Stale Concept"
description: "This concept is old."
status: stable
stale_after: "2020-01-01T00:00:00Z"
---
# Stale
`,
		"broken_link.md": `---
type: concept
title: "Broken Link"
description: "Points to nowhere."
relations:
  - type: depends_on
    target: "[[does_not_exist]]"
---
# Broken
`,
		"no_fm.md": `# Missing frontmatter completely`,
	}

	for rel, content := range files {
		fullPath := filepath.Join(tempDir, rel)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cat, err := scanner.ScanVault(tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	g, err := graph.BuildGraph(cat, true)
	if err != nil {
		t.Fatal(err)
	}

	simulatedNow := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	report := RunAudit(g, simulatedNow)

	if report.Passed {
		t.Errorf("expected audit to fail due to errors, but it passed")
	}

	ruleHits := make(map[string]int)
	for _, issue := range report.Issues {
		ruleHits[issue.Rule]++
	}

	expectedRules := []string{
		"missing_frontmatter",
		"missing_concept_type",
		"missing_description",
		"stale_concept",
		"dead_link",
	}

	for _, r := range expectedRules {
		if ruleHits[r] == 0 {
			t.Errorf("expected rule hit for '%s', got 0. All hits: %+v", r, ruleHits)
		}
	}

	if report.ErrorCount < 3 {
		t.Errorf("expected at least 3 errors, got %d", report.ErrorCount)
	}
	if report.WarningCount < 2 {
		t.Errorf("expected at least 2 warnings, got %d", report.WarningCount)
	}
}
