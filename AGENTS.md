# AGENTS Instructions for c4 Repository

This repository contains Go code. Please follow these conventions when making changes:

- **Formatting**: Run `gofmt -w` on all Go source files before committing.
- **Testing**: Execute `go test ./...` and ensure all tests pass.
- **Linting**: Run `go vet ./...` to catch common issues.
- **Commit messages**: Use concise titles followed by a blank line and a detailed description when necessary.

## Project Overview

The `c4` project is a Go implementation of the C4 ID system (SMPTE ST 2114:2017). C4 IDs are short, self-identifying, base58-encoded SHA‑512 digests designed to be:

- Globally unique and deterministic for the same content.
- Stable across locations, filenames, and timestamps (content-derived only).
- Easy to use in filenames, URLs, DB keys, JSON, and XML.

See `README.md` for background, examples, and links to the whitepaper and talks.

## Repository Layout

- `doc.go`: Package-level summary for `c4`.
- `id/`: Core C4 ID implementation (encoding/decoding, digest, slices, identify helpers).
- `store/`: Abstractions for opening/creating content by C4 ID (folder, map, ram, validating, logger).
- `manifest/`: Utilities for manifests and file modes; integrates with `db`.
- `db/`: BoltDB-backed helpers used by higher-level features.
- `util/`: Shared helpers (e.g., charset utilities).
- Top-level `id.go`, `tree.go`: Convenience wrappers mirroring `id/` capabilities for the root package.

Primary docs worth skimming:

- `README.md`: Motivation, features, and example usage.
- `id/doc.go`: Concepts, APIs, and patterns for single/multi-file identification.
- `store/doc.go`: Store abstraction and common use cases.

## Using the Library

Install in a module-aware project:

```bash
go get github.com/bgyss/c4@latest
```

Basic single-stream identification (also in README):

```go
id := c4.Identify(strings.NewReader("alfa"))
fmt.Println(id) // c4...
```

Multiple inputs (order-insensitive via slice insertion):

```go
var ids c4.IDs
for _, s := range []string{"alfa","bravo"} {
  ids = append(ids, c4.Identify(strings.NewReader(s)))
}
fmt.Println(ids.ID())
```

Parsing an ID string:

```go
parsed, err := c4.Parse("c4...")
```

Stores (read/write by C4 ID) are available under `store/` (e.g., folder, map, ram).

## Development Workflow

- Branching: `dev` is the integration branch; `master` tracks releases. Open PRs against `dev` unless otherwise requested.
- Style: Standard Go formatting and idioms; keep changes minimal and focused.
- Tests: Prefer adding or updating focused tests near the code you touch.
- Docs: Update `README.md` or package docs (`doc.go`) when changing behavior or APIs.

## Pre-Submit Checklist

- `gofmt -w .`
- `go vet ./...`
- `go test ./...` (run from repo root)

Tip: Run package-scoped tests first when iterating, e.g. `go test ./id -run TestID`.

## Releasing

- Current release is listed in `README.md` (e.g., `v0.8.1`).
- Keep changes backward compatible where possible; avoid breaking exported types without discussion.

## Security & License

- Security policies and contact are in `SECURITY.md`.
- Licensed under MIT (`LICENSE`).

## Quick Pointers

- Standard reference: SMPTE ST 2114:2017.
- C4 string format: 90 chars, `c4` prefix, base58 (no visually ambiguous chars).
- Regex for discovery: `c4[1-9A-HJ-NP-Za-km-z]{88}`.
