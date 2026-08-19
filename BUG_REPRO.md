# BUG-009 Reproduction

## 现象
保险或年检证照记录可以创建，但车辆资料中的对应到期日没有正确同步。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug009_VehicleLicenseExpiry$'

## 缺陷结果

    --- FAIL: TestBug009_VehicleLicenseExpiry (0.00s)
        bug007_010_native_test.go:62: kind="POLICY-1"
    FAIL
    exit code 1

## 调查结论
服务层把证照编号作为 kind 传给到期日更新，使仓储 switch 无法命中；仓储中保险和年检对应的更新列又彼此互换。当前两种证照都不会更新车辆字段，只修服务传参后则会写入对方的到期字段。
