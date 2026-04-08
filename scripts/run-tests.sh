#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SDK_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -h|--help)
    HELP=yes
    shift # past argument
    ;;
    --update-snapshots)
    UPDATE_SNAPSHOTS=yes
    shift # past argument
    ;;
    --integration)
    INTEGRATION=yes
    shift # past argument
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

function help {
  cat << EOF
Test runner
Runs tests for the deploy CLI SDK:

Usage:
  bash scripts/run-tests.sh [options]

Options:
  -h, --help           Show this help message
  --update-snapshots   Update test snapshots
  --integration        Include stateio integration tests (requires Docker)
                       This will start cloud storage emulators (LocalStack,
                       fake-gcs-server, Azurite) and Postgres for stateio tests.

Examples:
  # Run unit tests only (no Docker required)
  bash scripts/run-tests.sh

  # Run all tests including stateio integration (requires Docker)
  bash scripts/run-tests.sh --integration
EOF
  return 0
}

if [[ -n "$HELP" ]]; then
  help
  exit 0
fi

set -eo pipefail

cd "$SDK_DIR"

get_docker_container_status() {
  local container_name="$1"
  docker inspect -f "{{ .State.Status }} {{ .State.ExitCode }}" "$container_name"
  return 0
}

if [[ -n "$INTEGRATION" ]]; then
  # Pull postgres migrations from the blueprint-state module if not already present.
  MIGRATIONS_DIR="$SDK_DIR/stateio/postgres/migrations"
  if [[ ! -d "$MIGRATIONS_DIR" ]]; then
    echo "Pulling postgres migrations from blueprint-state module..."
    go mod download github.com/newstack-cloud/bluelink/libs/blueprint-state
    BLUEPRINT_STATE_DIR="$(go env GOMODCACHE)/$(go list -m -f '{{.Path}}@{{.Version}}' github.com/newstack-cloud/bluelink/libs/blueprint-state)"
    mkdir -p "$MIGRATIONS_DIR"
    cp "$BLUEPRINT_STATE_DIR/postgres/migrations/"*.sql "$MIGRATIONS_DIR/"
    echo "Copied $(ls "$MIGRATIONS_DIR" | wc -l | tr -d ' ') migration files."
  else
    echo "Postgres migrations already present, skipping download."
  fi

  echo "Starting stateio integration test dependencies (cloud storage emulators + postgres)..."
  docker compose --env-file "$SDK_DIR/.env.test" \
    -f "$SDK_DIR/stateio/docker-compose.test-deps.yml" \
    --project-directory "$SDK_DIR/stateio" up -d

  cleanup() {
    echo "Stopping test dependencies..."
    docker compose --env-file "$SDK_DIR/.env.test" \
      -f "$SDK_DIR/stateio/docker-compose.test-deps.yml" \
      --project-directory "$SDK_DIR/stateio" down
    return 0
  }
  trap cleanup EXIT

  # Wait for postgres migrations to complete
  echo "Waiting for postgres migrations to complete..."
  status="$(get_docker_container_status stateio_sdk_test_postgres_migrate)"
  while [[ "$status" != "exited 0" ]]; do
    if [[ "$status" == "exited 1" ]]; then
      echo "Postgres migration failed, see logs below:"
      docker logs stateio_sdk_test_postgres_migrate
      exit 1
    fi
    sleep 1
    status="$(get_docker_container_status stateio_sdk_test_postgres_migrate)"
  done

  echo "Waiting for LocalStack to be ready..."
  start=$EPOCHSECONDS
  completed="false"
  while [[ "$completed" != "true" ]]; do
    sleep 5
    completed=$(curl -s localhost:4580/_localstack/init/ready | jq .completed 2>/dev/null || echo "false")
    if (( EPOCHSECONDS - start > 60 )); then
      echo "LocalStack readiness timed out"
      exit 1
    fi
  done

  echo "Creating S3 test bucket and uploading test files..."
  aws --endpoint-url=http://localhost:4580 s3 mb s3://test-bucket --region eu-west-2 2>/dev/null || true
  aws --endpoint-url=http://localhost:4580 s3api put-object --bucket test-bucket \
    --body "$SDK_DIR/stateio/__testdata/s3/instances.json" --key instances.json --region eu-west-2

  echo "Waiting for Azurite to be ready..."
  sleep 3

  # Export environment variables for integration tests
  echo "Exporting environment variables for test suite..."
  set -a
  source "$SDK_DIR/.env.test"
  set +a

  # Make sure the Google Cloud SDK uses the fake GCS server emulator
  export STORAGE_EMULATOR_HOST="http://localhost:8185"
fi

# Determine which packages to test
TEST_PACKAGES=$(go list ./... | grep -v '/testutils$')
if [[ -z "$INTEGRATION" ]]; then
  # Exclude stateio package when not running integration tests
  # (its tests require cloud storage emulators and postgres)
  TEST_PACKAGES=$(echo "$TEST_PACKAGES" | grep -v '/stateio$')
fi

echo "" > coverage.txt

GO_TEST_ARGS="-count=1 -timeout 90000ms -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic"

echo "Running tests..."
if [[ -n "$GITHUB_ACTION" ]]; then
  # In CI, use -json for SonarCloud report. Tee to stdout so failures are visible in logs.
  if [[ -n "$UPDATE_SNAPSHOTS" ]]; then
    UPDATE_SNAPSHOTS=true go test $GO_TEST_ARGS -json $TEST_PACKAGES | tee report.json
  else
    go test $GO_TEST_ARGS -json $TEST_PACKAGES | tee report.json
  fi
else
  if [[ -n "$UPDATE_SNAPSHOTS" ]]; then
    UPDATE_SNAPSHOTS=true go test $GO_TEST_ARGS $TEST_PACKAGES
  else
    go test $GO_TEST_ARGS $TEST_PACKAGES
  fi
  go tool cover -html=coverage.txt -o coverage.html
  echo ""
  echo "Coverage report: coverage.html"
fi

echo ""
echo "Tests complete!"
