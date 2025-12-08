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

## Further documentation

- [Commit Guidelines](./COMMIT_GUIDELINES.md)
- [Source Control and Release Strategy](./SOURCE_CONTROL_RELEASE_STRATEGY.md)
