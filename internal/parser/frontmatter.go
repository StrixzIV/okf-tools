package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"okf/pkg/okf"

	"gopkg.in/yaml.v3"
)

const (
	DefaultMaxFrontmatterBytes = 16 * 1024 // 16 KB bounded limit
)

var (
	ErrMissingFrontmatter      = errors.New("missing YAML frontmatter")
	ErrUnterminatedFrontmatter = errors.New("unterminated YAML frontmatter within limit")
)

var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, DefaultMaxFrontmatterBytes)
		return &b
	},
}

// PeekResult holds the parsed document and body byte offset.
type PeekResult struct {
	Doc        *okf.OKFDocument
	BodyOffset int64
}

// PeekFrontmatter reads only up to the closing frontmatter delimiter (or maxBytes cap).
// It performs a single bounded read without reading the whole file into memory.
func PeekFrontmatter(filePath string, maxBytes int64) (*PeekResult, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrontmatterBytes
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	bufPtr := bufPool.Get().(*[]byte)
	buf := *bufPtr
	defer bufPool.Put(bufPtr)

	readCap := int(maxBytes)
	if readCap > len(buf) {
		buf = make([]byte, readCap)
	}

	n, err := io.ReadFull(f, buf[:readCap])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("read error: %w", err)
	}
	if n == 0 {
		return nil, ErrMissingFrontmatter
	}

	rawContent := buf[:n]

	// Strip BOM if present and track offset
	bomOffset := 0
	content := rawContent
	if bytes.HasPrefix(rawContent, []byte("\xef\xbb\xbf")) {
		bomOffset = 3
		content = rawContent[3:]
	}

	// Check opening ---
	firstLineEnd := bytes.IndexByte(content, '\n')
	if firstLineEnd == -1 {
		return nil, ErrMissingFrontmatter
	}
	firstLine := bytes.TrimSpace(content[:firstLineEnd])
	if !bytes.Equal(firstLine, []byte("---")) {
		return nil, ErrMissingFrontmatter
	}

	// Scan lines for closing --- or ...
	idx := firstLineEnd + 1
	var closingStart = -1
	var bodyOffset int64 = -1

	for idx < len(content) {
		lineEnd := bytes.IndexByte(content[idx:], '\n')
		var line []byte
		var nextIdx int
		if lineEnd == -1 {
			line = content[idx:]
			nextIdx = len(content)
		} else {
			line = content[idx : idx+lineEnd]
			nextIdx = idx + lineEnd + 1
		}

		trimmed := bytes.TrimSpace(line)
		if bytes.Equal(trimmed, []byte("---")) || bytes.Equal(trimmed, []byte("...")) {
			closingStart = idx
			bodyOffset = int64(bomOffset + nextIdx)
			break
		}

		idx = nextIdx
	}

	if closingStart == -1 {
		return nil, ErrUnterminatedFrontmatter
	}

	fmBytes := content[firstLineEnd+1 : closingStart]

	var doc okf.OKFDocument
	if err := yaml.Unmarshal(fmBytes, &doc); err != nil {
		return nil, fmt.Errorf("YAML unmarshal error: %w", err)
	}

	doc.Path = filePath

	return &PeekResult{
		Doc:        &doc,
		BodyOffset: bodyOffset,
	}, nil
}
