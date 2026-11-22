# Deploy CLI SDK

A Go library of commands, TUIs and helpers for the Bluelink CLI (`apps/cli` in bluelink) and
the Celerity CLI (`apps/cli` in celerity). The Bluelink CLI passes a `commands.CLIConfig`
(see `commands/config.go`) and registers stage, deploy, destroy, instances, state and cleanup
with `commands.Setup*Command`; its own `validate` runs `tui/validateui`. The Celerity CLI so
far uses only `config`, `styles`, `engine` and `tui/precommand`, with its own validate TUI,
and its stage, deploy and destroy aren't wired to this SDK yet.

## Workspace

This repo is one of NewStack's. NewStack's maintainers develop it alongside the others in a
shared workspace, using the NewStack Claude Code plugin for how work is planned, reviewed,
committed and coordinated across repos. The plugin isn't public, and nothing here depends on it:
this file and the repo's own docs cover what a contribution needs.

## Layout

| Path | Holds |
|---|---|
| `commands/` | Command setup, `CLIConfig`, flag and environment variable binding |
| `tui/<name>ui/` | One Bubble Tea app per command or view (`stageui`, `deployui`, `destroyui`, `validateui`, `driftui`, `inspectui`, `listui`, `cleanupui`, `stateexportui`, `stateimportui`); `tui/shared`, `tui/preflight` and the `*util` packages hold what they share |
| `tui/precommand/` | The `PreCommandStep` interface a CLI implements to run a step before a command |
| `ui/` | Reusable components: blueprint and file selection, `splitpane` |
| `headless/`, `jsonout/` | Plain output without a terminal; `--json` output |
| `engine/`, `config/` | Deploy engine client setup; CLI and deploy config loading |
| `stateio/` | State export and import between the engine's memfile or postgres store and export files, local or on S3, GCS or Azure Blob |
| `styles/` | Styles and the Bluelink and Celerity palettes |
| `diagutils/`, `consts/`, `strings/` | Diagnostic formatting, shared constants, string helpers |
| `testutils/` | Test helpers, including a fake deploy engine and `teatest` helpers |

## Checks

Commands are in [CONTRIBUTING.md](CONTRIBUTING.md).

- **Tools:** Go (`go.mod`), `staticcheck` on `PATH` (CI installs `@latest`, so a local copy
  can disagree), Node with Yarn 4 for commitlint (`yarn` first; the tracked `.yarnrc.yml`
  makes it create `node_modules`). Integration tests also need docker, the `aws` CLI, `jq`
  and `curl`; without `jq` or `curl`, the run waits and then fails with "LocalStack readiness
  timed out".
- **Lint:** `bash scripts/lint.sh` runs `staticcheck` and `go vet` per package.
- **Tests:** `bash scripts/run-tests.sh` runs every package except `stateio` and `testutils`
  with `-race` and coverage. `stateio`'s unit tests run only with `--integration`, and
  `go test ./...` fails there without the stack. `--update-snapshots` is accepted, but no
  tests use snapshots.
  - `--integration` adds `stateio`. It downloads the `libs/blueprint-state` version pinned in
    `go.mod`, copies its postgres migrations into `stateio/postgres/` (gitignored), starts
    `stateio/docker-compose.test-deps.yml` (LocalStack on 4580, fake-gcs-server on 8185,
    Azurite on 10003, postgres on 45434), and stops it on exit.
  - Set `AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test`, as CI does. Without them the
    script's `aws` calls and the Go S3 client use the shell's credentials (against LocalStack
    only), and with none at all the S3 seeding step fails.
  - Run one stack at a time: the container names are fixed, so a second checkout's run
    clashes with the first. The ports don't clash with Bluelink's stacks.
- **Trying a change in a CLI:** there's no `go.work` here, so point the CLI's module at this
  checkout with a temporary `replace`, or a `go.work` outside both repos, and don't commit
  either. Bluelink's root `go.work` is committed and doesn't include this repo.
- **Gates:**
  - With `git config core.hooksPath .githooks`, pre-commit runs `scripts/lint.sh` over the
    whole module on every commit, and commit-msg runs commitlint.
  - CI (`ci.yml`) runs lint, `run-tests.sh --integration` and SonarCloud on pull requests and
    pushes to `main`. `codeql.yml` runs CodeQL on the same triggers and weekly;
    `infrastructure-security.yml` runs actionlint when a workflow or shell script changes,
    and weekly.

## Compatibility

- **Go API:** pre-1.0 (`bump-minor-pre-major`, so a `feat` or a breaking change bumps the
  minor version). Each CLI pins its own version in `go.mod`, and the pins can differ. A
  breaking change says in its PR what each CLI must change when it bumps.
- **Command behaviour:** flags, environment variables
  (`<EnvVarPrefix>_<COMMAND>[_<SUBCOMMAND>]_<FLAG>`, bound per flag in `commands/`; not every
  flag has one), headless output and `--json` output are what CLI users and their scripts see. Additive only; the CLI docs in bluelink-docs and
  celerity-docs (`content/docs/cli/`) describe them.
- **Bluelink libraries:** `libs/blueprint`, `libs/blueprint-state`, `libs/common` and
  `libs/deploy-engine-client` are pinned in `go.mod`, so a change in bluelink reaches this
  repo only by release and bump. Renovate raises those bumps as bluelink releases and merges
  them itself when CI passes. Read the pinned code from the module cache
  (`go list -m -f '{{.Dir}}' <module>`), not a checkout.
- **Persisted state:** none of its own. `stateio` reads and writes the engine's memfile and
  postgres stores and export files in blueprint-state's format, so it follows bluelink's
  stance on persisted state.

## Conventions

- Commits: [COMMIT_GUIDELINES.md](COMMIT_GUIDELINES.md), with the types in
  `commitlint.config.js`. Scopes are allowed but unused; release-please releases the whole
  module as one package.
- Releases: release-please opens the release pull request; merging it tags `vX.Y.Z`. Don't
  edit `CHANGELOG.md` or `.release-please-manifest.json` by hand.
- Dependency updates: Renovate (`renovate.json`, run hourly by `renovate.yml`).
- Anything specific to one CLI goes through `CLIConfig` or a factory it passes in (such as
  `PreflightFactory`), not a check for the CLI's name.

## Safety

- Don't create tags or releases, or merge release pull requests.
- Don't run commands from this SDK against a running deploy engine unless asked: the
  maintainer's engine may be serving runs against real cloud accounts.
- `state import` and `state export` bypass the engine and work on the store named in the
  engine config file directly; don't point them at the maintainer's engine config.

## Known mistakes

<!-- One line per mistake an agent has made here, phrased as the rule that prevents it. -->
