# go-fleet-maintenance

Fleet maintenance management system with a Go REST API, MySQL, Redis, and a Vue 3 administration UI.

## Local Development

Start the application services:

```text
docker compose up --build
```

The web application is available at `http://localhost:8080`. The default local administrator is `admin / Admin123!`.

## Go Verification

Run the portable unit test suite directly with Go:

```text
go test ./... -count=1
```

Run a single regression test with an anchored test name:

```text
go test ./internal/service -count=1 -run '^TestName$'
```

Repository SQL tests use `go-sqlmock`, and Redis behavior tests use `miniredis`; they do not require Docker, Bash, MySQL, or Redis services.

## Optional Integration Tests

The integration package is guarded by the `integration` build tag. Supply reachable MySQL and Redis endpoints, then run:

```text
go test -tags=integration ./tests/integration/... -count=1
```

Required environment variables are documented in `tests/integration/integration_test.go`.
