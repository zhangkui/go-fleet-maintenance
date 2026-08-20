# BUG-017 Reproduction

## 现象
车辆近期油耗里放入明显偏高和明显偏低的记录后，两条记录的 abnormal 都还是 false，异常记录数量也是 0。先帮我分析清楚，不需要修改项目；请沿着最近样本的顺序、基准油耗和阈值比较查下去，说明为什么两个方向的异常都会漏掉。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug017_FuelAnomalyDetection$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/fuel.go、internal/repository/mysql/fuel.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	// 超过均值 15 倍或低于均值 40% 判定异常。
    -	return current > avg*15 || current < avg*0.4
    +	// 超过均值 1.5 倍或低于均值 40% 判定异常。
    +	return current > avg*1.5 || current < avg*0.4
    -		FROM fuel_records WHERE vehicle_id=? ORDER BY odometer_km ASC LIMIT ?`, vehicleID, limit)
    +		FROM fuel_records WHERE vehicle_id=? ORDER BY odometer_km DESC LIMIT ?`, vehicleID, limit)

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
