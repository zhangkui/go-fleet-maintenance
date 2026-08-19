# BUG-008 Reproduction

## 现象
合法车辆状态迁移成功后，状态历史中的 from_status 被写成目标状态；仓储 INSERT 又把 from_status 与 to_status 的参数顺序颠倒，导致历史记录与车辆主表状态不一致。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug008_VehicleStatusHistory$'

## 缺陷基线结果

    --- FAIL: TestBug008_VehicleStatusHistory (0.00s)
        bug007_010_native_test.go:44: history={ID:0 VehicleID:1 FromStatus:in_maintenance ToStatus:in_maintenance Reason:service ChangedBy:9 ChangedAt:0001-01-01 00:00:00 +0000 UTC}
    FAIL
    exit code 1

## 正确结果
合法迁移以变更前状态写入 from_status，以目标状态写入 to_status；车辆主表当前状态与历史 to_status 一致。失败操作仍由现有事务回滚，状态转换规则保持不变。
