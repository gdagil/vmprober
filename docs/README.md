<p align="center">
  <img src="assets/banner.svg" alt="VMProber" width="600">
</p>

# VMProber Documentation

This directory contains the documentation for VMProber, built with Jekyll for GitHub Pages.

## Structure

```
docs/
├── _config.yml              # Jekyll configuration
├── _layouts/
│   └── default.html         # Main layout template
├── assets/                   # Logo and brand assets
│   ├── logo.svg             # Main logo
│   ├── banner.svg           # Banner for headers
│   ├── favicon.svg          # Favicon
│   └── logo-inline.svg      # Inline logo for footers
├── index.md                 # Documentation home
├── getting-started/         # Installation and setup guides
│   ├── installation.md
│   ├── quick-start.md
│   ├── configuration.md
│   └── basic-usage.md
├── architecture/            # System design docs
│   ├── overview.md
│   └── design-principles.md
├── components/              # Component documentation
│   └── probes.md
├── operations/              # Deployment and ops guides
│   ├── deployment.md
│   ├── docker.md
│   └── troubleshooting.md
├── development/             # Developer guides
│   ├── setup.md
│   └── e2e-testing.md
├── reference/               # API and metrics reference
│   ├── api.md
│   └── metrics.md
└── guides/                  # How-to guides
    └── monitoring-setup.md
```

## Local Development

### Prerequisites

- Ruby 2.7+
- Bundler

### Setup

```bash
cd docs

# Install dependencies
bundle install

# Start local server
bundle exec jekyll serve
```

Visit `http://localhost:4000` to view the documentation.

### Using Docker

```bash
docker run --rm -v "$PWD:/srv/jekyll" -p 4000:4000 jekyll/jekyll jekyll serve
```

## Building for Production

```bash
cd docs
bundle exec jekyll build
```

Output is generated in the `_site` directory.

## GitHub Pages

This documentation is automatically deployed to GitHub Pages when changes are pushed to the `main` branch.

**Live URL**: https://gdagil.github.io/vmprober/

## Writing Documentation

### Frontmatter

Each Markdown file should include frontmatter:

```yaml
---
layout: default
title: Page Title
---
```

### Links

Use relative links for internal documentation:

```markdown
See [Configuration Guide](getting-started/configuration.md) for details.
```

### Code Blocks

Use fenced code blocks with language hints:

````markdown
```yaml
listen:
  address: ":8429"
```
````

### Tables

Use GitHub-flavored Markdown tables:

```markdown
| Column 1 | Column 2 |
|----------|----------|
| Value 1  | Value 2  |
```

## Brand Assets

The documentation uses consistent branding:

- **Primary Color**: `#00D4AA` (VMProber Green)
- **Dark Background**: `#1a1a2e`
- **Font**: Plus Jakarta Sans (headings), JetBrains Mono (code)

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines.

---

<p align="center">
  <img src="assets/logo-inline.svg" alt="VMProber" width="100">
</p>
