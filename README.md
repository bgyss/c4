
# C4 ID - Universally Unique and Consistent Identification


[![Go Report Card](https://goreportcard.com/badge/github.com/bgyss/c4)](https://goreportcard.com/report/github.com/bgyss/c4)
[![CI](https://github.com/bgyss/c4/workflows/CI/badge.svg)](https://github.com/bgyss/c4/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/bgyss/c4/branch/master/graph/badge.svg)](https://codecov.io/gh/bgyss/c4)
[![GoDoc](https://godoc.org/github.com/bgyss/c4?status.svg)](https://godoc.org/github.com/avalanche-io/c4)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![Release](https://img.shields.io/github/release/bgyss/c4.svg)](https://github.com/bgyss/c4/releases/latest)

```go
import "github.com/bgyss/c4"
```

This is a Go package that implements the C4 ID system **SMPTE standard ST 2114:2017**. C4 IDs are universally unique and consistent identifiers that standardize the derivation and formatting of data identification so that all users independently agree on the identification of any block or set of blocks of data.

C4 IDs are 90 character long strings suitable for use in filenames, URLs, database fields, or anywhere else that a string identifier might normally be used. In ram C4 IDs are represented in a 64 byte "digest" format.

#### Features

- A single C4 id can represent multiple files.
- C4 ids are unique, random, and unforgeable.
- C4 ids are identical for the same file in different locations or points in time.
- A network connection is not required to generate C4 ids.
- A C4 id can be used in filenames, URLs, json and xml.
- C4 ids can be selected easily with double click (_a problem for many unique identifiers_).
- Easily discover C4 ids in arbitrary text with a simple regex `c4[1-9A-HJ-NP-Za-km-z]{88}`
- Naming files by their C4 id automatically deduplicates them.

#### Comparison of Encodings

C4 is the shortest self identifying SHA-512 encoding and is the only standardized encoding.
To illustrate, the following is the SHA-512 of "foo" in hex, base64 and c4 encodings:

```yaml
# encoding     length   id
  hex          135:     sha512-f7fbba6e0636f890e56fbbf3283e524c6fa3204ae298382d624741d0dc6638326e282c41be5e4254d8820772c5518a2c5a8c0c7f7eda19594a7eb539453e1ed7
  base64        95:     sha512-9/u6bgY2+JDlb7vzKD5STG+jIErimDgtYkdB0NxmODJuKCxBvl5CVNiCB3LFUYosWowMf37aGVlKfrU5RT4e1w==
  c4            90:     c43inc2qGhSWQUMRvDMW6GAjJnRFY5sxq399wcUcWLTuPai84A2QWTfYu1gAW8f5FmZFGeYpLsSPyrSUh9Ao3J68Cc
```

### Example Usage

```go
package main

import (
  "fmt"
  "strings"

  "github.com/bgyss/c4"
)

func main() {

  // Generate a C4 ID for any contiguous block of data...
  id := c4.Identify(strings.NewReader("alfa"))
  fmt.Println(id)
  // output: c43zYcLni5LF9rR4Lg4B8h3Jp8SBwjcnyyeh4bc6gTPHndKuKdjUWx1kJPYhZxYt3zV6tQXpDs2shPsPYjgG81wZM1

  // Generate a C4 ID for any number of non-contiguous blocks...
  var ids c4.IDs
  var inputs = []string{"alfa", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india"}
  for _, input := range inputs {
    ids = append(ids, c4.Identify(strings.NewReader(input)))
  }
  fmt.Println(ids.ID())
  // output: c435RzTWWsjWD1Fi7dxS3idJ7vFgPVR96oE95RfDDT5ue7hRSPENePDjPDJdnV46g7emDzWK8LzJUjGESMG5qzuXqq
}
```

---

## Installation & Building

### Using Nix (Recommended)

This project includes a Nix flake for reproducible builds and development environments across multiple platforms.

#### Quick Start

```bash
# Build the CLI tool
nix build

# Run directly without installing
echo "Hello World" | nix run

# Enter development environment
nix develop
```

#### Supported Platforms

The flake supports building for all major platforms:
- `aarch64-darwin` (Apple Silicon macOS)
- `aarch64-linux` (ARM64 Linux)
- `i686-linux` (32-bit x86 Linux)
- `x86_64-darwin` (Intel macOS)
- `x86_64-linux` (64-bit x86 Linux)

#### Platform-Specific Builds

```bash
# Build for specific platform
nix build .#packages.x86_64-linux.c4
nix build .#packages.aarch64-linux.c4

# View all available platforms
nix flake show --all-systems
```

#### Development Environment

The development shell includes all necessary tools:

```bash
# Enter the development environment
nix develop

# Available tools in the shell:
go build ./cmd/c4          # Build the CLI tool
go test ./...              # Run all tests
go test -cover ./...       # Run tests with coverage
golangci-lint run          # Run linter
```

#### CI/CD Integration

```bash
# Run all checks (build, test, lint)
nix flake check

# Run checks for all platforms
nix flake check --all-systems
```

#### direnv Integration (Optional)

For automatic environment loading when entering the directory:

```bash
# Allow direnv (if you have direnv installed)
direnv allow

# The development environment will now load automatically
```

### Traditional Go Build

```bash
# Build manually with Go
go build -o c4 ./cmd/c4

# Run tests
go test ./...
```

### Docker (Including Synology Container Manager)

```bash
# Build from source
docker build -t c4:local .

# Identify piped data
echo "Hello World" | docker run --rm -i c4:local

# Identify files in the current directory (mounted read-only at /data)
docker run --rm -v "$PWD:/data:ro" c4:local -R /data
```

For Synology DSM 7+ Container Manager:

1. Pull `avalancheio/c4:latest` in the Registry tab.
2. The published image is multi-arch and includes both `linux/amd64` (x86_64 Synology) and `linux/arm64` variants.
3. Create a project using `docker-compose.synology.yml` from this repository.
4. Update `/volume1/data` in that file to the NAS folder you want to scan.
5. Keep restart policy set to `no` because this is a command-style container (it exits after work is done).
6. Run the project and view logs to collect generated C4 IDs.

---

## Testing & Quality

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. -run=^$ ./...
```

### Coverage Reports

The project maintains high test coverage across all packages:

- **Overall Coverage**: ~78%
- **Core Package**: 92.4%
- **Store Package**: 80.9%
- **Manifest Package**: 72.9%
- **Util Package**: 100%

### Performance Benchmarks

Performance benchmarks are run continuously to track regression:

```bash
# Run core performance benchmarks
go test -bench=BenchmarkIdentify -benchmem ./...

# Run memory allocation benchmarks
go test -bench=BenchmarkMemoryAllocation -benchmem ./...

# Run platform-specific benchmarks
go test -bench=. -benchmem ./...
```

### Code Quality

The project uses several tools to maintain code quality:

- **golangci-lint**: Comprehensive linting with 20+ enabled linters
- **gosec**: Security vulnerability scanning
- **govulncheck**: Dependency vulnerability checking
- **gofmt**: Code formatting consistency
- **go vet**: Static analysis for potential issues

### Continuous Integration

All code is validated through GitHub Actions CI/CD:

- ✅ Multi-platform testing (Linux, macOS, Windows)
- ✅ Multi-version Go support (1.20, 1.21)
- ✅ Automated security scanning
- ✅ Performance regression testing
- ✅ Coverage reporting
- ✅ Dependency vulnerability checks
- ✅ Nix build validation

---

### Releases

Current release: [v0.8.1](https://github.com/bgyss/c4/tree/v0.8.1)

### Links

Videos:
  - [C4 Framework Universal Asset ID](https://youtu.be/ZHQY0WYmGYU)
  - [The Magic of C4](https://youtu.be/vzh0JzKhY4o)

[C4 ID Whitepaper](http://www.cccc.io/c4id-whitepaper-u2.pdf)

### Contributing

Contributions are welcome. The following are some general guidelines for project organization. If you have questions please open an issue.

The `master` branch holds the current release, and older releases can be found by their version number. The `dev` branch represents the development branch from which bug and feature branches should be taken. Pull requests that are accepted will be merged against the `dev` branch and then pushed to versioned releases as appropriate.

Feature and bug branches should follow the github integrated naming convention.  Features should be given the `new` tag, and bugs the `bug` tag.  Here is an example of checking out a feature branch:

```bash
> git checkout dev
Switched to branch 'dev'
Your branch is up-to-date with 'origin/dev'.
> git checkout -b new/#99_some_github_issue
...
```

If a branch for an issue is already listed in this repository, then check it out and work from it.

### License
This software is released under the MIT license.  See [LICENSE](./LICENSE) for more information.
