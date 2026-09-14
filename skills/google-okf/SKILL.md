---
name: google-okf
description: Standardized workflows, vault structuring, and CLI tooling for Google Open Knowledge Format (OKF v0.2) knowledge graphs.
category: note-taking
version: 1.0.0
author: Ceres Gingashi (銀河獅セレス) & StrixzIV
license: MIT
platforms: [linux, macos, windows]
metadata:
  hermes:
    tags: [okf, google-okf, obsidian, knowledge-graph, taxonomy, indexing, go]
    related_skills: [obsidian, vault-taxonomy-and-indexing]
---

# Google Open Knowledge Format (OKF v0.2) Skill

This skill provides complete guidelines, directory structures, and tooling for creating, maintaining, and querying knowledge graphs conforming to the **Google Open Knowledge Format (OKF v0.2)** specification inside Obsidian vaults.

---

## 🌟 What is Google OKF?

Google OKF is an open, vendor-neutral specification for representing human- and agent-friendly knowledge. It packages knowledge as a directory of Markdown files with structured YAML frontmatter, versioned via Git.

### Core OKF Frontmatter Schema
```yaml
---
type: concept            # REQUIRED: concept, subject, cursus-hub, standard, delta-report, dashboard, person, etc.
title: Note Title        # Human-readable title
description: Single sentence summary of what this note covers.
status: stable           # draft | stable | deprecated
tags: [tag1, tag2]
generated:
  by: agent:ceres-gingashi/gemini-3.8-flash
  at: 2026-09-14T00:00:00Z
verified:
  by: human:strixziv
  at: 2026-09-14T00:00:00Z
relations:
  - type: depends_on     # depends_on | derives_from | related_to | supersedes
    target: "[[Target Node]]"
---
```

---

## 🏛️ Vault Structure & Layout Rules

1. **Root Entrypoint:** Place an `Index.md` at the vault root with `type: index`.
2. **Sub-Dashboards:** Always use descriptive names like `<Topic> Dashboard.md` (never generic `Index.md` or `README.md`) to prevent global search collisions in Obsidian.
3. **Domain Subdirectories:**
   - `Subjects/`: Project and curriculum specifications.
   - `Concepts/`: High-level thematic hubs.
   - `Standards/`: Global code quality, syntax, and validator standards.
   - `Remarks/`: Unofficial commentary, tips, and design notes.
   - `Historical_Deltas/`: Evolution and delta tracking reports.

---

## ⚡ High-Speed Tooling: `okf-tools` (Go CLI)

For sub-10ms queries, use the native Go binary `okf` from `https://github.com/StrixzIV/okf-tools.git`.

### Installation & Build:
```bash
git clone https://github.com/StrixzIV/okf-tools.git
cd okf-tools
make build
cp okf ~/.local/bin/okf
```

### Command Reference:
- `okf audit --vault <path>`: Scans for 100% OKF compliance, verifying frontmatter, required `type` field, and dead links.
- `okf list --type <type>`: Lists matching nodes by type, status, or tag.
- `okf get "<Note Name>"`: Fetches note metadata, description, and custom attributes.
- `okf relations "<Note Name>"`: Inspects outgoing relations and incoming backlinks.
- `okf export`: Dumps the complete vault graph as an adjacency list JSON.
