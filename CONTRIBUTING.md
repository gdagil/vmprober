<p align="center">
  <img src="assets/logo.svg" alt="VMProber Logo" width="120">
</p>

<h1 align="center">Contributing to VMProber</h1>

<p align="center">
  Thank you for your interest in contributing to VMProber! We welcome contributions from the community.
</p>

---

## Table of Contents

- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Pull Requests](#pull-requests)
- [Code Standards](#code-standards)
- [Development Process](#development-process)
- [Areas for Contribution](#areas-for-contribution)
- [Code of Conduct](#code-of-conduct)

## How to Contribute

### Reporting Bugs

If you found a bug:

1. **Check existing issues** - Search [Issues](https://github.com/gdagil/vmprober/issues) to avoid duplicates
2. **Create a new issue** with the following information:
   - Clear description of the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details:
     - VMProber version
     - Go version
     - Operating system
   - Relevant logs and configuration (without secrets)

**Bug Report Template:**

```markdown
## Description
[Clear description of the bug]

## Steps to Reproduce
1. Step one
2. Step two
3. ...

## Expected Behavior
[What you expected to happen]

## Actual Behavior
[What actually happened]

## Environment
- VMProber Version:
- Go Version:
- OS:
- Docker Version (if applicable):

## Logs
```
[Relevant log output]
```

## Configuration
```yaml
[Relevant configuration (remove secrets)]
```
```

### Suggesting Enhancements

For new feature suggestions:

1. Check existing issues and discussions
2. Create an issue describing:
   - The problem the feature solves
   - Proposed solution
   - Alternatives considered (if any)
   - Potential impact on existing functionality

### Pull Requests

1. **Fork the repository**
2. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```
3. **Make changes** following [code standards](#code-standards)
4. **Add tests** for new features
5. **Update documentation** if needed
6. **Ensure all tests pass**:
   ```bash
   make test
   make lint
   ```
7. **Create a Pull Request** with a clear description

## Code Standards

### Project Structure

```
vmprober/
├── cmd/vmprober/          # Application entry point
├── internal/              # Private application code
│   ├── adapter/           # External system adapters
│   ├── config/            # Configuration management
│   ├── metrics/           # Metrics collection
│   ├── normalizer/        # Data normalization
│   ├── observability/     # Logging, tracing
│   ├── probe/             # Probe implementations
│   ├── scheduler/         # Task scheduling
│   ├── server/            # HTTP server
│   ├── shutdown/          # Graceful shutdown
│   ├── types/             # Common types
│   └── wal/               # Write-Ahead Log
├── pkg/interfaces/        # Public interfaces
├── config/                # Configuration files
├── docs/                  # Documentation
└── assets/                # Brand assets
```

### Formatting

Use standard Go tools:

```bash
make fmt
```

This runs:
- `gofmt` - Standard Go formatting
- `goimports` - Import organization

### Linting

Code must pass linting:

```bash
make lint
```

We use `golangci-lint` with the following enabled linters:
- `errcheck` - Error handling
- `gosimple` - Code simplification
- `govet` - Go vet checks
- `staticcheck` - Static analysis
- `unused` - Unused code detection

### Testing

- All new features **must have tests**
- Minimum code coverage: **80%**
- Run tests:
  ```bash
  # Unit tests
  make test

  # Tests with coverage
  make test-coverage

  # E2E tests (requires Docker)
  make e2e-test
  ```

### Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Examples:**

```
feat(probe): add HTTP probe support

Implements HTTP/HTTPS probe with configurable headers,
method, and expected status codes.

Closes #123
```

```
fix(scheduler): resolve race condition in job execution

Add mutex lock around concurrent job map access.
```

**Commit Types:**

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `test` | Adding or modifying tests |
| `refactor` | Code refactoring (no feature change) |
| `perf` | Performance improvements |
| `chore` | Maintenance (dependencies, config) |
| `style` | Formatting (no logic changes) |
| `ci` | CI/CD changes |

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| **Packages** | lowercase, single word | `probe`, `config` |
| **Public Functions** | PascalCase | `NewCollector()` |
| **Private Functions** | camelCase | `parseConfig()` |
| **Public Variables** | PascalCase | `DefaultTimeout` |
| **Private Variables** | camelCase | `internalState` |
| **Constants** | PascalCase | `MaxRetries` |
| **Interfaces** | PascalCase, often ending in `-er` | `Prober`, `Collector` |

### Documentation

All public APIs must have documentation comments:

```go
// Collector collects and exports Prometheus metrics.
// It supports both counter and histogram metric types.
type Collector struct {
    namespace string
    registry  *prometheus.Registry
}

// NewCollector creates a new metrics collector with the given namespace.
// The namespace is used as a prefix for all metric names.
func NewCollector(namespace string) *Collector {
    return &Collector{
        namespace: namespace,
        registry:  prometheus.NewRegistry(),
    }
}
```

### Error Handling

Always handle errors explicitly:

```go
// Good
result, err := someFunction()
if err != nil {
    return nil, fmt.Errorf("failed to process: %w", err)
}

// Bad - ignoring errors
result, _ := someFunction()
```

Use error wrapping for context:

```go
if err := config.Load(path); err != nil {
    return fmt.Errorf("loading config from %s: %w", path, err)
}
```

## Development Process

### Setting Up Environment

1. **Fork the repository** on GitHub

2. **Clone your fork**:
   ```bash
   git clone https://github.com/your-username/vmprober.git
   cd vmprober
   ```

3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/gdagil/vmprober.git
   ```

4. **Install dependencies**:
   ```bash
   make deps
   ```

5. **Verify setup**:
   ```bash
   make build
   make test
   ```

### Development Workflow

1. **Sync with upstream**:
   ```bash
   git checkout main
   git fetch upstream
   git merge upstream/main
   ```

2. **Create feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make changes** with frequent commits

4. **Run checks before pushing**:
   ```bash
   make fmt
   make lint
   make test
   ```

5. **Push changes**:
   ```bash
   git push origin feature/your-feature-name
   ```

### Running Locally with Docker

```bash
# Start the monitoring stack
docker-compose up -d vmstorage vminsert vmselect grafana

# Run VMProber locally
go run ./cmd/vmprober --config=config/vmprober/config.yaml.example

# Access services
# - VMProber Dashboard: http://localhost:8429
# - Grafana: http://localhost:3000 (admin/admin)
# - VictoriaMetrics: http://localhost:8481
```

### Creating Pull Request

1. Go to GitHub and create a Pull Request
2. Fill out the PR template:
   - Description of changes
   - Related issues
   - Testing performed
   - Checklist
3. Request review from maintainers
4. Address feedback and update as needed

### PR Checklist

- [ ] Code follows project style guidelines
- [ ] Tests added for new functionality
- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventional commits
- [ ] PR description is clear and complete

## Areas for Contribution

### 🔌 Probes

- HTTP/HTTPS probe implementation
- DNS probe implementation
- gRPC probe implementation
- Custom probe interface improvements

### ⚡ Performance

- Scheduler optimizations
- Connection pooling
- Memory usage improvements
- Benchmark suite

### 📊 Metrics & Observability

- Additional metric types
- OpenTelemetry integration
- Distributed tracing
- Enhanced logging

### 📚 Documentation

- Usage examples
- Architecture diagrams
- Video tutorials
- Translations

### 🧪 Testing

- Increase test coverage
- Integration tests
- Chaos testing
- Performance benchmarks

### 🚀 Infrastructure

- Helm chart
- Kubernetes operator
- Terraform modules
- CI/CD improvements

## Code of Conduct

### Our Standards

We are committed to providing a welcoming and inspiring community:

✅ **Do:**
- Use welcoming and inclusive language
- Be respectful of differing viewpoints
- Accept constructive criticism gracefully
- Focus on what's best for the community
- Show empathy towards others

❌ **Don't:**
- Use sexualized language or imagery
- Engage in trolling or insulting comments
- Harass others publicly or privately
- Publish others' private information
- Engage in other inappropriate professional conduct

### Enforcement

Violations may be reported to the maintainers. All complaints will be reviewed and investigated, resulting in appropriate responses.

## Questions?

If you have questions:

- 💬 Create an issue with the `question` label
- 📖 Check documentation in `docs/`
- 🔍 Search existing issues and discussions

---

<p align="center">
  <strong>Thank you for contributing to VMProber! 🎉</strong>
</p>

<p align="center">
  <img src="assets/logo-inline.svg" alt="VMProber" width="120">
</p>
