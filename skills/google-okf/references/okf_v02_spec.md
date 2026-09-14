# Google Open Knowledge Format (OKF) v0.2 Quick Reference

- **Standard:** Minimal directory of Markdown files + YAML frontmatter.
- **Reserved Files:**
  - `index.md`: Directory listing for progressive disclosure.
  - `log.md`: Chronological history of updates.
- **Required Frontmatter:**
  - `type`: String identifying concept kind.
- **Recommended Frontmatter:**
  - `title`: Display name.
  - `description`: Single sentence summary.
  - `status`: `draft` | `stable` | `deprecated`.
  - `tags`: List of strings.
  - `generated`: `{ by: <actor>, at: <ISO 8601> }`.
  - `verified`: `{ by: <actor>, at: <ISO 8601> }`.
  - `relations`: `[{ type: <relation_type>, target: <path_or_wikilink> }]`.
- **Actor Convention:**
  - `agent:<name>/<version>` or `<producer>/<version>`
  - `human:<id>`
  - `process:<id>`
