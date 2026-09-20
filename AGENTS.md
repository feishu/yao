# Yao Engine - Project Rules

Reply in Chinese; use Conventional Commits with Chinese or English descriptions (e.g., `feat(asset): ...`, `fix(v8): ...`).

## Overview

Yao is an open-source application engine designed for building web services, REST APIs, and enterprise business applications. It allows developers to define applications via declarative JSON/Yao DSL, JavaScript/TypeScript business scripts (V8 runtime), and built-in Admin UI (Xgen/SUI) with AI-first workflows.

- **Ecosystem Role**: Top-level engine binary orchestrating `gou` (runtime), `kun` (utils), `xun` (DBAL), and `v8go` (V8 CGO).
- **Multi-Repo Note**: Local development with sister repositories uses `go.mod` `replace` directives.

## Tech Stack

- **Core Runtime**: Go 1.25+ (Toolchain 1.25.5)
- **Web Framework**: Gin (HTTP router, middleware, streaming)
- **Scripting Engine**: `rogchap.com/v8go` (Customized V8 CGO bindings for JS/TS)
- **App Framework**: `github.com/yaoapp/gou` (Process dispatch, DSL loading, Models)
- **Database Layer**: `github.com/yaoapp/xun` (Eloquent-style DBAL, multi-dialect SQL)
- **Foundations**: `github.com/yaoapp/kun` (Structured logs, exception system)
- **Frontend / UI**: React + TypeScript (Xgen admin UI, SUI template engine)
- **Supported DBs**: MySQL, PostgreSQL, SQLite3, Dameng

## Common Commands & Fast Feedback Loop

```bash
# Fast Feedback Inner Loop (<5s)
go vet ./engine/...                     # Static analysis on modified package
go test -v -run TestEngine ./engine/... # Run targeted subsystem unit test
go run . run <process> [args...]        # Test-run a Yao process directly

# Development & Service
go run . start                          # Start HTTP service & API gateway
go run . migrate                        # Run database schema migrations

# Quality & Full Build
make fmt                                # Format Go code (go fmt)
make vet                                # Static analysis across packages
go build -o dist/yao .                  # Build local production binary
make test                               # Run full test suite with coverage
```

## Navigation & Key Entrypoints

| Subsystem / Responsibility | Primary Entry File / Directory |
| :--- | :--- |
| **CLI Commands Entry** | `cmd/root.go`, `cmd/start.go`, `cmd/run.go`, `main.go` |
| **Engine Initialization & Process Router** | `engine/load.go`, `engine/process.go` |
| **DAG Asset Engine & Preheating** | `asset/asset.go`, `asset/dag.go`, `asset/discovery.go` |
| **HTTP Service, Routing & Guards** | `service/service.go`, `service/guard.go`, `service/server.go` |
| **SUI Server-side UI Engine** | `sui/core/`, `sui/api/` |
| **Xgen Admin UI Generator** | `xgen/` |
| **AI Assistant & Generative Content** | `neo/`, `aigc/`, `openai/` |

## Directory Structure

```
yao/
├── cmd/               # CLI commands entry points (root, start, run, migrate, pack)
├── engine/            # Core engine initialization, process router & lifecycle
├── asset/             # Concurrent DAG asset loading engine & preheating
├── service/           # HTTP server, Gin routing, auth guards & middleware
├── sui/               # SUI server-side rendered UI & template engine
├── xgen/              # Xgen admin interface generator & static asset bindings
├── aigc/              # AI generative content integrations & orchestration
├── neo/               # Built-in AI assistant & conversation memory
├── model/             # Model loading & ORM metadata registry
├── api/               # API DSL parsing & HTTP handler dispatch
├── flow/              # Workflow pipeline execution engine
├── script/            # V8 script execution adapter
├── share/             # Shared constants, version definition & utilities
├── config/            # Application configuration & env parsing
├── test/              # Test suites, fixtures & environment setup
└── main.go            # Engine CLI main entry point
```

## System Invariants (DO NOT BREAK)

1. **DAG Topological Order**: Asset loading order in `asset/dag.go` (models -> connectors -> flows -> scripts -> apis) must be preserved to prevent dependency deadlocks.
2. **Universal Context Propagation**: Every CLI command and HTTP request must propagate `context.Context` down into `gou.Process` and `xun` queries for instant cancellation.
3. **Clean Architecture Layering**: Entities stay in `types/` or domain packages; Process adapters stay in `process.go`; loaders stay in `load.go`.

## Footguns & Anti-Patterns (DO NOT)

- **DO NOT** use standard Go `panic()` in engine/service handlers. Always throw Yao exceptions: `exception.New(err.Error(), 500).Throw()`.
- **DO NOT** launch unmanaged background goroutines without passing and listening to `ctx.Done()`.
- **DO NOT** edit pre-packaged frontend artifacts in `dist/` directly; make changes in source packages and run `make pack`.
- **DO NOT** break backward compatibility of public Process signatures in `engine/process.go`.
- **DO NOT** nest or re-acquire the same `sync.Mutex` / `sync.RWMutex` in internal loading, parser, or helper call paths; Go mutexes are strictly non-reentrant.
