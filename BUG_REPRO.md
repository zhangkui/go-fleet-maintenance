# BUG-011 Reproduction

## 现象
给司机创建休息班次时，`off` 被接口当成非法类型返回了 400，但 morning、evening、night 都能正常返回 201。`off` 本来就是系统支持的班次，未知类型和空值仍然应该报错，麻烦帮我把这处校验不一致的问题处理一下。

## 触发命令

    go test ./internal/service -count=1 -run '^TestBug011_DriverOffShiftValidation$'

## 缺陷基线结果
固定回归命令在缺陷分支退出码为 1，目标业务断言失败。

## 失效链路
涉及生产文件：internal/service/driver.go、internal/domain/entity/driver.go。请求输入或任务状态在这些职责层之间传递时被错误转换、遗漏或使用，最终形成题面中的可见结果。

## Gold 机制变化

    -	case entity.ShiftMorning, entity.ShiftEvening, entity.ShiftNight:
    +	case entity.ShiftMorning, entity.ShiftEvening, entity.ShiftNight, entity.ShiftOff:
    -	ShiftOff     = "rest"
    +	ShiftOff     = "off"

## 正确结果
题面中的目标场景恢复正常，边界与对照场景保持原有行为；既有测试文件和断言不作修改。
