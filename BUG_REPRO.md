# BUG-019 Reproduction

## 现象
维保工单点完成后，工单状态已经结束了，但车辆还停在 `in_maintenance`，对应停机记录的 downtime_end 也是空的。正常完工应该让车辆恢复 active，同时结束这段停机时间，麻烦帮我看看为什么后两步没有一起完成。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug019_MaintenanceVehicleRecovery$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/maintenance.go、internal/repository/mysql/maintenance.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		// 车辆转回在用态（跳过）。
    -		_, _ = stores.Vehicles.GetVehicleByIDForUpdate(ctx, o.VehicleID)
    +		// 车辆转回在用态。
    +		v, err := stores.Vehicles.GetVehicleByIDForUpdate(ctx, o.VehicleID)
    +		if err != nil {
    +			return err
    +		}
    +		if v.Status == entity.VehicleStatusInMaintenance {

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
