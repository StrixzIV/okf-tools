# Google OKF Agent Skill (`google-okf`)

This directory contains the official **Google Open Knowledge Format (OKF v0.2)** agent skill. It equips autonomous coding agents, LLM harnesses, and research bots with the instructions, templates, reference schemas, and tooling needed to parse, create, audit, and traverse human- and agent-friendly knowledge vaults.

---

## 📦 Directory Structure

```text
skills/
├── google-okf.skill            # Pre-packaged distributable skill archive (zip)
├── README.md                   # This integration and setup guide
└── google-okf/                 # Extracted skill directory
    ├── SKILL.md                # Standardized Agent Skill definition & rules
    ├── references/
    │   └── okf_v02_spec.md     # Google OKF v0.2 quick specification reference
    ├── templates/
    │   └── concept.md          # Standard concept note template with OKF frontmatter
    └── scripts/
        └── okf_graph.py        # Python fallback query & audit utility
```

---

## ⚙️ Setup Instructions by Agent Harness

Select your agent framework or harness below:

### 1. 🪽 Hermes Agent (NousResearch)

Hermes supports the standardized `.skill` format and scans both global and workspace skill directories.

#### Option A: Install via Hermes CLI
Run in your terminal:
```bash
# Install directly from the packaged .skill bundle
hermes skill install skills/google-okf.skill

# Or install from directory
hermes skill install skills/google-okf
```

#### Option B: Manual Installation
Copy the unpacked skill folder to your Hermes skills directory:

- **Global (All Projects):**
  ```bash
  mkdir -p ~/.hermes/skills
  cp -r skills/google-okf ~/.hermes/skills/
  ```

- **Workspace / Project-Level:**
  ```bash
  mkdir -p .hermes/skills
  cp -r skills/google-okf .hermes/skills/
  ```

#### Environment Configuration
Ensure your vault path is configured in your shell or Hermes environment:
```bash
export OBSIDIAN_VAULT_PATH="$HOME/Documents/ObsidianVault"
```

Hermes reads `SKILL.md` metadata frontmatter (`tags: [okf, google-okf, obsidian, knowledge-graph]`) and will automatically invoke `okf` CLI (or the Python fallback `okf_graph.py`) when interacting with your vault.

---

### 2. 🦞 OpenClaw

OpenClaw discovers skills defined with standard `SKILL.md` manifests.

#### Option A: Install via OpenClaw CLI
```bash
openclaw skill install ./skills/google-okf.skill
# Or add directory
openclaw skill add ./skills/google-okf
```

#### Option B: Configure `openclaw.json` / `config.yaml`
Add the repository's `skills/` path to your OpenClaw configuration:

```json
{
  "skills": {
    "paths": [
      "./skills/google-okf",
      "~/.openclaw/skills"
    ]
  }
}
```

#### Option C: Copy to OpenClaw Global Skills
```bash
mkdir -p ~/.openclaw/skills
cp -r skills/google-okf ~/.openclaw/skills/
```

---

### 3. 🤖 Claude Code (Anthropic)

Claude Code detects instructions placed in `CLAUDE.md` or `.claude/` directories.

#### Option A: Integrate via `CLAUDE.md` (Recommended)
Add the following block to your project's `CLAUDE.md` (or your user-level `~/.claude/CLAUDE.md`):

```markdown
## Knowledge Graph & OKF Skill

When reading, updating, creating, or auditing notes in an OKF/Obsidian vault:
- Consult the Google OKF v0.2 skill: `skills/google-okf/SKILL.md`
- Reference specification: `skills/google-okf/references/okf_v02_spec.md`
- Concept templates: `skills/google-okf/templates/concept.md`
- High-speed queries: Use `okf` CLI (`okf audit`, `okf list`, `okf get`, `okf relations`, `okf export`)
- Fallback script: `python3 skills/google-okf/scripts/okf_graph.py`
```

#### Option B: Slash Command / Custom Action

Create `.claude/commands/okf.md`:

Run vault audit and health check using okf-tools:

```bash
okf audit --vault "${OBSIDIAN_VAULT_PATH:-.}"
```

#### Option C: Global Claude Directory

```bash
mkdir -p ~/.claude/skills
cp -r skills/google-okf ~/.claude/skills/
```

---

### 4. 🧠 OpenAI Codex / Agents SDK / Custom Harness

For frameworks using OpenAI Agents SDK, Swarm, LangChain, or Codex CLI:

#### OpenAI Agents SDK
Register the skill definition and link the `okf` executable tool:

```python
from agents import Agent, Runner

with open("skills/google-okf/SKILL.md", "r") as f:
    okf_instructions = f.read()

agent = Agent(
    name="VaultCurator",
    instructions=f"You are a knowledge curator adhering to Google OKF v0.2:\n\n{okf_instructions}",
    tools=[
        # Register shell execution tool for `okf` CLI or `okf_graph.py`
    ]
)
```

#### Project-level Agent Instructions
Many modern agent harnesses look for `.agents/skills`:
```bash
mkdir -p .agents/skills
cp -r skills/google-okf .agents/skills/
```

#### ChatGPT Custom GPT
1. In the **Instructions** box of your Custom GPT, paste the contents of `skills/google-okf/SKILL.md`.
2. Under **Knowledge**, upload `references/okf_v02_spec.md` and `templates/concept.md`.

---

### 5. 🪐 Google Antigravity & Gemini CLI

Antigravity natively scans skills folders in both workspace and user configuration roots.

#### Workspace (Instant Discovery)
The skill is already located at:
```text
skills/google-okf/SKILL.md
```
Antigravity automatically discovers skills inside the workspace.

#### Global Installation (Across All Repositories)
To make this skill available across all Antigravity projects:
```bash
mkdir -p ~/.gemini/antigravity/skills
cp -r skills/google-okf ~/.gemini/antigravity/skills/
```

---

### 6. 💻 Cursor & Windsurf

#### Cursor Rules
Add an `.cursorrules` or `.cursor/rules/okf.mdc` file to your vault repository:

```markdown
---
description: Google OKF Knowledge Graph Guidelines
globs: ["**/*.md"]
alwaysApply: false
---

Follow Google OKF v0.2 specifications from `skills/google-okf/SKILL.md`:
1. All concept notes MUST contain YAML frontmatter with `type: concept`, `title`, `description`, `status`, and `relations`.
2. Never create generic `Index.md` in subdirectories—use `<Topic> Dashboard.md`.
3. Use `okf audit` or `okf list` to inspect vault graph consistency.
```

---

## 🛠️ Tooling Cheat Sheet for Agents

When an agent needs to manipulate an OKF vault:

| Action | Native Go CLI (`okf`) | Python Fallback (`okf_graph.py`) |
| :--- | :--- | :--- |
| **Audit Compliance** | `okf audit --vault <path>` | `python3 okf_graph.py --vault <path> audit` |
| **List Documents** | `okf list --type concept` | `python3 okf_graph.py --vault <path> list` |
| **Inspect Note** | `okf get "<Note Name>"` | `python3 okf_graph.py --vault <path> get "<Note Name>"` |
| **Traverse Relations** | `okf relations "<Note Name>"` | (Inspect frontmatter `relations`) |
| **Export Graph** | `okf export --json` | (JSON adjacency list) |
