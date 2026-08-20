# BUG-016 Reproduction

## 现象
录入一笔不能整除的油耗费用时，接口计算出来的总价多了一分：期望 699，实际是 700；数据库里同一笔记录最后又成了 701。金额单位一直是分，麻烦帮我看看为什么同一笔费用会被连续取整，并让返回值和落库值保持一致。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug016_FuelCostRounding$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/fuel.go、internal/repository/mysql/fuel.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		product := in.LitersMilli * in.UnitPriceCents
    -		in.TotalCostCents = product / 1000
    -		if product%1000 != 0 {
    -			in.TotalCostCents++
    -		}
    +		in.TotalCostCents = in.LitersMilli * in.UnitPriceCents / 1000
    -	totalCostCents := f.TotalCostCents
    -	if f.LitersMilli*f.UnitPriceCents%1000 != 0 {

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
