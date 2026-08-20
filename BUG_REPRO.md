# BUG-018 Reproduction

## 现象
创建带配件的维保工单后，工单主记录能看到，但库存还是 100，配件明细是 0 条，配件费也是 0；再次消耗后库存仍没有从 100 变成 93。主工单、配件明细和库存应该一次完成，任何一步失败都要一起撤销，麻烦帮我修一下这条创建流程。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug018_MaintenancePartsConsumption$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/maintenance.go、internal/repository/mysql/maintenance.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -		// 校验配件存在但跳过扣减库存。
    +		// 扣减配件库存并写流水。
    -			if _, err := stores.Parts.GetPartByIDForUpdate(ctx, parts[i].PartID); err != nil {
    +			p, err := stores.Parts.GetPartByIDForUpdate(ctx, parts[i].PartID)
    +			if err != nil {
    +				return err
    +			}
    +			if p.StockQuantity < parts[i].Quantity {

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
