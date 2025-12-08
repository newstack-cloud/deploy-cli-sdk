# Contributing to NewStack Deploy CLI SDK

## Setup

Ensure git uses the custom directory for git hooks so the pre-commit and commit-msg linting hooks
kick in.

```bash
git config core.hooksPath .githooks
```

### Requirements

- [Go](https://go.dev/doc/install) >= 1.24.4
- [Node.js](https://nodejs.org/en/download/) >= 22.20.0
- [Yarn](https://yarnpkg.com/getting-started/install) >= 4.3.1

### NPM dependencies

There are npm dependencies that provide tools that are used in git hooks for commit linting.

Install dependencies from the root directory by simply running:
```bash
yarn
```

## Running Tests

Run all tests with race detection and coverage:

```bash
bash scripts/run-tests.sh
```

This will:
- Run all tests with race detection enabled
- Generate `coverage.txt` with coverage data
- Generate `coverage.html` for visual coverage inspection (on dev machines)

To update test snapshots (if any):

```bash
bash scripts/run-tests.sh --update-snapshots
```

To run tests for a specific package:

```bash
go test -v ./styles/...
go test -v ./config/...
```

## Linting

Run static analysis and vet checks:

```bash
bash scripts/lint.sh
```

This runs:
- `staticcheck` - static analysis tool for Go
- `go vet` - reports suspicious constructs

### Prerequisites

Install staticcheck if not already installed:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Running Individual Checks

```bash
# Run staticcheck on all packages
staticcheck ./...

# Run go vet on all packages
go vet ./...

# Format code
go fmt ./...
```

## Further documentation

- [Commit Guidelines](./COMMIT_GUIDELINES.md)
- [Source Control and Release Strategy](./SOURCE_CONTROL_RELEASE_STRATEGY.md)
