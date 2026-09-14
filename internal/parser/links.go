package parser

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// ExtractedLink represents a link found in the markdown body.
type ExtractedLink struct {
	Target string // Target note name or relative path (anchor stripped)
	Anchor string // Optional anchor (heading/block)
	Text   string // Label or display text
	Type   string // "wikilink" or "markdown_link"
}

var (
	// Wikilink: [[Target#Anchor|Label]] or [[Target|Label]] or [[Target#Anchor]] or [[Target]]
	wikilinkRegex = regexp.MustCompile(`\[\[([^\]\|#]+)(?:#([^\]\|]+))?(?:\|([^\]]+))?\]\]`)

	// Markdown link: [Text](path/to/file.md) or [Text](./file.md#heading)
	markdownLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+\.md(?:#[^)]*)?)\)`)
)

// ExtractLinks extracts wikilinks and markdown links from a string or reader.
func ExtractLinks(content string) []ExtractedLink {
	var links []ExtractedLink

	// 1. Wikilinks
	wikiMatches := wikilinkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range wikiMatches {
		target := strings.TrimSpace(match[1])
		anchor := strings.TrimSpace(match[2])
		label := strings.TrimSpace(match[3])
		if label == "" {
			label = target
		}

		if target != "" {
			links = append(links, ExtractedLink{
				Target: target,
				Anchor: anchor,
				Text:   label,
				Type:   "wikilink",
			})
		}
	}

	// 2. Markdown Links
	mdMatches := markdownLinkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range mdMatches {
		text := strings.TrimSpace(match[1])
		rawTarget := strings.TrimSpace(match[2])

		// Skip external links if any
		if strings.HasPrefix(rawTarget, "http://") || strings.HasPrefix(rawTarget, "https://") {
			continue
		}

		target := rawTarget
		anchor := ""
		if hashIdx := strings.Index(rawTarget, "#"); hashIdx != -1 {
			target = rawTarget[:hashIdx]
			anchor = rawTarget[hashIdx+1:]
		}

		if target != "" {
			links = append(links, ExtractedLink{
				Target: target,
				Anchor: anchor,
				Text:   text,
				Type:   "markdown_link",
			})
		}
	}

	return links
}

// ExtractBodyLinksFromFile seeks to bodyOffset and parses links from the body only.
func ExtractBodyLinksFromFile(filePath string, bodyOffset int64) ([]ExtractedLink, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer f.Close()

	if bodyOffset > 0 {
		if _, err := f.Seek(bodyOffset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek error: %w", err)
		}
	}

	bodyBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return ExtractLinks(string(bodyBytes)), nil
}
