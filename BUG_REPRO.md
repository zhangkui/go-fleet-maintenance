# BUG-015 Reproduction

## 现象
任务交接接口现在会接受几种不合理的请求：原司机和新司机相同、原司机 ID 为 0、新司机 ID 为 0 时都返回了 201，预期应该是 400；没有提供新司机时也不应该自动换成某个固定用户。这次只查问题，不用动代码；麻烦说明这些输入分别是在哪一步被放过去的。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug015_TripHandoverValidation$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/trip.go、internal/transport/http/handler/trip.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	if h.TripID == 0 {
    -		return entity.TripHandover{}, domain.NewCoded("validation_error", "任务不能为空", domain.ErrValidation)
    +	if h.TripID == 0 || h.FromDriverID == 0 || h.ToDriverID == 0 {
    +		return entity.TripHandover{}, domain.NewCoded("validation_error", "任务与交接司机不能为空", domain.ErrValidation)
    +	}
    +	if h.FromDriverID == h.ToDriverID {
    +		return entity.TripHandover{}, domain.NewCoded("validation_error", "原司机与新司机不能相同", domain.ErrValidation)
    -	if hd.FromDriverID == 0 {

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
