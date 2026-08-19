# go-fleet-maintenance BENZHI Evaluation

## Project Description

`zhangkui/go-fleet-maintenance` is a Go fleet-maintenance REST API with MySQL and Redis integrations and a Vue 3 administration UI. The backend evaluation image uses the `golang:1.22` toolchain. The frontend uses Node.js 22 with Vue 3, TypeScript, and Vite.

## Standard Build, Run, and Test Commands

Inside the evaluation container:

```bash
cd '/app' && GOTOOLCHAIN=local go build ./...
cd '/app' && GOTOOLCHAIN=local go test ./... -count=1
cd '/app' && GOTOOLCHAIN=local go run ./cmd/api
```

## Docker Build and Container Entry

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh go-fleet-maintenance-bug001-candidate linux/amd64
./build_benzhi_docker.sh go-fleet-maintenance-bug001-candidate linux/arm64
docker run --rm -it go-fleet-maintenance-bug001-candidate bash
```

The two commands above document the standard architecture targets. The corresponding validation records state whether each target was actually run.

## BUG-001 Verification Command

```bash
cd '/app' && GOTOOLCHAIN=local go test ./internal/service -count=1 -run '^TestBug001_AuthenticationLockout$'
```

Expected exit code: `0` on the repaired candidate.

## Bug Reproduction

See `BUG_REPRO.md` for the observed BUG-001 symptom, trigger steps, and complete failing output.
