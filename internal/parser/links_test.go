package parser

import (
	"testing"
)

func TestExtractLinks(t *testing.T) {
	content := `
# Sample Document

Here is a simple [[Direct Note]] link and an aliased [[Complex Architecture|The Arch]].
Also check out [[System Design#Scalability]] and [[Microservices#Patterns|Service Mesh]].

In addition to Obsidian wikilinks, we have standard markdown links:
See the [Raft Protocol](consensus/raft.md) or [Local Relative](./relative.md#subheading).
External link should be ignored: [Google](https://google.com).
`
	links := ExtractLinks(content)
	if len(links) != 6 {
		t.Fatalf("expected 6 links, got %d: %+v", len(links), links)
	}

	expected := []struct {
		target string
		anchor string
		text   string
		lType  string
	}{
		{"Direct Note", "", "Direct Note", "wikilink"},
		{"Complex Architecture", "", "The Arch", "wikilink"},
		{"System Design", "Scalability", "System Design", "wikilink"},
		{"Microservices", "Patterns", "Service Mesh", "wikilink"},
		{"consensus/raft.md", "", "Raft Protocol", "markdown_link"},
		{"./relative.md", "subheading", "Local Relative", "markdown_link"},
	}

	for i, exp := range expected {
		if links[i].Target != exp.target {
			t.Errorf("[%d] expected target '%s', got '%s'", i, exp.target, links[i].Target)
		}
		if links[i].Anchor != exp.anchor {
			t.Errorf("[%d] expected anchor '%s', got '%s'", i, exp.anchor, links[i].Anchor)
		}
		if links[i].Text != exp.text {
			t.Errorf("[%d] expected text '%s', got '%s'", i, exp.text, links[i].Text)
		}
		if links[i].Type != exp.lType {
			t.Errorf("[%d] expected type '%s', got '%s'", i, exp.lType, links[i].Type)
		}
	}
}
