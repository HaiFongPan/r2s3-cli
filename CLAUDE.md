# Claude Code - Development Excellence Guidelines

*As Claude Code, I establish these comprehensive development principles to ensure code quality, reliability, and maintainability.*

## 🎯 Core Development Principles

### 1. **Test-First Development**
Every modification MUST pass all tests before completion:
```bash
# Mandatory test execution after each change
go test ./...
```
- ✅ All existing tests must pass
- ✅ New features require corresponding unit tests
- ✅ Test coverage should be maintained or improved
- ❌ No code commits without passing tests

### 2. **Build Integrity Assurance**
Every change MUST result in a successful build:
```bash
# Mandatory build verification
go build .
```
- ✅ Zero compilation errors
- ✅ All dependencies properly resolved
- ✅ Go language standards compliance
- ✅ Import paths correctness verified

## 🧪 Testing Excellence

### Test Commands Arsenal
```bash
# Complete test suite execution
go test ./...

# Package-specific testing
go test ./internal/tui
go test ./cmd
go test ./internal/utils

# Verbose test output with details
go test -v ./...

# Coverage analysis and reporting
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race condition detection
go test -race ./...

# Benchmarking performance tests
go test -bench=. ./...
```

### Testing Standards
- **Unit Tests**: Every public function requires tests
- **Integration Tests**: Critical workflows must be tested end-to-end
- **Error Path Testing**: All error conditions must be covered
- **Edge Case Coverage**: Boundary conditions and corner cases
- **Mock Testing**: External dependencies properly mocked

## 🏗️ Build Excellence

### Build Commands
```bash
# Standard development build
go build .

# Production build with output specification
go build -o bin/r2s3-cli .

# Cross-platform compilation matrix
GOOS=linux GOARCH=amd64 go build -o bin/r2s3-cli-linux-amd64 .
GOOS=darwin GOARCH=amd64 go build -o bin/r2s3-cli-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -o bin/r2s3-cli-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o bin/r2s3-cli-windows-amd64.exe .

# Optimized production builds
go build -ldflags="-s -w" -o bin/r2s3-cli .
```

### Build Quality Gates
- **Dependency Management**: `go mod tidy` before builds
- **Vendor Verification**: `go mod verify` for integrity
- **Static Analysis**: `go vet ./...` for code issues
- **Format Compliance**: `go fmt ./...` for consistency

## 📐 Code Quality Standards

### Go Excellence Practices
```bash
# Code formatting (mandatory)
go fmt ./...

# Static analysis and linting
go vet ./...

# Advanced linting with golangci-lint
golangci-lint run

# Import organization
goimports -w .

# Code complexity analysis
gocyclo .
```

### Code Quality Principles
- **Single Responsibility**: Each function serves one clear purpose
- **Error Handling**: Comprehensive error handling with context
- **Documentation**: GoDoc comments for all exported functions
- **Naming Conventions**: Clear, descriptive, and Go-idiomatic names
- **Interface Design**: Small, focused interfaces following Go philosophy
- **Memory Management**: Efficient resource usage and cleanup

## 🔄 Git Workflow Excellence

### Pre-Commit Checklist
```bash
# Comprehensive pre-commit verification
go fmt ./...           # Format code
go vet ./...           # Static analysis
go test ./...          # Run all tests
go build .             # Verify build
go mod tidy            # Clean dependencies
```

### Commit Standards
- **Atomic Commits**: Each commit represents one logical change
- **Descriptive Messages**: Clear, actionable commit messages
- **Feature Branches**: Use feature branches for major changes
- **Progressive Commits**: Break large features into smaller, testable commits

### Commit Message Format
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Examples:
```
feat(upload): add batch delete optimization using DeleteObjects API
fix(progress): unify progress bar format across single and multi-file uploads
refactor(config): improve configuration validation and error handling
test(utils): add comprehensive tests for progress bar functionality
docs(api): update API documentation with new endpoints
```

## 🏛️ Architecture Excellence

### Project Structure
```
r2s3-cli/
├── cmd/                    # Command-line interface layer
│   ├── root.go            # Root command and global flags
│   ├── upload.go          # Upload command implementation
│   ├── download.go        # Download command implementation
│   ├── delete.go          # Delete command implementation
│   └── list.go            # List command implementation
├── internal/              # Internal packages (not for external use)
│   ├── config/           # Configuration management
│   │   ├── config.go     # Configuration structures
│   │   └── validation.go # Configuration validation
│   ├── r2/               # R2 client abstraction
│   │   ├── client.go     # R2 client implementation
│   │   └── auth.go       # Authentication handling
│   ├── tui/              # Terminal user interface
│   │   ├── components/   # Reusable TUI components
│   │   ├── views/        # Application views
│   │   └── theme/        # UI theming
│   └── utils/            # Utility functions
│       ├── progress.go   # Progress bar implementations
│       ├── validation.go # Input validation
│       └── helpers.go    # Common helper functions
├── examples/             # Example configurations and usage
├── docs/                 # Documentation
├── scripts/              # Build and deployment scripts
├── main.go              # Application entry point
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── Makefile             # Build automation
├── README.md            # Project documentation
└── CLAUDE.md            # This development guide
```

### Design Principles
- **Separation of Concerns**: Clear boundaries between layers
- **Dependency Injection**: Testable, loosely coupled components
- **Error Propagation**: Consistent error handling patterns
- **Configuration Management**: Centralized, validated configuration
- **Logging Strategy**: Structured logging with appropriate levels

## 🚀 Performance Excellence

### Performance Monitoring
```bash
# CPU profiling
go test -cpuprofile cpu.prof -bench .

# Memory profiling
go test -memprofile mem.prof -bench .

# Profile analysis
go tool pprof cpu.prof
go tool pprof mem.prof
```

### Performance Standards
- **Batch Operations**: Use bulk APIs where available (e.g., DeleteObjects)
- **Progress Indicators**: Provide user feedback for long operations
- **Resource Management**: Proper cleanup of resources and connections
- **Concurrency**: Utilize Go's concurrency patterns appropriately
- **Memory Efficiency**: Minimize allocations in hot paths

## 🔒 Security Excellence

### Security Practices
- **Credential Management**: Secure handling of API keys and secrets
- **Input Validation**: Comprehensive validation of all user inputs
- **Path Sanitization**: Prevent directory traversal attacks
- **Error Information**: Avoid exposing sensitive data in error messages
- **Dependency Security**: Regular dependency vulnerability scanning

## 📚 Documentation Excellence

### Documentation Requirements
- **API Documentation**: GoDoc for all exported functions
- **Usage Examples**: Practical examples in README
- **Configuration Guide**: Complete configuration documentation
- **Troubleshooting**: Common issues and solutions
- **Development Setup**: Clear setup instructions for contributors

## ⚡ Automation Excellence

### Makefile Targets
```makefile
.PHONY: test build clean lint format check-deps

test:
	go test ./...

build:
	go build -o bin/r2s3-cli .

clean:
	rm -rf bin/
	go clean

lint:
	golangci-lint run

format:
	go fmt ./...
	goimports -w .

check-deps:
	go mod verify
	go mod tidy

all: format lint test build
```

## 🎖️ Excellence Commitment

*As Claude Code, I commit to maintaining these standards in every code modification, ensuring that this project remains a benchmark of Go development excellence.*

---

**Remember**: Excellence is not an act, but a habit. Every line of code is an opportunity to demonstrate craftsmanship and professionalism.
