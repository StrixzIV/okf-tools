# IMPLEMENTATIONS.md: High-Performance Go CLI for Google Open Knowledge Format (`okf`)

## 1. Executive Summary & Goals
Build an ultra-fast, concurrent CLI binary (`okf`) written in Go 1.21+ to parse, query, traverse, and audit knowledge vaults adhering to the **Google Open Knowledge Format (OKF v0.2)** specification.
The tool operates over local Markdown directories (such as Obsidian vaults) with sub-10ms response times, zero token bloat, and resilient dynamic structure handling.

---

## 2. Core Architecture & Design Principles

### 2.1 Low-Overhead I/O Strategy (Bounded Streaming)
- **Frontmatter Peeking:** Do NOT read whole files into memory during scanning. Read only the initial bytes up to the closing `---` or `...` delimiter (or a maximum cap of 16 KB) using `bufio.Reader` or `io.LimitReader`.
- **Lazy Body Parsing:** Only scan the Markdown body when graph links or backlinks are explicitly requested (`okf relations` or full export).

### 2.2 High-Concurrency Worker Pool
- Leverage `runtime.NumCPU()` worker goroutines feeding from a channel populated by `filepath.WalkDir`.
- Synchronize node registration via a thread-safe map or collector channel to avoid lock contention.

---

## 3. Dynamic OKF v0.2 Data Model & Polymorphic Parsing

OKF v0.2 is an open-ended specification where metadata fields can be polymorphic and documents carry extensible custom keys. The Go implementation must use custom YAML unmarshaling (`yaml.Node`) to prevent unmarshal errors.

### 3.1 Document Kinds
Per OKF v0.2 §3.1:
- `KindConcept` (`"concept"`): Standard concept files.
- `KindIndex` (`"index"`): Directory listings (`index.md`, `* Dashboard.md`).
- `KindLog` (`"log"`): Update logs (`log.md`).

### 3.2 Data Structs

```go
package okf

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

type NodeKind string

const (
	KindConcept NodeKind = "concept"
	KindIndex   NodeKind = "index"
	KindLog     NodeKind = "log"
)

type OKFDocument struct {
	Path        string                 `json:"path"`
	RelPath     string                 `json:"rel_path"`
	Stem        string                 `json:"stem"`
	Kind        NodeKind               `json:"kind"`
	Type        string                 `yaml:"type" json:"type"`
	Title       string                 `yaml:"title" json:"title"`
	Description string                 `yaml:"description" json:"description"`
	Status      string                 `yaml:"status" json:"status"` // draft | stable | deprecated
	Tags        []string               `yaml:"tags" json:"tags"`
	Relations   []Relation             `yaml:"relations" json:"relations"`
	Generated   *ActorEvent            `yaml:"generated,omitempty" json:"generated,omitempty"`
	Verified    []ActorEvent           `json:"verified,omitempty"`
	Sources     []Source               `json:"sources,omitempty"`
	StaleAfter  string                 `yaml:"stale_after,omitempty" json:"stale_after,omitempty"`
	Attributes  map[string]interface{} `yaml:",inline" json:"attributes,omitempty"`
}

type Relation struct {
	Type   string `yaml:"type" json:"type"`     // depends_on | derives_from | related_to | supersedes | custom
	Target string `yaml:"target" json:"target"` // Absolute path, relative path, or [[Wikilink]]
}

type ActorEvent struct {
	By string `yaml:"by" json:"by"` // e.g. "agent:ceres-gingashi/gemini-3.8-flash", "human:strixziv"
	At string `yaml:"at" json:"at"` // ISO 8601 timestamp (e.g. "2026-09-14T00:00:00Z")
}

type Source struct {
	ID         string `yaml:"id,omitempty" json:"id,omitempty"`
	Resource   string `yaml:"resource" json:"resource"`
	Title      string `yaml:"title,omitempty" json:"title,omitempty"`
	Author     string `yaml:"author,omitempty" json:"author,omitempty"`
	UsageCount int    `yaml:"usage_count,omitempty" json:"usage_count,omitempty"`
}
```

### 3.3 Polymorphic Unmarshalers

#### A. Polymorphic `verified` Field
In OKF v0.2 (§5.2), `verified` can appear either as a single mapping or a sequence of mappings:
```go
func (d *OKFDocument) UnmarshalYAML(value *yaml.Node) error {
	type RawDoc OKFDocument
	var raw RawDoc
	if err := value.Decode(&raw); err != nil {
		return err
	}
	*d = OKFDocument(raw)

	// Custom traversal of value.Content for polymorphic fields
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		valNode := value.Content[i+1]

		if key == "verified" {
			switch valNode.Kind {
			case yaml.MappingNode:
				var single ActorEvent
				if err := valNode.Decode(&single); err == nil {
					d.Verified = []ActorEvent{single}
				}
			case yaml.SequenceNode:
				var list []ActorEvent
				if err := valNode.Decode(&list); err == nil {
					d.Verified = list
				}
			}
		} else if key == "sources" {
			if valNode.Kind == yaml.SequenceNode {
				var sources []Source
				for _, item := range valNode.Content {
					if item.Kind == yaml.ScalarNode {
						sources = append(sources, Source{Resource: item.Value})
					} else if item.Kind == yaml.MappingNode {
						var s Source
						if err := item.Decode(&s); err == nil {
							sources = append(sources, s)
						}
					}
				}
				d.Sources = sources
			}
		}
	}
	return nil
}
```

---

## 4. Dual-Syntax Link Resolution & Graph Engine

The graph engine must reconcile connections declared in two formats:
1. **Frontmatter Relations:** `relations: [{ type: "depends_on", target: "..." }]`
2. **Body Markdown & Wikilinks:**
   - Standard Markdown Links: `[Text](/path/to/concept.md)` or `[Text](./relative.md)`
   - Obsidian Wikilinks: `[[Note Name]]` or `[[Note Name|Display Label]]`

### Link Extraction Regex Patterns
- Wikilinks: `\[\[([^\]\|]+)(?:\|[^\]]+)?\]\]`
- Markdown Links: `\[([^\]]+)\]\(([^)]+\.md)\)`

### Normalization
All links must resolve to a canonical node identifier (either relative vault path or filename stem) to maintain an in-memory bi-directional adjacency map:
- `ForwardEdges`: `map[string][]GraphEdge` (outgoing)
- `Backlinks`: `map[string][]GraphEdge` (incoming)

---

## 5. CLI Specification & Commands

Binary Name: `okf`

### Global Flags
- `--vault <path>`: Explicit path to knowledge bundle/vault. Defaults to `$OBSIDIAN_VAULT_PATH` or `.`.
- `--json`: Format command output as machine-readable JSON.

### 5.1 `okf list`
Filters and displays knowledge nodes.
- `--type <type>`: Filter by OKF concept type (e.g. `subject`, `concept`, `person`, `standard`, `research-hub`).
- `--status <status>`: Filter by lifecycle status (`draft`, `stable`, `deprecated`).
- `--tag <tag>`: Filter by tag.
- `--kind <kind>`: Filter by document kind (`concept`, `index`, `log`).

### 5.2 `okf get <query>`
Retrieves a specific node by stem, path, or title match.
- Prints clean human-readable metadata, description, lifecycle, actors, and custom attributes.
- With `--json`, outputs full serialized `OKFDocument`.

### 5.3 `okf relations <query>`
Inspects the graph neighborhood:
- Outgoing declared relations (type + target)
- Outgoing body wikilinks/markdown links
- Incoming backlinks from other notes in the vault

### 5.4 `okf audit`
Scans and evaluates vault health according to OKF v0.2 compliance:
- Files missing YAML frontmatter
- Concept files missing the required `type` field
- Notes missing single-sentence `description`
- Stale concepts where `now >= stale_after`
- Dead/broken links pointing to non-existent notes
- Returns exit code `0` on clean pass, `1` if critical errors are detected.

### 5.5 `okf export`
Dumps the complete vault knowledge graph as an adjacency list JSON (Nodes + Edges) for visualization or agent context injection.

---

## 6. Project Layout & Build Instructions

### Recommended Directory Structure
```text
okf/
├── cmd/
│   └── okf/
│       └── main.go
├── internal/
│   ├── parser/
│   │   ├── frontmatter.go
│   │   └── unmarshal.go
│   ├── graph/
│   │   ├── engine.go
│   │   └── links.go
│   └── audit/
│       └── validator.go
├── go.mod
├── go.sum
└── Makefile
```

### Build Command
```bash
go build -ldflags="-s -w" -o okf ./cmd/okf
```

### Performance Target
- Cold execution across 1,000 Markdown documents in under **15ms**.
- Memory footprint under **15MB RSS**.
