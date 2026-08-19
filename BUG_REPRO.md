# BUG-007 Reproduction

## 现象
车辆里程更新链路错误修改 created_at，并给 updated_at 传入回拨时间；相同里程重复提交也被当作递减请求拒绝。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug007_VehicleMileageTimestamps$'

## 缺陷基线结果

    --- FAIL: TestBug007_VehicleMileageTimestamps (0.00s)
        bug007_010_native_test.go:20: equal mileage: 里程必须单调递增
    FAIL
    exit code 1

## 正确结果
相同里程幂等返回且不改时间字段；更小里程继续被拒绝；里程增加时 created_at 保持不变，并使用本次操作时间推进 updated_at。候选修复运行同一命令应返回退出码 0。
