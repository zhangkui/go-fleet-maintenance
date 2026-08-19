# go-fleet-maintenance BENZHI Evaluation

## Project Description
go-fleet-maintenance is a Go fleet-maintenance REST API with MySQL, Redis, and a Vue 3 administration UI. The backend module declares Go 1.22.

## Standard Commands

    cd '/app' && GOTOOLCHAIN=local go build ./...
    cd '/app' && GOTOOLCHAIN=local go test ./... -count=1
    cd '/app' && GOTOOLCHAIN=local go run ./cmd/api

## Docker Build Template

    chmod +x build_benzhi_docker.sh
    ./build_benzhi_docker.sh go-fleet-maintenance-bug006-candidate linux/amd64

The Docker command is an evaluation template only. Docker and architecture validation were not run.

## BUG-006 Verification Command

    cd '/app' && GOTOOLCHAIN=local go test ./internal/service -count=1 -run '^TestBug006_PermissionAggregation$'

Expected defective diagnosis candidate result: exit code 1.

## Bug Reproduction
See BUG_REPRO.md.