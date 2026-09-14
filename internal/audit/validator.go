package audit

import (
	"fmt"
	"strings"
	"time"

	"okf/internal/graph"
	"okf/pkg/okf"
)

type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
)

// AuditIssue represents a single compliance violation or warning.
type AuditIssue struct {
	File     string   `json:"file"`
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

// AuditReport summarizes the vault audit.
type AuditReport struct {
	VaultPath    string       `json:"vault_path"`
	TotalScanned int          `json:"total_scanned"`
	ErrorCount   int          `json:"error_count"`
	WarningCount int          `json:"warning_count"`
	Passed       bool         `json:"passed"`
	Issues       []AuditIssue `json:"issues"`
}

var dateLayouts = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006/01/02",
}

func parseTime(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format: %s", raw)
}

// RunAudit scans the graph and catalog for OKF v0.2 compliance.
func RunAudit(g *graph.Graph, now time.Time) *AuditReport {
	if now.IsZero() {
		now = time.Now()
	}

	report := &AuditReport{
		VaultPath:    g.Catalog.VaultPath,
		TotalScanned: len(g.Catalog.Nodes),
		Issues:       make([]AuditIssue, 0),
	}

	// 1. Audit each scanned node
	for _, node := range g.Catalog.Nodes {
		if node.Err != nil {
			report.Issues = append(report.Issues, AuditIssue{
				File:     node.RelPath,
				Rule:     "missing_frontmatter",
				Severity: SeverityError,
				Message:  fmt.Sprintf("Invalid or missing YAML frontmatter: %v", node.Err),
			})
			continue
		}

		if node.Doc == nil {
			continue
		}

		// Check required 'type' on concept nodes
		if node.Doc.Kind == okf.KindConcept && strings.TrimSpace(node.Doc.Type) == "" {
			report.Issues = append(report.Issues, AuditIssue{
				File:     node.RelPath,
				Rule:     "missing_concept_type",
				Severity: SeverityError,
				Message:  "Concept file is missing required 'type' field",
			})
		}

		// Check description
		if strings.TrimSpace(node.Doc.Description) == "" {
			report.Issues = append(report.Issues, AuditIssue{
				File:     node.RelPath,
				Rule:     "missing_description",
				Severity: SeverityWarning,
				Message:  "Note is missing a single-sentence 'description'",
			})
		}

		// Check stale_after
		if strings.TrimSpace(node.Doc.StaleAfter) != "" {
			staleDate, err := parseTime(node.Doc.StaleAfter)
			if err != nil {
				report.Issues = append(report.Issues, AuditIssue{
					File:     node.RelPath,
					Rule:     "invalid_stale_date",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("Invalid stale_after timestamp: %s", node.Doc.StaleAfter),
				})
			} else if !now.Before(staleDate) {
				report.Issues = append(report.Issues, AuditIssue{
					File:     node.RelPath,
					Rule:     "stale_concept",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("Concept has been stale since %s", staleDate.Format("2006-01-02")),
				})
			}
		}
	}

	// 2. Audit Dead / Broken Links
	for _, dead := range g.DeadEdges {
		report.Issues = append(report.Issues, AuditIssue{
			File:     dead.Source,
			Rule:     "dead_link",
			Severity: SeverityError,
			Message:  fmt.Sprintf("Dead link pointing to non-existent target '%s' (syntax: %s)", dead.RawTarget, dead.Syntax),
		})
	}

	// Compute counts
	for _, issue := range report.Issues {
		if issue.Severity == SeverityError {
			report.ErrorCount++
		} else {
			report.WarningCount++
		}
	}

	report.Passed = (report.ErrorCount == 0)
	return report
}
