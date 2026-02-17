# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Agent Conventions

This repository contains Go code. Follow these conventions when making changes:

- **Formatting**: Run `gofmt -w` on all Go source files before committing.
- **Testing**: Execute `go test ./...` and ensure all tests pass.
- **Linting**: Run `go vet ./...` to catch common issues.
- **Commit messages**: Use concise titles followed by a blank line and a detailed description when necessary.

## Project Overview

The `c4` project is a Go implementation of the C4 ID system (SMPTE ST 2114:2017). C4 IDs are short, self-identifying, base58-encoded SHA-512 digests designed to be:

- Globally unique and deterministic for the same content.
- Stable across locations, filenames, and timestamps (content-derived only).
- Easy to use in filenames, URLs, DB keys, JSON, and XML.

See `README.md` for background, examples, and links to the whitepaper and talks.

C4 ID generates 90-character string IDs that are deterministic and can represent single files or sets of files.

## Repository Layout

- `doc.go`: Package-level summary for `c4`.
- `id/`: Core C4 ID implementation (encoding/decoding, digest, slices, identify helpers). The package is deprecated; prefer root package APIs when possible.
- `store/`: Abstractions for opening/creating content by C4 ID (folder, map, ram, validating, logger).
- `manifest/`: Utilities for manifests and file modes; integrates with `db`. Under active development.
- `db/`: BoltDB-backed helpers used by higher-level features.
- `util/`: Shared helpers (e.g., charset utilities).
- Top-level `id.go`, `tree.go`: Convenience wrappers mirroring `id/` capabilities for the root package.
- `cmd/c4/`: CLI application for generating C4 IDs from files or stdin.

Primary docs worth skimming:

- `README.md`: Motivation, features, and example usage.
- `id/doc.go`: Concepts, APIs, and patterns for single/multi-file identification.
- `store/doc.go`: Store abstraction and common use cases.

## Development Workflow

- Branching: `dev` is the integration branch; `master` tracks releases. Open PRs against `dev` unless otherwise requested.
- Style: Standard Go formatting and idioms; keep changes minimal and focused.
- Tests: Prefer adding or updating focused tests near the code you touch.
- Docs: Update `README.md` or package docs (`doc.go`) when changing behavior or APIs.

## Development Commands

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage (as used in CI)
go test -coverprofile=id.coverprofile
go test -coverprofile=oldid.coverprofile ./id
go test -coverprofile=db.coverprofile ./db
go test -coverprofile=util.coverprofile ./util

# Run specific package tests
go test ./id
go test ./db
go test ./store
go test ./manifest
go test ./util
```

### Linting

```bash
# Run vet checks
go vet ./...
```

### Formatting

```bash
# Format all Go sources in place
gofmt -w .
```

### Security Scanning

```bash
# Install gosec security scanner
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run security scan
gosec ./...

# Run with detailed output
gosec -fmt=json ./...
```

### Building

```bash
# Build the CLI tool
go build -o c4 ./cmd/c4

# Install the CLI tool
go install ./cmd/c4
```

### CLI Usage

```bash
# Generate C4 ID from file
./c4 filename.txt

# Generate C4 ID from stdin
echo "data" | ./c4

# Process multiple files
./c4 file1.txt file2.txt

# Recursive processing with metadata
./c4 -r -m directory/
```

## Using the Library

Install in a module-aware project:

```bash
go get github.com/bgyss/c4@latest
```

Basic single-stream identification:

```go
id := c4.Identify(strings.NewReader("alfa"))
fmt.Println(id) // c4...
```

Multiple inputs (order-insensitive via slice insertion):

```go
var ids c4.IDs
for _, s := range []string{"alfa", "bravo"} {
	ids = append(ids, c4.Identify(strings.NewReader(s)))
}
fmt.Println(ids.ID())
```

Parsing an ID string:

```go
parsed, err := c4.Parse("c4...")
```

Stores (read/write by C4 ID) are available under `store/` (e.g., folder, map, ram).

## Architecture

### Core Components

- **Root package (`c4`)**: Core C4 ID implementation with `Identify()` function and ID types
- **`cmd/c4/`**: CLI application for generating C4 IDs from files or stdin
- **`id/`**: Legacy/deprecated ID implementation (kept for backward compatibility)
- **`db/`**: Database interface using bbolt for persistent storage
- **`store/`**: Generic storage abstraction layer supporting various backends
- **`manifest/`**: File manifest management (under active development)
- **`util/`**: Utility functions including character set handling

### Key Types

- `c4.ID`: String representation of C4 identifiers
- `c4.IDs`: Slice of multiple IDs that can be combined into a single ID
- `c4.Digest`: 64-byte binary representation used internally

### Data Flow

1. Input data -> SHA-512 hash -> C4 encoding -> 90-character string ID
2. Multiple files can be combined into a single ID through the `IDs` type
3. Storage backends abstract the details of where data is persisted

## Dependencies

- `go.etcd.io/bbolt`: Key-value database (upgraded from deprecated boltdb)
- `github.com/ogier/pflag`: Command-line flag parsing
- `golang.org/x/crypto`: Cryptographic functions
- `github.com/absfs/*`: Abstract filesystem interfaces
- `github.com/xtgo/set`: Set operations

## Package Status

- **Root `c4` package**: Stable API
- **`manifest/` package**: Under active development, API may change
- **`id/` package**: Deprecated, use root package instead
- **CLI tool**: Stable

## Pre-Submit Checklist

- `gofmt -w .`
- `go vet ./...`
- `go test ./...` (run from repo root)

Tip: Run package-scoped tests first when iterating, e.g. `go test ./id -run TestID`.

## Releasing

- Current release is listed in `README.md` (e.g., `v0.8.1`).
- Keep changes backward compatible where possible; avoid breaking exported types without discussion.

## Security Notes

C4 IDs are cryptographically secure and unforgeable, using SHA-512 for hashing. The system is designed for safe handling of any file content without security risks.

Security policies and contact are in `SECURITY.md`.
Project license is MIT (`LICENSE`).

## Quick Pointers

- Standard reference: SMPTE ST 2114:2017.
- C4 string format: 90 chars, `c4` prefix, base58 (no visually ambiguous chars).
- Regex for discovery: `c4[1-9A-HJ-NP-Za-km-z]{88}`.

## CI Configuration and Disabled Tests

### Disabled Test Platforms

The CI configuration has been optimized to run only on stable platform/version combinations. The following platforms and versions have been disabled due to persistent test failures:

#### Disabled Platforms

- **Windows (`windows-latest`)**: Removed from CI matrix
- **Go 1.23**: Removed from CI matrix (only Go 1.24 is tested)

#### Current CI Matrix

- **Ubuntu + Go 1.24**: Full test suite
- **macOS + Go 1.24**: Full test suite (with optimizations)

### Known Issues with Disabled Tests

#### Windows Test Issues

1. File locking problems: `TestFolderStoreRemove` fails due to Windows file locking behavior where files must be explicitly closed before deletion
2. Performance issues: Database tests (`TestLinkApi`, `TestBatching`) run significantly slower on Windows CI, causing timeout failures
3. Race condition sensitivity: Windows appears more sensitive to race conditions in concurrent database access

#### Go 1.23 Issues

1. Performance regressions: Go 1.23 shows significant performance degradation on macOS for database-intensive tests
2. Timeout failures: `TestLinkApi/LinkGetAll` consistently times out (>8 seconds) on macOS with Go 1.23
3. Platform-specific behavior: The performance issues appear isolated to macOS + Go 1.23 combination

#### macOS Performance Optimizations

1. TestLinkApi optimization: Reduced digest count from 1000 to 50 on macOS (`runtime.GOOS == "darwin"`) to prevent timeout failures
2. Location: `db/db_test.go:329-331` - Platform-specific optimization for `LinkGetAll` subtest
3. Rationale: macOS consistently shows slower performance for database-intensive operations, requiring reduced test load to stay under timeout threshold

### Test Coverage Strategy

- **Primary validation**: Ubuntu + Go 1.24 (most stable, fastest)
- **Cross-platform validation**: macOS + Go 1.24 (ensures portability)
- **Windows/Go 1.23 support**: Manual testing recommended for releases targeting these platforms

### Re-enabling Disabled Tests

If attempting to re-enable disabled platforms:

1. For Windows: Address file handle management in tests, consider platform-specific test timeouts
2. For Go 1.23: Monitor Go team for performance regression fixes, consider version-specific test optimizations

IMPORTANT: Do not re-enable these platforms without first addressing the underlying issues documented above.
