# Contributing to Wain

Thank you for your interest in contributing to Wain! This guide will help you get started.

## Code Style

### TODO Comments

All TODO comments in the codebase must reference a tracked item in `TECHNICAL_DEBT.md`:

```go
// TODO(TD-N): Brief description
```

Where `N` is the item number in TECHNICAL_DEBT.md.

**DO NOT** use untracked TODO comments like:
```go
// TODO: Fix this later  ❌
```

**Process for adding new TODOs:**
1. Add entry to TECHNICAL_DEBT.md with:
   - TD-N identifier (next available number)
   - File location
   - Priority (High/Medium/Low)
   - Description
   - Impact
   - Effort estimate
   - Related items
2. Reference it in code: `// TODO(TD-N): Description`

This ensures technical debt is visible and trackable.

### Testing

- Use `t.Parallel()` for independent tests to speed up test execution
- Run tests with: `make test-go` (includes CGO_LDFLAGS for Rust library linking)
- Never use `go test ./...` directly (will fail due to missing CGO flags)

### Documentation

- All exported identifiers require godoc comments
- Use imperative mood: "Add does X" not "Add is for doing X"
- Include edge cases and error conditions
- Target: 95%+ documentation coverage

### Naming

- Follow Go naming conventions (effective Go guide)
- Use mixedCase for multi-word identifiers
- Common acronyms: API, HTTP, URL, ID (all caps when part of identifier)

## Building

```bash
# Build Rust library
make build-rust

# Build Go binary
make build

# Verify static linkage
make check-static
```

## Pre-commit Checklist

- [ ] Tests pass: `make test-go`
- [ ] No vet warnings: `go vet ./...`
- [ ] Static linkage verified: `make check-static`
- [ ] Documentation added for new exports
- [ ] TODOs tracked in TECHNICAL_DEBT.md

## Questions?

Open an issue or check existing documentation:
- README.md - Project overview and setup
- ROADMAP.md - Development phases and status
- TECHNICAL_DEBT.md - Known limitations and planned improvements
