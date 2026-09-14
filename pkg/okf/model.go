package okf

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type NodeKind string

const (
	KindConcept NodeKind = "concept"
	KindIndex   NodeKind = "index"
	KindLog     NodeKind = "log"
)

// OKFDocument represents an OKF v0.2 knowledge node.
type OKFDocument struct {
	Path        string                 `json:"path"`
	RelPath     string                 `json:"rel_path"`
	Stem        string                 `json:"stem"`
	Kind        NodeKind               `yaml:"kind,omitempty" json:"kind"`
	Type        string                 `yaml:"type" json:"type"`
	Title       string                 `yaml:"title" json:"title"`
	Description string                 `yaml:"description" json:"description"`
	Status      string                 `yaml:"status" json:"status"` // draft | stable | deprecated
	Tags        []string               `yaml:"tags,omitempty" json:"tags"`
	Relations   []Relation             `yaml:"relations,omitempty" json:"relations"`
	Generated   *ActorEvent            `yaml:"generated,omitempty" json:"generated,omitempty"`
	Verified    []ActorEvent           `json:"verified,omitempty"`
	Sources     []Source               `json:"sources,omitempty"`
	StaleAfter  string                 `yaml:"stale_after,omitempty" json:"stale_after,omitempty"`
	Attributes  map[string]interface{} `yaml:",inline" json:"attributes,omitempty"`
}

// Relation declares a typed edge to another document or external resource.
type Relation struct {
	Type   string `yaml:"type" json:"type"`     // depends_on | derives_from | related_to | supersedes | custom
	Target string `yaml:"target" json:"target"` // Absolute path, relative path, or [[Wikilink]]
}

// ActorEvent records generation or verification by an agent or human.
type ActorEvent struct {
	By string `yaml:"by" json:"by"` // e.g. "agent:ceres-gingashi/gemini-3.8-flash", "human:strixziv"
	At string `yaml:"at" json:"at"` // ISO 8601 timestamp (e.g. "2026-09-14T00:00:00Z")
}

// Source records an external reference or citation.
type Source struct {
	ID         string `yaml:"id,omitempty" json:"id,omitempty"`
	Resource   string `yaml:"resource" json:"resource"`
	Title      string `yaml:"title,omitempty" json:"title,omitempty"`
	Author     string `yaml:"author,omitempty" json:"author,omitempty"`
	UsageCount int    `yaml:"usage_count,omitempty" json:"usage_count,omitempty"`
}

// DetermineKind infers the document kind based on frontmatter or filename conventions.
func DetermineKind(explicitKind NodeKind, stem, relPath string) NodeKind {
	switch explicitKind {
	case KindConcept, KindIndex, KindLog:
		return explicitKind
	}

	lowerStem := strings.ToLower(stem)
	if lowerStem == "index" || strings.HasSuffix(lowerStem, " dashboard") || lowerStem == "dashboard" {
		return KindIndex
	}
	if lowerStem == "log" || lowerStem == "changelog" {
		return KindLog
	}

	return KindConcept
}

// UnmarshalYAML implements polymorphic parsing for OKF v0.2 documents.
func (d *OKFDocument) UnmarshalYAML(value *yaml.Node) error {
	type RawDoc struct {
		Kind        NodeKind               `yaml:"kind,omitempty"`
		Type        string                 `yaml:"type"`
		Title       string                 `yaml:"title"`
		Description string                 `yaml:"description"`
		Status      string                 `yaml:"status"`
		Tags        yaml.Node              `yaml:"tags"`
		Relations   []Relation             `yaml:"relations"`
		Generated   *ActorEvent            `yaml:"generated,omitempty"`
		Verified    yaml.Node              `yaml:"verified"`
		Sources     yaml.Node              `yaml:"sources"`
		StaleAfter  string                 `yaml:"stale_after,omitempty"`
		Attributes  map[string]interface{} `yaml:",inline"`
	}

	var raw RawDoc
	if err := value.Decode(&raw); err != nil {
		return err
	}

	d.Kind = raw.Kind
	d.Type = raw.Type
	d.Title = raw.Title
	d.Description = raw.Description
	d.Status = raw.Status
	d.Relations = raw.Relations
	d.Generated = raw.Generated
	d.StaleAfter = raw.StaleAfter
	d.Attributes = raw.Attributes
	if d.Attributes == nil {
		d.Attributes = make(map[string]interface{})
	}

	// Polymorphic tags: scalar or sequence
	if raw.Tags.Kind == yaml.SequenceNode {
		for _, item := range raw.Tags.Content {
			if item.Kind == yaml.ScalarNode && item.Value != "" {
				d.Tags = append(d.Tags, item.Value)
			}
		}
	} else if raw.Tags.Kind == yaml.ScalarNode && raw.Tags.Value != "" {
		parts := strings.Split(raw.Tags.Value, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				d.Tags = append(d.Tags, trimmed)
			}
		}
	}

	// Polymorphic verified: single mapping or sequence of mappings
	if raw.Verified.Kind == yaml.MappingNode {
		var single ActorEvent
		if err := raw.Verified.Decode(&single); err == nil {
			d.Verified = []ActorEvent{single}
		}
	} else if raw.Verified.Kind == yaml.SequenceNode {
		var list []ActorEvent
		if err := raw.Verified.Decode(&list); err == nil {
			d.Verified = list
		}
	}

	// Polymorphic sources: sequence of scalars (strings) or mappings (Source structs)
	if raw.Sources.Kind == yaml.SequenceNode {
		var sources []Source
		for _, item := range raw.Sources.Content {
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

	return nil
}
