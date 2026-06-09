# Zensical Documentation Site Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the existing `docs/` directory into a Zensical static site with iFood branding and deploy to GitHub Pages via GitHub Actions.

**Architecture:** Python/uv project inside `docs/` with Zensical as a dev dependency. A single `zensical.toml` configures the theme (iFood colors, Montserrat font, logos), flat sidebar navigation, and excludes internal `superpowers/` content. A GitHub Actions workflow builds the site on push to `main` and deploys to GitHub Pages.

**Tech Stack:** Python 3.12, uv (project management), Zensical (static site generator), GitHub Actions + GitHub Pages (CI/CD + hosting)

---

### Task 1: Copy logo assets into docs

**Files:**
- Create: `docs/assets/ifood_logo.png`
- Create: `docs/assets/ifood_logo_white.png`

- [ ] **Step 1: Create the assets directory**

```bash
mkdir -p docs/assets
```

- [ ] **Step 2: Copy the red logo (light theme)**

```bash
cp ~/.agents/skills/ifood-slides/assets/ifood_logo.png docs/assets/ifood_logo.png
```

- [ ] **Step 3: Copy the white logo (dark theme / header)**

```bash
cp ~/.agents/skills/ifood-slides/assets/ifood_logo_white.png docs/assets/ifood_logo_white.png
```

- [ ] **Step 4: Verify logos were copied**

```bash
ls -la docs/assets/ifood_logo.png docs/assets/ifood_logo_white.png
```

Expected: both files exist and are non-zero sized (red ~49KB, white ~8KB).

- [ ] **Step 5: Commit**

```bash
git add docs/assets/
git commit -m "docs: add iFood logo assets for Zensical site"
```

---

### Task 2: Create pyproject.toml for uv/Zensical

**Files:**
- Create: `docs/pyproject.toml`

- [ ] **Step 1: Write the pyproject.toml**

```toml
[project]
name = "anonymizer-docs"
version = "0.1.0"
requires-python = ">=3.12"

[dependency-groups]
dev = [
    "zensical",
]
```

- [ ] **Step 2: Generate the lockfile with uv**

Run from the `docs/` directory:

```bash
uv lock
```

Expected: creates `docs/uv.lock` with resolved dependencies including zensical. No errors.

This command reads `pyproject.toml`, resolves all dependencies, and writes `docs/uv.lock`. The lockfile is auto-generated JSON — do not edit it by hand.

- [ ] **Step 3: Verify lockfile was created**

```bash
ls -la docs/uv.lock
```

Expected: file exists.

- [ ] **Step 4: Commit**

```bash
git add docs/pyproject.toml docs/uv.lock
git commit -m "docs: add uv project files with zensical dependency"
```

---

### Task 3: Create Zensical configuration

**Files:**
- Create: `docs/zensical.toml`

- [ ] **Step 1: Write zensical.toml**

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

- [ ] **Step 2: Verify Zensical can read the config**

```bash
uv run zensical --help
```

Expected: Zensical CLI help output, no config errors. This confirms zensical is installed and executable.

- [ ] **Step 3: Commit**

```bash
git add docs/zensical.toml
git commit -m "docs: add zensical.toml with iFood brand theme and flat nav"
```

---

### Task 4: Create site home page (index.md)

**Files:**
- Create: `docs/index.md`

- [ ] **Step 1: Write index.md**

```markdown
# Anonymizer

A high-performance PII anonymization service built for AI prompt workloads.

## What it does

Detects and anonymizes personally identifiable information (PII) — emails, CPF/CNPJ numbers, IP addresses, credit cards, phone numbers, and more — using byte-level processing for minimal latency and maximal throughput.

## Highlights

- **Inline privacy rules** — supply anonymization settings directly in the request body
- **Batch processing** — anonymize multiple texts in a single request
- **Redaction and masking** — replace PII entirely or partially mask it
- **Plugin system** — inject custom middleware at compile time
- **Embeddable library** — import as a Go package in any application

## Where to start

[Getting Started](getting-started.md){ .md-button }
[Entity Reference](entities.md){ .md-button }
[API Specification](openapi.yaml){ .md-button }
```

- [ ] **Step 2: Commit**

```bash
git add docs/index.md
git commit -m "docs: add Zensical site home page"
```

---

### Task 5: Create GitHub Actions workflow

**Files:**
- Create: `.github/workflows/docs.yml`

- [ ] **Step 1: Create workflows directory**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 2: Write the workflow file**

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

- [ ] **Step 3: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/docs.yml')); print('Valid YAML')"
```

Expected: `Valid YAML`.

- [ ] **Step 4: Verify GitHub Actions can parse the workflow**

```bash
python3 -c "
import yaml
w = yaml.safe_load(open('.github/workflows/docs.yml'))
assert w['name'] == 'Documentation'
assert w['on']['push']['branches'] == ['main']
assert len(w['jobs']['deploy']['steps']) == 8
print('Workflow structure valid')
"
```

Expected: `Workflow structure valid`.

Note: The `pyyaml` package must be available (`python3 -m pip install pyyaml` if needed). If `pyyaml` is not installed, this validation step can be skipped — the YAML is syntactically simple and GitHub Actions itself will validate on push.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/docs.yml
git commit -m "ci: add GitHub Actions workflow for Zensical docs deployment"
```

---

### Task 6: Verify local build works end-to-end

**Files:**
- No new files — verification only

- [ ] **Step 1: Run a clean Zensical build**

```bash
uv run zensical build --clean
```

Expected: build completes without errors. Output shows files written to `docs/site/`.

- [ ] **Step 2: Verify the output site directory exists and has content**

```bash
ls docs/site/
```

Expected: directory listing includes `index.html`, `404.html`, `assets/`, and at minimum one page HTML file (e.g., `getting-started/index.html`).

- [ ] **Step 3: Verify superpowers is excluded from output**

```bash
ls docs/site/superpowers/ 2>&1
```

Expected: `ls: docs/site/superpowers/: No such file or directory` (directory does not exist in build output).

- [ ] **Step 4: Verify index.html was generated**

```bash
head -5 docs/site/index.html
```

Expected: valid HTML output, not a 404 or error page.

- [ ] **Step 5: Commit the .gitignore (if site output should not be tracked)**

Check if `docs/site` is already gitignored or needs to be:

```bash
git check-ignore docs/site/ 2>&1 || echo "NOT IGNORED"
```

If `NOT IGNORED`, add it:

```bash
echo "docs/site/" >> .gitignore
git add .gitignore
git commit -m "chore: gitignore Zensical build output (docs/site/)"
```

---

### Task 7: Final review and post-deploy notes

- [ ] **Step 1: Review git log of all changes**

```bash
git log --oneline -8
```

Expected: 6-7 commits covering logos, pyproject, zensical config, index, workflow, verification.

- [ ] **Step 2: Verify no unexpected files are staged**

```bash
git status
```

Expected: clean working tree.

---

## Post-Deploy Manual Steps

These must be done by a human with repo admin access after the first push to `main`:

1. Go to repo **Settings > Pages** and confirm GitHub Pages is set to deploy from **GitHub Actions** (not a branch)
2. Verify the workflow ran successfully at **Actions > Documentation**
3. Visit `https://Prosus-Cyber-Xchange.github.io/anonymizer` and confirm the site loads with iFood branding
