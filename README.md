# NewStack Deploy CLI SDK

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_deploy-cli-sdk&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_deploy-cli-sdk)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_deploy-cli-sdk&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_deploy-cli-sdk)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_deploy-cli-sdk&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_deploy-cli-sdk)

An SDK with shared TUI components, behaviours and utilities for building CLIs that handle blueprint deployments such as the Bluelink and Celerity CLIs.

## Installation

```bash
go get github.com/newstack-cloud/deploy-cli-sdk
```

## Overview

The SDK provides the following packages:

- **commands** — Shared CLI command factories (deploy, destroy, stage, cleanup, state) parameterised by a `CLIConfig` for branding and defaults.
- **tui** — Bubbletea TUI models for interactive deployment workflows including staging, deploying, destroying, state import/export, and drift review.
- **diagutils** — Converts blueprint diagnostic errors into actionable CLI commands and registry links.
- **jsonout** — Structured JSON output types for headless/CI mode across all operations.
- **stateio** — Import/export of deploy engine state from/to local files and remote storage (S3, GCS, Azure Blob).
- **config** — Configuration provider with flag and environment variable binding.
- **engine** — Deploy engine client setup and configuration.
- **styles** — TUI colour palettes and styling utilities.
- **headless** — Headless mode flag validation and output formatting.

## Documentation

- [Contributing](CONTRIBUTING.md)
- [Commit Guidelines](COMMIT_GUIDELINES.md)
- [Source Control and Release Strategy](SOURCE_CONTROL_RELEASE_STRATEGY.md)
