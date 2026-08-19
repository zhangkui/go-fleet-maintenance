# BUG-010 Reproduction

## 现象
创建司机与车辆绑定时，新绑定的起始时间落在已有活跃绑定区间内，接口仍可能返回 201，而不是拒绝冲突并返回 409。仅边界相接但没有实际重叠的下一段绑定应继续允许创建。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug010_DriverVehicleBindingWindow$'

## 缺陷基线结果

    --- FAIL: TestBug010_DriverVehicleBindingWindow (0.01s)
        bug007_010_native_test.go:80: checked at=2026-08-19 20:22:06.2225149 +0800 CST m=+0.003131901 want=2026-09-01 00:00:00 +0000 UTC
    FAIL
    exit code 1

## 失效链路
`DriverService.CreateBinding` 把当前时间传给冲突查询，未使用请求中的绑定起始时间；`DriverRepository.HasActiveBindingForVehicle` 又查询 ended 状态并忽略时间窗口，因此未来生效的真实重叠记录会被漏掉。

## 正确结果
服务层使用新绑定的 StartDate 检查冲突；仓储层只查询 active 绑定，并用 start_date、end_date 与指定日期判断该时刻是否落在现有有效区间内。修复后同一条锚定测试退出码为 0，其他绑定创建和错误映射行为保持不变。
