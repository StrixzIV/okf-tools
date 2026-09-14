package graph

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"okf/internal/parser"
	"okf/internal/scanner"
)

type EdgeSyntax string

const (
	SyntaxFrontmatter EdgeSyntax = "frontmatter"
	SyntaxWikilink    EdgeSyntax = "wikilink"
	SyntaxMarkdown    EdgeSyntax = "markdown"
)

// GraphEdge represents a directed connection between two documents.
type GraphEdge struct {
	Source     string               `json:"source"`
	Target     string               `json:"target"`
	RawTarget  string               `json:"raw_target"`
	Type       string               `json:"type"`
	Syntax     EdgeSyntax           `json:"syntax"`
	Label      string               `json:"label,omitempty"`
	Anchor     string               `json:"anchor,omitempty"`
	Resolved   bool                 `json:"resolved"`
	SourceNode *scanner.ScannedNode `json:"-"`
	TargetNode *scanner.ScannedNode `json:"-"`
}

// Graph holds the complete resolved bidirectional knowledge graph.
type Graph struct {
	Catalog      *scanner.VaultCatalog
	ForwardEdges map[string][]GraphEdge // Keyed by Source RelPath
	Backlinks    map[string][]GraphEdge // Keyed by Target RelPath
	DeadEdges    []GraphEdge            // Links pointing to non-existent notes
}

// BuildGraph constructs the knowledge graph from a scanned catalog.
func BuildGraph(catalog *scanner.VaultCatalog, parseBodies bool) (*Graph, error) {
	g := &Graph{
		Catalog:      catalog,
		ForwardEdges: make(map[string][]GraphEdge),
		Backlinks:    make(map[string][]GraphEdge),
	}

	// 1. Process Frontmatter Relations
	for _, node := range catalog.Nodes {
		if node.Doc == nil {
			continue
		}
		for _, rel := range node.Doc.Relations {
			edge := resolveEdge(catalog, node, rel.Target, rel.Type, SyntaxFrontmatter, "", "")
			g.addEdge(edge)
		}
	}

	// 2. Process Body Links if requested
	if parseBodies {
		type nodeLinks struct {
			node  *scanner.ScannedNode
			links []parser.ExtractedLink
		}

		numWorkers := runtime.NumCPU()
		if numWorkers < 2 {
			numWorkers = 2
		}

		jobsChan := make(chan *scanner.ScannedNode, 256)
		resultsChan := make(chan nodeLinks, 256)

		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for n := range jobsChan {
					if n.Err != nil || n.Doc == nil {
						continue
					}
					links, err := parser.ExtractBodyLinksFromFile(n.Path, n.BodyOffset)
					if err == nil && len(links) > 0 {
						resultsChan <- nodeLinks{node: n, links: links}
					}
				}
			}()
		}

		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		var bodyResults []nodeLinks
		collectorDone := make(chan struct{})
		go func() {
			for res := range resultsChan {
				bodyResults = append(bodyResults, res)
			}
			close(collectorDone)
		}()

		for _, n := range catalog.Nodes {
			jobsChan <- n
		}
		close(jobsChan)
		<-collectorDone

		for _, res := range bodyResults {
			for _, l := range res.links {
				syntax := SyntaxWikilink
				if l.Type == "markdown_link" {
					syntax = SyntaxMarkdown
				}
				edge := resolveEdge(catalog, res.node, l.Target, "links_to", syntax, l.Text, l.Anchor)
				g.addEdge(edge)
			}
		}
	}

	return g, nil
}

// resolveEdge resolves a raw link target against the catalog.
func resolveEdge(catalog *scanner.VaultCatalog, sourceNode *scanner.ScannedNode, rawTarget, relType string, syntax EdgeSyntax, label, anchor string) GraphEdge {
	cleaned := strings.TrimSpace(rawTarget)
	// Handle wikilink syntax in frontmatter: [[Note]] or [[Note#Heading|Alias]]
	if strings.HasPrefix(cleaned, "[[") && strings.HasSuffix(cleaned, "]]") {
		inner := cleaned[2 : len(cleaned)-2]
		// Check alias
		if pipeIdx := strings.Index(inner, "|"); pipeIdx != -1 {
			if label == "" {
				label = inner[pipeIdx+1:]
			}
			inner = inner[:pipeIdx]
		}
		// Check anchor
		if hashIdx := strings.Index(inner, "#"); hashIdx != -1 {
			if anchor == "" {
				anchor = inner[hashIdx+1:]
			}
			inner = inner[:hashIdx]
		}
		cleaned = inner
	}

	edge := GraphEdge{
		Source:     sourceNode.RelPath,
		RawTarget:  rawTarget,
		Type:       relType,
		Syntax:     syntax,
		Label:      label,
		Anchor:     anchor,
		SourceNode: sourceNode,
	}

	// Resolution Strategy:
	// 1. Direct relative path from vault root
	targetNode := catalog.FindNode(cleaned)

	// 2. If not found and target looks relative (e.g. ./sub.md or sub.md), resolve relative to source node dir
	if targetNode == nil && sourceNode != nil {
		sourceDir := filepath.Dir(sourceNode.RelPath)
		candidateRel := filepath.Clean(filepath.Join(sourceDir, cleaned))
		targetNode = catalog.FindNode(candidateRel)
	}

	// 3. Match by stem (stripping .md)
	if targetNode == nil {
		stem := strings.TrimSuffix(cleaned, filepath.Ext(cleaned))
		targetNode = catalog.FindNode(stem)
	}

	if targetNode != nil && targetNode.Doc != nil {
		edge.Resolved = true
		edge.Target = targetNode.RelPath
		edge.TargetNode = targetNode
	} else {
		edge.Resolved = false
		edge.Target = cleaned
	}

	return edge
}

// addEdge inserts an edge with deduplication.
func (g *Graph) addEdge(edge GraphEdge) {
	// Deduplicate in ForwardEdges
	existing := g.ForwardEdges[edge.Source]
	for _, e := range existing {
		if e.Target == edge.Target && e.Type == edge.Type && e.Syntax == edge.Syntax && e.Anchor == edge.Anchor {
			return // Duplicate
		}
	}

	g.ForwardEdges[edge.Source] = append(g.ForwardEdges[edge.Source], edge)

	if edge.Resolved {
		g.Backlinks[edge.Target] = append(g.Backlinks[edge.Target], edge)
	} else {
		g.DeadEdges = append(g.DeadEdges, edge)
	}
}

// Neighborhood represents incoming and outgoing relations for a single node.
type Neighborhood struct {
	Node              *scanner.ScannedNode `json:"node"`
	DeclaredRelations []GraphEdge          `json:"declared_relations"`
	BodyLinks         []GraphEdge          `json:"body_links"`
	Backlinks         []GraphEdge          `json:"backlinks"`
}

// GetNeighborhood returns the graph neighborhood for a given node.
func (g *Graph) GetNeighborhood(node *scanner.ScannedNode) *Neighborhood {
	if node == nil {
		return nil
	}

	res := &Neighborhood{
		Node:              node,
		DeclaredRelations: make([]GraphEdge, 0),
		BodyLinks:         make([]GraphEdge, 0),
		Backlinks:         make([]GraphEdge, 0),
	}

	// Outgoing
	outgoing := g.ForwardEdges[node.RelPath]
	for _, edge := range outgoing {
		if edge.Syntax == SyntaxFrontmatter {
			res.DeclaredRelations = append(res.DeclaredRelations, edge)
		} else {
			res.BodyLinks = append(res.BodyLinks, edge)
		}
	}

	// Incoming
	incoming := g.Backlinks[node.RelPath]
	res.Backlinks = append(res.Backlinks, incoming...)

	return res
}
