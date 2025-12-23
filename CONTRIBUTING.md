# Contributing to VMProber

Thank you for your interest in VMProber! We welcome contributions from the community.

## How to Contribute

### Reporting Bugs

If you found a bug:

1. Check if it hasn't already been reported in [Issues](https://github.com/gdagil/vmprober/issues)
2. If not, create a new issue with:
   - Clear description of the problem
   - Steps to reproduce
   - Expected and actual behavior
   - VMProber version, Go version, OS
   - Logs and configuration (without secrets)

### Suggesting Enhancements

For new feature suggestions:

1. Check existing issues and discussions
2. Create an issue describing:
   - The problem the feature solves
   - Proposed solution
   - Alternatives considered (if any)

### Pull Requests

1. **Fork the repository**
2. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```
3. **Make changes** following code standards (see below)
4. **Add tests** for new features
5. **Update documentation** if needed
6. **Ensure all tests pass**:
   ```bash
   make test
   make lint
   ```
7. **Create a Pull Request** with a description of changes

## Code Standards

### Formatting

Use standard Go tools:

```bash
make fmt
```

### Linting

Code must pass linting:

```bash
make lint
```

### Testing

- All new features must have tests
- Code coverage should be at least 80%
- Run tests:
  ```bash
  make test
  make test-coverage
  ```

### Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add ICMP probe support
fix: resolve scheduler race condition
docs: update configuration documentation
test: add tests for UDP probe
refactor: simplify metrics collector
chore: update dependencies
```

Commit types:
- `feat` - new feature
- `fix` - bug fix
- `docs` - documentation changes
- `test` - adding or modifying tests
- `refactor` - code refactoring
- `chore` - dependency updates, configuration, etc.
- `perf` - performance improvements
- `style` - formatting, no logic changes

### Naming

- **Packages**: lowercase, single word
- **Functions**: `ExportFunction` for public, `internalFunction` for private
- **Variables**: `exportedVar` for public, `internalVar` for private
- **Constants**: `ExportedConstant`

### Comments

All public functions, types, and variables must have comments:

```go
// Collector collects and exports metrics
type Collector struct {
    // ...
}

// NewCollector creates a new metrics collector
func NewCollector(namespace string) *Collector {
    // ...
}
```

### Error Handling

Always check errors:

```go
result, err := someFunction()
if err != nil {
    return nil, fmt.Errorf("failed to execute: %w", err)
}
```

## Development Process

### Setting Up Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/vmprober.git
   cd vmprober
   ```
3. Add upstream:
   ```bash
   git remote add upstream https://github.com/gdagil/vmprober.git
   ```
4. Install dependencies:
   ```bash
   make deps
   ```

### Working on Changes

1. Update your branch:
   ```bash
   git checkout main
   git pull upstream main
   ```
2. Create a branch for changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. Make changes
4. Commit changes:
   ```bash
   git add .
   git commit -m "feat: add your feature"
   ```
5. Push changes:
   ```bash
   git push origin feature/your-feature-name
   ```

### Creating Pull Request

1. Go to GitHub and create a Pull Request
2. Fill out the PR template:
   - Description of changes
   - Related issues (if any)
   - Checklist of completed tasks
3. Wait for review from maintainers

### After Creating PR

- Respond to comments and change requests
- Update PR if needed
- Ensure CI passes successfully

## Areas for Contribution

### Code

- New probe types (ICMP, HTTP, etc.)
- Scheduler improvements
- Performance optimizations
- Bug fixes

### Documentation

- Improving existing documentation
- Adding examples
- Documentation translations
- Fixing typos

### Tests

- Increasing test coverage
- Integration tests
- Benchmarks

### Infrastructure

- CI/CD improvements
- Docker images
- Kubernetes manifests

## Code of Conduct

### Our Standards

- Use welcoming and friendly language
- Respect different viewpoints and experiences
- Accept constructive criticism gracefully
- Focus on what's best for the community
- Show empathy towards other community members

### Unacceptable Behavior

- Use of sexualized language or imagery
- Trolling, insulting/derogatory comments
- Public or private harassment
- Publishing others' private information without permission
- Other conduct that could be inappropriate in a professional setting

## Questions?

If you have questions:

- Create an issue with the `question` label
- Contact maintainers
- Check existing documentation in `docs/`

## Acknowledgments

Thank you to all contributors who make VMProber better!

---

**Thank you for your contribution!** 🎉
