package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"okf/internal/parser"
	"okf/pkg/okf"
)

// ScannedNode represents a single scanned file in the vault.
type ScannedNode struct {
	Path       string
	RelPath    string
	Stem       string
	Doc        *okf.OKFDocument
	BodyOffset int64
	Err        error
}

// VaultCatalog is an in-memory index of all nodes in a vault.
type VaultCatalog struct {
	VaultPath      string
	Nodes          []*ScannedNode
	ByRelPath      map[string]*ScannedNode
	ByRelPathLower map[string]*ScannedNode
	ByStem         map[string]*ScannedNode
	ByStemLower    map[string]*ScannedNode
	ByTitle        map[string]*ScannedNode
	ByTitleLower   map[string]*ScannedNode
}

// ScanOptions configures the vault scan.
type ScanOptions struct {
	MaxWorkers int
}

type fileJob struct {
	path    string
	relPath string
	stem    string
}

// ScanVault scans a directory concurrently using bounded streaming.
func ScanVault(vaultPath string, opts *ScanOptions) (*VaultCatalog, error) {
	absVaultPath, err := filepath.Abs(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve vault path: %w", err)
	}

	info, err := os.Stat(absVaultPath)
	if err != nil {
		return nil, fmt.Errorf("vault path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vault path is not a directory: %s", absVaultPath)
	}

	numWorkers := runtime.GOMAXPROCS(0) * 4
	if opts != nil && opts.MaxWorkers > 0 {
		numWorkers = opts.MaxWorkers
	}
	if numWorkers < 4 {
		numWorkers = 4
	}

	jobsChan := make(chan fileJob, 4096)

	var mu sync.Mutex
	nodes := make([]*ScannedNode, 0, 1024)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var localNodes []*ScannedNode

			for job := range jobsChan {
				res, pErr := parser.PeekFrontmatter(job.path, parser.DefaultMaxFrontmatterBytes)
				node := &ScannedNode{
					Path:    job.path,
					RelPath: job.relPath,
					Stem:    job.stem,
					Err:     pErr,
				}

				if pErr == nil && res != nil && res.Doc != nil {
					doc := res.Doc
					doc.Path = job.path
					doc.RelPath = job.relPath
					doc.Stem = job.stem
					doc.Kind = okf.DetermineKind(doc.Kind, job.stem, job.relPath)
					if doc.Title == "" {
						doc.Title = job.stem
					}
					node.Doc = doc
					node.BodyOffset = res.BodyOffset
				}
				localNodes = append(localNodes, node)
			}

			if len(localNodes) > 0 {
				mu.Lock()
				nodes = append(nodes, localNodes...)
				mu.Unlock()
			}
		}()
	}

	prefixLen := len(absVaultPath)
	if !strings.HasSuffix(absVaultPath, string(filepath.Separator)) {
		prefixLen++
	}

	// Walk vault directory
	walkErr := filepath.WalkDir(absVaultPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			// Skip hidden directories (.git, .obsidian, .trash) and special dirs
			if (strings.HasPrefix(name, ".") && path != absVaultPath) || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".markdown") {
			return nil
		}

		var rel string
		if len(path) >= prefixLen {
			rel = path[prefixLen:]
		} else {
			rel = name
		}

		// Fast stem
		stem := name
		if extIdx := strings.LastIndexByte(name, '.'); extIdx != -1 {
			stem = name[:extIdx]
		}

		jobsChan <- fileJob{
			path:    path,
			relPath: rel,
			stem:    stem,
		}
		return nil
	})

	close(jobsChan)
	wg.Wait()

	if walkErr != nil {
		return nil, fmt.Errorf("error walking vault directory: %w", walkErr)
	}

	catalog := &VaultCatalog{
		VaultPath:      absVaultPath,
		Nodes:          nodes,
		ByRelPath:      make(map[string]*ScannedNode, len(nodes)),
		ByRelPathLower: make(map[string]*ScannedNode, len(nodes)),
		ByStem:         make(map[string]*ScannedNode, len(nodes)),
		ByStemLower:    make(map[string]*ScannedNode, len(nodes)),
		ByTitle:        make(map[string]*ScannedNode, len(nodes)),
		ByTitleLower:   make(map[string]*ScannedNode, len(nodes)),
	}

	for _, node := range nodes {
		catalog.ByRelPath[node.RelPath] = node
		catalog.ByRelPathLower[strings.ToLower(node.RelPath)] = node
		catalog.ByStem[node.Stem] = node
		catalog.ByStemLower[strings.ToLower(node.Stem)] = node
		if node.Doc != nil && node.Doc.Title != "" {
			catalog.ByTitle[node.Doc.Title] = node
			catalog.ByTitleLower[strings.ToLower(node.Doc.Title)] = node
		}
	}

	return catalog, nil
}

// FindNode resolves a node query by relative path, stem, or title in O(1) time.
func (c *VaultCatalog) FindNode(query string) *ScannedNode {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	// 1. Direct match on relative path
	if n, ok := c.ByRelPath[query]; ok {
		return n
	}
	// RelPath with .md appended
	if n, ok := c.ByRelPath[query+".md"]; ok {
		return n
	}

	// 2. Direct match on Stem
	if n, ok := c.ByStem[query]; ok {
		return n
	}

	// 3. Case-insensitive stem
	lowerQuery := strings.ToLower(query)
	if n, ok := c.ByStemLower[lowerQuery]; ok {
		return n
	}

	// 4. Exact Title
	if n, ok := c.ByTitle[query]; ok {
		return n
	}

	// 5. Case-insensitive RelPath or Title
	if n, ok := c.ByRelPathLower[lowerQuery]; ok {
		return n
	}
	if n, ok := c.ByRelPathLower[lowerQuery+".md"]; ok {
		return n
	}
	if n, ok := c.ByTitleLower[lowerQuery]; ok {
		return n
	}

	return nil
}
