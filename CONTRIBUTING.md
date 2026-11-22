# Contributing to NewStack Deploy CLI SDK

## Setup

Ensure git uses the custom directory for git hooks so the pre-commit and commit-msg linting hooks
kick in.

```bash
git config core.hooksPath .githooks
```

### Requirements

- [Go](https://go.dev/doc/install) >= 1.26.4 (`go.mod`)
- [Node.js](https://nodejs.org/en/download/) >= 22.20.0
- [Yarn](https://yarnpkg.com/getting-started/install) 4 (`packageManager` in `package.json`)

### NPM dependencies

There are npm dependencies that provide tools that are used in git hooks for commit linting.

Install dependencies from the root directory by simply running:
```bash
yarn
```

## Running Tests

Run all tests with integration tests, race detection and coverage:

```bash
bash scripts/run-tests.sh --integration
```

This will:
- Run all tests with race detection enabled
- Include integration tests
- Generate `coverage.txt` with coverage data
- Generate `coverage.html` for visual coverage inspection (on dev machines)

To run unit tests only (without integration tests):

```bash
bash scripts/run-tests.sh
```

To run tests for a specific package:

```bash
go test -v ./styles/...
go test -v ./config/...
```

`stateio`'s tests, unit tests included, need the integration stack, so `go test ./...` fails
there without it; use `bash scripts/run-tests.sh --integration` for that package.

The integration run needs docker, the `aws` CLI, `jq` and `curl`, and test credentials for
LocalStack, as CI sets them:

```bash
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test bash scripts/run-tests.sh --integration
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
