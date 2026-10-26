# AGENTS.md — Go 1.27 Solo Agent & MBVM-DB Standard

Universal configuration following the open `agents.md` specification.
Canonical specification: `AGENTS.mbvm`.

## Overview
- **Language**: Go 1.27 (Solo autonomous agent execution, zero external orchestrators).
- **Context Engine**: MBVM-DB Smart Slicing (99% token reduction over raw files).
- **Single-Ingestion Law**: Heavy assets (Protobuf, JSON catalogs, AST, game data) are ingested ONCE into `.mbvmdb` binary databases (`codebase.mbvmdb`, `game.mbvmdb`). The AI NEVER reads raw 50k+ token files. Always query `mbvm-db slice <db> <term/symbol> [depth]` (60–200 tokens, 1-3ms).

## Commands
- **Context Slice**: `mbvm-db slice <db_or_dir> <symbol_or_term> [depth]`
- **Inspect Bytecode**: `mbvm-db inspect .`
- **Test Suite**: `go test -race -count=1 -v ./...`
- **Static Vet**: `go vet ./...`
- **Security Audit**: `govulncheck ./...`
- **Pre-commit Compile**: `go-tools mbvm .`

## Architecture & Go Conventions
- **Errors**: Always wrap with context using `%w` (`fmt.Errorf("context: %w", err)`). Never use `%v` for errors.
- **Consumer-side Interfaces**: Define interfaces in the package that consumes them (*"accept interfaces, return structs"*).
- **Goroutine Leak Prevention**: Every goroutine must have a deterministic exit path listening to `<-ctx.Done()`.
- **Modern Stdlib**: Use `slog` for structured logging, `slices`/`maps`, `unique` for value interning, and `iter` for range-over-func.
- **Modularity**: Strict 400 LOC ceiling per file. Refactor into cohesive submodules near the limit.

## Never-Do List (Guardrails)
- Never read raw heavy catalogs, Protobuf definitions, or whole packages when a `.mbvmdb` exists.
- Never omit `-race` or `-count=1` when validating tests.
- Never use `time.Sleep` for concurrency synchronization in tests.
- Never introduce `interface{}` or `any` without strict codec justification.
- Never commit with `--no-verify`; the global pre-commit hook must validate AST integrity.
