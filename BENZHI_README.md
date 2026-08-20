# go-fleet-maintenance BENZHI Evaluation

## Project Description
go-fleet-maintenance 是一个使用 MySQL、Redis 和 Vue 3 管理端的 Go 车队维护系统，后端模块声明 Go 1.22。

## Standard Commands

    cd '/app' && GOTOOLCHAIN=local go build ./...
    cd '/app' && GOTOOLCHAIN=local go test ./... -count=1
    cd '/app' && GOTOOLCHAIN=local go run ./cmd/api

## Docker Build Template

    chmod +x build_benzhi_docker.sh
    ./build_benzhi_docker.sh go-fleet-maintenance-bug012-candidate linux/amd64

Docker 命令仅作为评测模板保留。本题未运行 Docker 或架构验证，BENZHI_VALIDATION 中均明确记录为 NOT_RUN。

## BUG-012 Verification Command

    go test ./internal/service -count=1 -run '^TestBug012_DriverViolationFineCents$'

该题为 diagnosis，诊断轨迹不修改项目；固定命令在缺陷分支预期保持 red。

## Bug Reproduction
详见 BUG_REPRO.md。
