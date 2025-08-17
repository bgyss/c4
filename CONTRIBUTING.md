# Contributing to C4

Thank you for your interest in contributing to C4! This document provides guidelines and information for contributors.

## 🎯 Project Overview

C4 ID is an implementation of the SMPTE standard ST 2114:2017 for universally unique and consistent identifiers. The project aims to provide a robust, performant, and secure implementation of the C4 ID system in Go.

## 🛠 Development Setup

### Prerequisites

- Go 1.20 or later
- Git
- (Optional) Nix for reproducible builds

### Getting Started

1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-username/c4.git
   cd c4
   ```

2. **Set up Development Environment**
   
   **Option A: Using Nix (Recommended)**
   ```bash
   nix develop
   # This provides Go, golangci-lint, and all necessary tools
   ```
   
   **Option B: Manual Setup**
   ```bash
   # Ensure Go 1.20+ is installed
   go version
   
   # Install linting tools
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install golang.org/x/vuln/cmd/govulncheck@latest
   ```

3. **Verify Setup**
   ```bash
   # Run tests
   go test ./...
   
   # Run linter
   golangci-lint run
   
   # Build CLI
   go build ./cmd/c4
   ```

## 📝 Development Workflow

### Branch Organization

- `master`: Current stable release
- `develop`: Development branch (base for feature branches)
- `feature/*`: Feature branches
- `bugfix/*`: Bug fix branches
- `hotfix/*`: Critical fixes for production

### Creating a Contribution

1. **Create a Branch**
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/your-feature-name
   ```

2. **Make Changes**
   - Write code following project conventions
   - Add/update tests for your changes
   - Update documentation if needed

3. **Test Your Changes**
   ```bash
   # Run all tests
   go test ./...
   
   # Check coverage
   go test -cover ./...
   
   # Run linter
   golangci-lint run
   
   # Run security check
   govulncheck ./...
   ```

4. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   ```

5. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   # Create PR via GitHub interface
   ```

## 📋 Coding Standards

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use meaningful variable and function names
- Write clear, concise comments for public APIs
- Keep functions focused and reasonably sized

### Testing Requirements

- **Minimum 75% test coverage** for new code
- Write unit tests for all public functions
- Include edge cases and error conditions
- Add benchmarks for performance-critical code

### Documentation

- Document all exported functions, types, and constants
- Include usage examples for complex APIs
- Update README.md for user-facing changes
- Add inline comments for complex logic

## 🎯 Quality Standards

### Before Submitting

Ensure your contribution meets these standards:

- [ ] All tests pass (`go test ./...`)
- [ ] Code coverage ≥ 75% for modified packages
- [ ] Linter passes (`golangci-lint run`)
- [ ] No security vulnerabilities (`govulncheck ./...`)
- [ ] Documentation updated
- [ ] Commit messages follow conventional format

### Conventional Commits

Use conventional commit format for clear history:

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Adding/updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(manifest): add new file traversal algorithm
fix(id): handle edge case in digest computation
docs(readme): update installation instructions
test(store): add benchmarks for cache operations
```

## 🏗 Architecture Guidelines

### Package Organization

- **Root package**: Core C4 ID functionality
- **`cmd/c4/`**: CLI application
- **`db/`**: Database operations
- **`store/`**: Storage abstraction layer
- **`manifest/`**: File manifest management
- **`util/`**: Utility functions

### Performance Considerations

- Benchmark performance-critical code
- Avoid unnecessary allocations in hot paths
- Use appropriate data structures for use cases
- Consider memory usage for large datasets

### Security Guidelines

- Never log sensitive data
- Validate all inputs
- Use crypto/rand for randomness
- Follow secure coding practices

## 🐛 Reporting Issues

### Bug Reports

Include in your bug report:

- Go version and OS
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Any relevant error messages

### Feature Requests

Include in your feature request:

- Use case and motivation
- Proposed API or interface
- Potential implementation approach
- Impact on existing functionality

## 🔄 Review Process

### Pull Request Requirements

- Clear description of changes
- Links to related issues
- Tests for new functionality
- Documentation updates
- Passing CI checks

### Review Criteria

Reviewers will evaluate:

- Code quality and style
- Test coverage and quality
- Performance impact
- Security implications
- API design consistency
- Documentation completeness

## 📚 Resources

### Useful Links

- [C4 ID Whitepaper](http://www.cccc.io/c4id-whitepaper-u2.pdf)
- [SMPTE ST 2114:2017 Standard](https://www.smpte.org/)
- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)

### Getting Help

- Open an issue for questions
- Join discussions in existing issues
- Review existing code for patterns
- Check documentation and examples

## 📄 License

By contributing to C4, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to C4! 🚀