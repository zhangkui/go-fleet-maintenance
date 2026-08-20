# BUG-026 Reproduction

## 现象
油耗效率报表在不传日期时查不到数据，测试里的里程差、总油量和每公里费用都是 null；手动给日期后，L/100km 又比预期小了大约一百倍。这次不用提交任何改动，把原因查明白即可；请分别说明默认时间范围和油耗公式的问题，再用具体数值讲清单位是怎么变化的。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug026_FuelEfficiencyReport$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/report.go、internal/repository/mysql/report.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		q.From = s.now()
    +		q.From = s.now().AddDate(0, -1, 0)
    -		q.To = s.now().AddDate(0, -1, 0)
    +		q.To = s.now()
    -			rep.LitersPer100KM = float64(rep.TotalLiters) / 10000.0 / float64(rep.TotalDistanceKM)
    +			rep.LitersPer100KM = float64(rep.TotalLiters) / 1000.0 / float64(rep.TotalDistanceKM) * 100

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
