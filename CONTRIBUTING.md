# Contributing to NewStack Deploy CLI SDK

## Setup

Ensure git uses the custom directory for git hooks so the pre-commit and commit-msg linting hooks
kick in.

```bash
git config core.hooksPath .githooks
```

### Requirements

- [Go](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download/)
- [Yarn](https://yarnpkg.com/getting-started/install)

### NPM dependencies

There are npm dependencies that provide tools that are used in git hooks for commit linting.

Install dependencies from the root directory by simply running:
```bash
yarn
```

## Further documentation

- [Commit Guidelines](./COMMIT_GUIDELINES.md)
- [Source Control and Release Strategy](./SOURCE_CONTROL_RELEASE_STRATEGY.md)
