# `okf` — High-Performance CLI for Google Open Knowledge Format (OKF v0.2)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Specification](https://img.shields.io/badge/Spec-OKF%20v0.2-blue)](IMPLEMENTATIONS.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

An ultra-fast, concurrent CLI tool written in Go to parse, query, traverse, and audit knowledge vaults adhering to the **Google Open Knowledge Format (OKF v0.2)** specification.

Designed for AI agent workflows, personal knowledge management (such as Obsidian vaults), and automated CI/CD knowledge bundle verification with sub-20ms cold scans across 1,000+ documents and minimal memory overhead.

---

## Highlights

- ⚡ **Ultra-Fast & Concurrent:** Benchmarked at **~19ms cold scan** across 1,000 Markdown documents with a memory footprint under 15MB RSS.
- 🔍 **Bounded Streaming I/O:** Never reads whole files into memory during scanning. Peeks only the initial YAML frontmatter delimiter (capped at 16KB) using buffer pooling.
- 🧩 **Polymorphic OKF v0.2 Parsing:** Seamlessly decodes dynamic metadata schema variations (polymorphic `verified` events, scalar or structured `sources`, extensible inline custom attributes).
- 🕸️ **Dual-Syntax Graph Engine:** Resolves Obsidian wikilinks (`[[Note#Heading|Alias]]`), standard relative Markdown links (`[Text](./path.md)`), and frontmatter relations (`depends_on`, `derives_from`, etc.) into a bi-directional knowledge graph.
- 🛡️ **Automated Vault Health Auditor:** Detects broken links, missing frontmatter, missing required concept types, stale documents (`stale_after`), and missing descriptions.
- 🤖 **Agent Context Injection:** Dumps the entire graph as an adjacency list in clean, machine-readable JSON for LLM retrieval and graph embeddings.

---

## Installation

### From Source

Ensure you have **Go 1.21+** installed:

```bash
git clone https://github.com/okf-tools/okf.git
cd okf
make build
```

This compiles a stripped, standalone binary (`./okf`) into the current directory.

To install globally:

```bash
go install ./cmd/okf
```

---

## Quickstart

Set your default vault location via environment variable, or pass `--vault <path>` to any command:

```bash
export OBSIDIAN_VAULT_PATH="$HOME/Documents/ObsidianVault"
```

### 1. List Documents (`okf list`)

Filter documents by concept type, status, tag, or kind:

```bash
# List all documents in table format
okf list

# Filter by concept type and status
okf list --type concept --status stable

# Filter by tag
okf list --tag distributed-systems

# Output as JSON
okf list --json
```

**Sample Output:**
```text
STEM                   KIND     TYPE      STATUS  TAGS                            PATH
----                   ----     ----      ------  ----                            ----
distributed-consensus  concept  concept   stable  distributed-systems,consensus   concepts/distributed-consensus.md
raft-protocol          concept  concept   stable  consensus,raft,fault-tolerance  concepts/raft-protocol.md
index                  index    -         stable  -                               index.md
okf-spec               concept  standard  stable  specification,okf,standard      standards/okf-spec.md
```

---

### 2. Inspect a Node (`okf get <query>`)

Fetch detailed metadata and lifecycle information by file stem, relative path, or title:

```bash
okf get "Distributed Consensus"
```

**Sample Output:**
```text
Title:       Distributed Consensus
Stem:        distributed-consensus
Path:        concepts/distributed-consensus.md
Kind:        concept
Type:        concept
Status:      stable
Description: Foundational algorithms achieving reliable agreement in distributed clusters.
Tags:        distributed-systems, consensus
Declared Relations:
  - [derives_from] -> [[raft-protocol]]
```

With `--json`, outputs the fully serialized `OKFDocument`:

```bash
okf get raft-protocol --json
```

---

### 3. Graph Neighborhood & Backlinks (`okf relations <query>`)

Inspect connections to and from a note, including declared frontmatter relations, body wikilinks, and incoming backlinks:

```bash
okf relations "Distributed Consensus"
```

**Sample Output:**
```text
Neighborhood for: distributed-consensus (concepts/distributed-consensus.md)

Declared Relations (1):
  - [derives_from] -> concepts/raft-protocol.md [RESOLVED]

Outgoing Body Links (2):
  - (wikilink) -> concepts/raft-protocol.md (text: "Raft Protocol") [RESOLVED]
  - (markdown) -> standards/okf-spec.md (text: "OKF Specification") [RESOLVED]

Incoming Backlinks (3):
  <- standards/okf-spec.md (via frontmatter [related_to])
  <- index.md (via wikilink [links_to])
  <- concepts/raft-protocol.md (via wikilink [links_to])
```

---

### 4. Vault Health Audit (`okf audit`)

Validate vault compliance with the OKF v0.2 specification:

```bash
okf audit
```

**Sample Output:**
```text
==================================================
OKF v0.2 Vault Audit Report: /path/to/vault
==================================================
Total Files Scanned: 142
Errors:              0
Warnings:            2
--------------------------------------------------
SEVERITY  FILE                      RULE                 MESSAGE
--------  ----                      ----                 -------
WARNING   concepts/legacy-cache.md  stale_concept        Concept has been stale since 2026-01-01
WARNING   standards/draft-spec.md   missing_description  Note is missing a single-sentence 'description'
--------------------------------------------------
PASSED WITH WARNINGS: 0 errors, 2 warnings.
```

**Audit Rules:**
| Rule | Severity | Description |
| :--- | :--- | :--- |
| `missing_frontmatter` | `ERROR` | File is missing opening/closing `---` YAML delimiters |
| `missing_concept_type`| `ERROR` | Document of kind `concept` is missing required `type` field |
| `dead_link` | `ERROR` | Declared relation or body link points to a non-existent note |
| `missing_description` | `WARNING` | Document does not provide a single-sentence `description` |
| `stale_concept` | `WARNING` | `stale_after` timestamp has expired relative to current time |

*Exit Code:* Returns `0` on clean pass (including warnings), and `1` if critical `ERROR` violations are found (ideal for CI pipelines).

---

### 5. Export Full Knowledge Graph (`okf export`)

Dump the entire vault graph as an adjacency list JSON for agent ingestion, RAG, or graph visualization:

```bash
okf export > knowledge_graph.json
```

**Schema Output Format:**
```json
{
  "vault_path": "/path/to/vault",
  "nodes": [
    {
      "path": "/path/to/vault/concepts/raft.md",
      "rel_path": "concepts/raft.md",
      "stem": "raft",
      "kind": "concept",
      "type": "concept",
      "title": "Raft",
      "description": "Raft consensus protocol.",
      "status": "stable",
      "tags": ["consensus", "distributed"],
      "relations": [
        { "type": "depends_on", "target": "[[paxos]]" }
      ]
    }
  ],
  "edges": [
    {
      "source": "concepts/raft.md",
      "target": "concepts/paxos.md",
      "raw_target": "[[paxos]]",
      "type": "depends_on",
      "syntax": "frontmatter",
      "resolved": true
    }
  ]
}
```

---

## OKF v0.2 Document Format

An OKF v0.2 document consists of YAML frontmatter followed by a standard Markdown body:

```markdown
---
type: concept
title: "Distributed Consensus"
description: "Core algorithms for distributed agreement across nodes."
status: stable # draft | stable | deprecated
tags:
  - distributed-systems
  - consensus
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
stale_after: "2027-01-01T00:00:00Z"
custom_key: "custom_value" # Extensible attributes
---

# Distributed Consensus

Content with Obsidian [[Wikilinks]] or [Markdown Links](other.md).
```

### Document Kinds
- `concept` (`KindConcept`): Default knowledge notes. Requires `type` metadata.
- `index` (`KindIndex`): Table-of-contents or dashboards (`index.md`, `* Dashboard.md`).
- `log` (`KindLog`): Event logs, session notes, or changelogs (`log.md`, `changelog.md`).

---

## 🤖 Agent Skills & Harness Integration

This repository includes the official **Google OKF Agent Skill** under [`skills/`](skills/), packaging prompt guidelines, templates, references, and CLI hooks for autonomous agents:

- **Packaged Bundle:** [`skills/google-okf.skill`](skills/google-okf.skill)
- **Extracted Skill:** [`skills/google-okf/`](skills/google-okf/)
- **Comprehensive Setup Guide:** [`skills/README.md`](skills/README.md)

### Quick Setup by Harness

- **Hermes Agent:** `hermes skill install skills/google-okf.skill`
- **OpenClaw:** `openclaw skill install skills/google-okf.skill`
- **Claude Code:** Add instructions pointing to `skills/google-okf/SKILL.md` in your project `CLAUDE.md`.
- **OpenAI Codex / Agents SDK:** Use `skills/google-okf/SKILL.md` in agent instructions and bind `okf` CLI.
- **Antigravity / Gemini CLI:** Automatically discovered at `skills/google-okf/SKILL.md`.
- **Cursor / Windsurf:** Add project rule pointing to `skills/google-okf/SKILL.md` in `.cursorrules`.

For full step-by-step setup guides, see [**`skills/README.md`**](skills/README.md).

---

## Benchmarks

Tested on Apple M1 (8 cores) across a synthetic vault of 1,000 Markdown documents:

| Benchmark Operation | Latency | Memory Allocs |
| :--- | :--- | :--- |
| `ScanVault` (1,000 files) | **~19.6 ms** | ~18 MB |
| `BuildGraph` + Full Link Extraction (1,000 files) | **~31.8 ms** | ~23 MB |

Run benchmarks locally:

```bash
make bench
```

---

## Development

```bash
# Run all unit and integration tests
make test

# Build optimized binary
make build

# Clean up binaries
make clean
```

---

## License

This project is licensed under the Apache 2.0 License.
