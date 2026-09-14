package okf

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestModelPolymorphicParsing(t *testing.T) {
	yamlInput := `
type: concept
title: "Distributed Consensus"
description: "Core algorithms for distributed agreement."
status: stable
tags: [distributed-systems, consensus, raft]
relations:
  - type: depends_on
    target: "[[Network Partitions]]"
  - type: related_to
    target: "Paxos.md"
generated:
  by: "agent:ceres-gingashi/gemini-3.8-flash"
  at: "2026-09-14T00:00:00Z"
verified:
  by: "human:reviewer"
  at: "2026-09-14T12:00:00Z"
sources:
  - "https://raft.github.io"
  - resource: "https://lamport.azurewebsites.net/pubs/paxos-simple.pdf"
    title: "Paxos Made Simple"
    author: "Leslie Lamport"
    usage_count: 5
stale_after: "2027-01-01T00:00:00Z"
custom_field: "custom_value"
priority: 10
`
	var doc OKFDocument
	err := yaml.Unmarshal([]byte(yamlInput), &doc)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if doc.Type != "concept" {
		t.Errorf("expected type 'concept', got '%s'", doc.Type)
	}
	if doc.Title != "Distributed Consensus" {
		t.Errorf("expected title 'Distributed Consensus', got '%s'", doc.Title)
	}
	if doc.Status != "stable" {
		t.Errorf("expected status 'stable', got '%s'", doc.Status)
	}
	if len(doc.Tags) != 3 || doc.Tags[0] != "distributed-systems" {
		t.Errorf("expected 3 tags, got %v", doc.Tags)
	}
	if len(doc.Relations) != 2 || doc.Relations[0].Type != "depends_on" || doc.Relations[0].Target != "[[Network Partitions]]" {
		t.Errorf("unexpected relations: %v", doc.Relations)
	}
	if doc.Generated == nil || doc.Generated.By != "agent:ceres-gingashi/gemini-3.8-flash" {
		t.Errorf("unexpected generated event: %+v", doc.Generated)
	}

	// Verified should have been parsed as single item in slice
	if len(doc.Verified) != 1 || doc.Verified[0].By != "human:reviewer" {
		t.Errorf("expected 1 verified event, got %+v", doc.Verified)
	}

	// Sources should have 1 scalar and 1 struct
	if len(doc.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(doc.Sources))
	}
	if doc.Sources[0].Resource != "https://raft.github.io" {
		t.Errorf("expected scalar source resource, got '%s'", doc.Sources[0].Resource)
	}
	if doc.Sources[1].Title != "Paxos Made Simple" || doc.Sources[1].UsageCount != 5 {
		t.Errorf("expected struct source, got %+v", doc.Sources[1])
	}

	// Custom inline attributes
	if doc.Attributes["custom_field"] != "custom_value" {
		t.Errorf("expected custom_field 'custom_value', got '%v'", doc.Attributes["custom_field"])
	}
	if doc.Attributes["priority"] != 10 {
		t.Errorf("expected priority 10, got '%v'", doc.Attributes["priority"])
	}
}

func TestModelVerifiedList(t *testing.T) {
	yamlInput := `
type: standard
verified:
  - by: "agent:linter"
    at: "2026-09-01"
  - by: "human:alice"
    at: "2026-09-02"
`
	var doc OKFDocument
	err := yaml.Unmarshal([]byte(yamlInput), &doc)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(doc.Verified) != 2 {
		t.Fatalf("expected 2 verified events, got %d", len(doc.Verified))
	}
	if doc.Verified[0].By != "agent:linter" || doc.Verified[1].By != "human:alice" {
		t.Errorf("unexpected verified events: %+v", doc.Verified)
	}
}

func TestDetermineKind(t *testing.T) {
	tests := []struct {
		explicit NodeKind
		stem     string
		relPath  string
		expected NodeKind
	}{
		{"", "index", "index.md", KindIndex},
		{"", "Architecture Dashboard", "Architecture Dashboard.md", KindIndex},
		{"", "Dashboard", "Dashboard.md", KindIndex},
		{"", "log", "log.md", KindLog},
		{"", "changelog", "changelog.md", KindLog},
		{"", "concept_node", "concepts/concept_node.md", KindConcept},
		{KindIndex, "custom", "custom.md", KindIndex},
		{KindLog, "custom_log", "custom_log.md", KindLog},
	}

	for _, tc := range tests {
		got := DetermineKind(tc.explicit, tc.stem, tc.relPath)
		if got != tc.expected {
			t.Errorf("DetermineKind(%q, %q, %q) = %q; want %q", tc.explicit, tc.stem, tc.relPath, got, tc.expected)
		}
	}
}
