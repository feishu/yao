# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

Yao is an all-in-one application engine that enables developers to create web apps, REST APIs, business applications, and more, with AI as a development partner. Yao uses a Domain-Specific Language (DSL) that is both human-readable and AI-friendly, allowing seamless transitions between AI-generated and manually written code.

**Homepage**: https://yaoapps.com  
**Documentation**: https://yaoapps.com/docs  
**Current Version**: 0.10.4

## Development Environment

### Go Installation

**Go Binary Location**: `/usr/local/go/bin/go`

If `go` is not in your PATH, use the full path:
```bash
# Run Go commands with full path
/usr/local/go/bin/go version
/usr/local/go/bin/go test ./...
/usr/local/go/bin/go build -o ./dist/release/yao .
```

To add Go to your PATH permanently, add this to your `~/.zshrc` or `~/.bash_profile`:
```bash
export PATH="/usr/local/go/bin:$PATH"
```

## Build and Development Commands

### Core Build Commands
```bash
# Build the main binary
go build -o yao .
# Or with full path if go not in PATH:
/usr/local/go/bin/go build -o yao .

# Build with make (recommended)
make

# Run all tests
make test

# Format code
make fmt

# Check code formatting
make fmt-check

# Run linter
make lint

# Check for misspellings
make misspell-check
make misspell

# Vet code
make vet
```

### Development Mode
```bash
# Run in development mode
./yao start --dev

# Start with specific app directory
./yao start -a /path/to/app

# Run with environment file
./yao start -f .env.dev

# Inspect application configuration
./yao inspect

# Run a specific process
./yao run <process.name>

# Migrate database schema
./yao migrate

# Show version information
./yao version
```

### Testing Commands
```bash
# Run single test file
go test ./aigc -v
# Or with full path:
/usr/local/go/bin/go test ./aigc -v

# Run tests with coverage
go test -cover ./...

# Run specific test function
go test -run TestFunctionName ./package

# Run specific package tests (example)
/usr/local/go/bin/go test -v ./utils/x2j/

# Test with race detection
go test -race ./...

# Benchmark tests
go test -bench=. ./...
```

### SUI Template Engine Commands
```bash
# Watch and build SUI templates
./yao sui watch

# Build SUI templates
./yao sui build

# Transform SUI templates
./yao sui trans
```

### Package Management
```bash
# Generate bindata
make bindata

# Create distribution package
make pack

# Build Linux artifacts
make artifacts-linux

# Build macOS artifacts  
make artifacts-macos

# Debug build
make debug

# Production release
make release
```

## High-Level Architecture

Yao follows a modular, layered architecture with clear separation of concerns:

### Core Engine (`engine/`)
- **Engine Loader**: Orchestrates the loading of all application components in sequence
- **Process Registry**: Manages all available processes and their execution
- **Bootstrap Sequence**: Handles application initialization, database connections, and component loading

### Command Layer (`cmd/`)
- **Cobra CLI Framework**: All commands are implemented using spf13/cobra
- **Root Command**: Central command dispatcher with subcommands for different operations
- **Internationalization**: Built-in support for Chinese and English via language maps

### Configuration System (`config/`)
- **Environment-based Config**: Loads from `.env` files with fallbacks
- **Mode-specific Settings**: Automatic production/development mode configuration  
- **Logging Configuration**: File-based logging with rotation via lumberjack

### Key Application Layers

#### Data Layer
- **Models** (`model/`): Database model definitions and operations
- **Connectors** (`connector/`): Database connection management
- **Query Engine** (`query/`): Query processing and execution
- **Stores** (`store/`): Key-value storage abstractions

#### Processing Layer
- **Scripts** (`script/`): JavaScript/TypeScript execution via V8 engine
- **Flows** (`flow/`): Data flow processing and orchestration
- **Pipes** (`pipe/`): Data transformation pipelines
- **Processes**: Business logic processors (distributed across modules)

#### Service Layer
- **APIs** (`api/`): HTTP API endpoint definitions
- **Sockets** (`socket/`): Socket.io server implementations  
- **WebSockets** (`websocket/`): WebSocket client connections
- **Tasks** (`task/`): Asynchronous task management
- **Schedules** (`schedule/`): Cron-like job scheduling

#### Utilities and Extensions
- **File System** (`fs/`): File operations abstraction
- **Crypto** (`crypto/`): Cryptographic utilities (AES, hash functions)
- **Helper** (`helper/`): Utility functions (arrays, strings, JWT, etc.)
- **Excel** (`excel/`): Excel file processing and generation
- **I18n** (`i18n/`): Internationalization support

#### AI/ML Integration
- **AIGC** (`aigc/`): AI-generated content processing
- **Neo** (`neo/`): AI assistant and conversation management
- **OpenAI** (`openai/`): OpenAI API integration
- **RAG** (`neo/rag/`): Retrieval-Augmented Generation
- **Vision** (`neo/vision/`): Image processing capabilities

#### Business Services
- **Payment** (`payment/`): Payment processing (Alipay, WeChat Pay)
- **Excel Import/Export**: Business data exchange
- **Widget System** (`widget/`, `widgets/`): UI component framework

### Key Architectural Patterns

1. **Plugin Architecture**: Modular components that register themselves on initialization
2. **Process-based Architecture**: Everything is a process that can be called via unified interface
3. **DSL-driven Configuration**: JSON/YAML configurations define application behavior
4. **Event-driven Components**: Hooks and callbacks for extensibility
5. **V8 Runtime Integration**: TypeScript/JavaScript execution within Go application

### Loading Sequence
The engine follows a strict loading order (see `engine/load.go`):
1. Application loading (`loadApp`)
2. Database connections (`share.DBConnect`)
3. Certificates (`cert.Load`)
4. Connectors (`connector.Load`)  
5. FileSystem (`fs.Load`)
6. i18n (`i18n.Load`)
7. V8 Runtime (`runtime.Start`)
8. Query Engine (`query.Load`)
9. Scripts (`script.Load`)
10. Models (`model.Load`)
11. Flows (`flow.Load`) 
12. Stores (`store.Load`)
13. Plugins (`plugin.Load`)
14. Widgets (`widgets.Load`)
15. Importers (`importer.Load`)
16. APIs (`api.Load`)
17. Sockets/WebSockets
18. Tasks and Schedules

## Development Rules and Standards

### Code Quality Standards (from .cursorrules)
- Follow Clean Architecture principles with clear layer separation
- Use Domain-Driven Design (DDD) for service boundaries
- Implement interfaces for module decoupling (dependency inversion)
- Apply design patterns: Factory, Decorator, Strategy, Middleware
- Maintain strict adherence to Effective Go standards
- Use `go fmt`, `go vet` for code quality
- Write Table-Driven unit tests with 80%+ coverage
- Use `sync.Pool` for object reuse optimization
- Implement proper context-based goroutine lifecycle management
- Use channels for goroutine communication

### Concurrency Patterns
- Context-based cancellation for all long-running operations
- Worker pool patterns for parallel processing
- Proper use of sync primitives (Mutex, RWMutex, WaitGroup)
- Channel-based communication over shared memory

### Testing Standards
- Table-driven tests following existing patterns
- Benchmark tests for performance-critical code
- Use `gomock` for interface mocking
- Race condition detection with `-race` flag
- Performance profiling with pprof

### Error Handling
- Use the `kun/exception` package for structured error handling
- Implement proper error wrapping and context
- Log errors with structured logging (logrus-compatible)

## Local Dependencies

The project uses local dependencies for core libraries:
```
replace github.com/yaoapp/kun => ../kun
replace github.com/yaoapp/xun => ../xun  
replace github.com/yaoapp/gou => ../gou
replace rogchap.com/v8go => ../v8go
```

Ensure these repositories are cloned as siblings to the main yao directory for local development.

## Environment Configuration

Key environment variables:
- `YAO_MODE`: Set to "development" or "production"
- `YAO_LANG`: Language preference (empty for English, set for Chinese)
- `YAO_APP_SOURCE`: Application source directory or binary reference
- `XGEN_BASE`: Admin interface base path (defaults to "yao")

## File Structure Conventions

- **DSL Files**: Use `.yao`, `.json`, or `.jsonc` extensions
- **Process Files**: Follow naming pattern `<module>.<function>`
- **Script Files**: JavaScript/TypeScript files in designated directories
- **Configuration**: Environment-specific `.env` files
- **Tests**: `*_test.go` files alongside source files
- **Documentation**: Markdown files in `docs/` directory

## Build Artifacts

Production builds create versioned binaries:
- Linux: `yao-{VERSION}-{BUILD}-linux-{ARCH}`
- macOS: `yao-{VERSION}-{BUILD}-darwin-{ARCH}`
- Debug: `yao-debug`

All artifacts are placed in `dist/release/` directory and should be executable.