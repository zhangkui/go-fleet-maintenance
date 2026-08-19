# BUG-005 Reproduction

## Symptom
车辆列表请求按车牌升序排序时，实际方向相反，并且车牌排序可能按车型列执行。

## Trigger

    go test ./internal/transport/http/handler -count=1 -run '^TestBug005_VehiclePlateSorting$'

## Defective Result

    sort={Field:plate_number Order:desc}
    FAIL

## Expected Result
合法排序字段与方向经白名单解析后原样传递，车牌升序生成 plate_number ASC；默认排序、分页和过滤保持不变。