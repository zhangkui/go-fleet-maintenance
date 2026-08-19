# BUG-004 Reproduction

## Symptom
新用户注册成功后没有获得系统默认 operator 角色，导致后续读取 /api/me 时缺少基础角色。

## Trigger

    go test ./internal/service -count=1 -run '^TestBug004_RegistrationAssignsOperatorRole$'

## Defective Result

    repository_finds_role_by_code: actual SQL uses WHERE name=?; expected WHERE code=?
    requested role="daily_operator"
    FAIL

## Expected Result
注册流程使用标准 operator 角色码查询既有角色数据，并为新用户建立角色关联；用户创建、注册校验和其他角色不受影响。