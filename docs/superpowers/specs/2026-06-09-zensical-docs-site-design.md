# Zensical Documentation Site

**Status:** approved
**Date:** 2026-06-09

## Goal

Transform the existing `docs/` directory into a Zensical static site, using `uv` for Python project management, and deploy it to GitHub Pages via GitHub Actions.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Hosting | GitHub Pages | Free, built-in, zero infra |
| Python root | `docs/` | Keeps tooling self-contained, `cd docs` to preview |
| Branding | iFood (moderate) | Logo, colors, Montserrat font |
| Navigation | Flat sidebar | Matches current README order, no need for grouping |
| Scope | Exclude `superpowers/` | Internal process docs, not for public consumption |
| Features | Minimal (no search, no social cards) | YAGNI — add later if needed |

## Project Structure

All new files live inside `docs/`. No changes to the Go project root.

```
docs/
├── pyproject.toml            # uv project with zensical as dev dependency
├── uv.lock                   # lockfile (uv generates)
├── zensical.toml             # Zensical config
├── assets/
│   ├── ifood_logo.png        # red logo (light theme)
│   └── ifood_logo_white.png  # white logo (dark theme / header)
├── index.md                  # site home page (new)
├── getting-started.md        # existing, unchanged
├── architecture.md           # existing, unchanged
├── entities.md               # existing, unchanged
├── redaction.md              # existing, unchanged
├── batch.md                  # existing, unchanged
├── anonymize.md              # existing, unchanged
├── configuration.md          # existing, unchanged
├── deployment.md             # existing, unchanged
├── plugins.md                # existing, unchanged
├── observability.md          # existing, unchanged
├── errors.md                 # existing, unchanged
├── content-negotiation.md    # existing, unchanged
├── openapi.yaml              # existing, unchanged
├── examples/                 # existing, unchanged
└── superpowers/              # excluded from build
```

Also created: `.github/workflows/docs.yml` at the repo root.

## Zensical Configuration

`docs/zensical.toml`:

```toml
[project]
site_name = "Anonymizer"
site_url = "https://Prosus-Cyber-Xchange.github.io/anonymizer"

[theme]
font = { text = "Montserrat", code = "JetBrains Mono" }
primary = "#EA1D2C"
accent = "#EA1D2C"
background = "#F5F0EB"

[theme.logo]
light = "assets/ifood_logo.png"
dark  = "assets/ifood_logo_white.png"

[build]
exclude = ["superpowers/**"]

[nav]
flat = [
  "index.md",
  "getting-started.md",
  "entities.md",
  "redaction.md",
  "batch.md",
  "anonymize.md",
  "content-negotiation.md",
  "architecture.md",
  "configuration.md",
  "deployment.md",
  "plugins.md",
  "observability.md",
  "errors.md",
]
```

- `site_url` must be updated with the actual GitHub org/user name before first deploy
- iFood brand colors: primary `#EA1D2C` (red), background `#F5F0EB` (cream)
- Montserrat for body text, JetBrains Mono for code blocks
- `superpowers/` is excluded from the build

## New Files

### docs/index.md

A short landing page with project name, one-line description, and links to Getting Started and API docs. No duplicate of README content — serves as the site homepage.

### docs/assets/ifood_logo.png

Copied from `~/.agents/skills/ifood-slides/assets/ifood_logo.png` (red logo on white background).

### docs/assets/ifood_logo_white.png

Copied from `~/.agents/skills/ifood-slides/assets/ifood_logo_white.png` (white logo on transparent).

## GitHub Actions Workflow

`.github/workflows/docs.yml`:

```yaml
name: Documentation

on:
  push:
    branches: [main]

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/configure-pages@v5
      - uses: actions/checkout@v5
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - uses: astral-sh/setup-uv@v5
      - name: Build site
        run: uv run zensical build --clean
        working-directory: docs
      - uses: actions/upload-pages-artifact@v4
        with:
          path: docs/site
      - uses: actions/deploy-pages@v4
        id: deployment
```

- Triggers on push to `main`
- Uses `astral-sh/setup-uv` to install uv
- Runs `uv run zensical build --clean` inside `docs/`
- Uploads `docs/site/` (Zensical default output dir) as Pages artifact
- Deploys via `deploy-pages`

## Non-Goals

- No search plugin
- No social card metadata
- No PR preview builds
- No Dark Mode toggle (Zensical default auto-detection only)
- No changes to existing Markdown content

## Implementation Notes

1. After first deploy, verify GitHub Pages is enabled in repo Settings
2. Logos must be copied from `~/.agents/skills/ifood-slides/assets/` — the skill path may differ per machine
