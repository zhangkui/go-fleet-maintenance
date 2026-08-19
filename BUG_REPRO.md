# BUG-006 Reproduction

## Symptom
用户和角色已被授予多项权限，但权限校验、权限投影和角色权限接口只能得到极少量权限。

## Trigger

    go test ./internal/service -count=1 -run '^TestBug006_PermissionAggregation$'

## Defective Result

    actual SQL ends with WHERE ur.user_id=? LIMIT 1
    expected query returns all permission rows
    FAIL

## Expected Result
用户权限查询返回通过全部角色获得的完整权限码集合，角色权限接口返回仓储提供的完整权限切片。