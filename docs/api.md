# API 资源说明

## 认证

- `POST /api/auth/register` — 注册，默认 operator 角色
- `POST /api/auth/login` — 登录，返回 access_token + refresh_token
- `POST /api/auth/refresh` — 刷新令牌（刷新令牌轮转，旧令牌撤销）
- `POST /api/auth/logout` — 撤销刷新令牌
- `GET /api/me` — 当前用户 + 角色 + 权限码
- `POST /api/me/password` — 修改密码（撤销全部刷新令牌）
- `POST /api/users/{id}/reset-password` — 管理员重置密码

## RBAC

- `GET /api/roles` — 列出全部角色
- `POST /api/roles` — 创建角色（name, code, description）
- `GET /api/permissions` — 列出全部权限
- `GET /api/roles/{id}/permissions`、`GET /api/users/{id}/roles`
- `POST /api/users/{id}/roles` — 给用户分配角色
- `DELETE /api/users/{id}/roles/{role_id}` — 撤销用户角色
- `POST /api/roles/{id}/permissions` — 给角色授予权限

## 车辆

- `POST /api/vehicles`、`GET /api/vehicles`、`GET /api/vehicles/{id}`
- `POST /api/vehicles/{id}/status`（状态流转）、`POST /api/vehicles/{id}/mileage`（里程单调递增）
- `GET /api/vehicles/{id}/status-history`、`POST|GET /api/vehicles/{id}/licenses`

## 司机

- `POST|GET /api/drivers`、`GET /api/drivers/{id}`、`POST /api/drivers/{id}/status`
- `POST /api/drivers/bindings`、`GET /api/vehicles/{id}/bindings`（司机-车辆绑定）
- `POST /api/drivers/schedules`、`GET /api/drivers/{id}/schedules`（排班）
- `POST|GET /api/drivers/{id}/violations`（违章）

## 出车任务

- `POST|GET /api/trips`、`GET /api/trips/{id}`（创建需 `idempotency_key`）
- `POST /api/trips/{id}/start|complete|cancel`（状态流转，完成需事务更新车辆里程）
- `POST /api/trips/{id}/handovers`（交接，需 from_driver_id + to_driver_id）

## 油耗

- `POST /api/fuel-records`（幂等、异常识别、事务更新里程）、`GET /api/fuel-records`、`GET /api/fuel-records/{id}`

## 维保

- `POST|GET /api/maintenance/policies`（里程+日期双条件计划，到期按 join vehicles 当前里程判定）
- `POST|GET /api/maintenance/orders`、`GET /api/maintenance/orders/{id}`（创建需 `idempotency_key`，可带 parts 明细扣减库存）
- `POST /api/maintenance/orders/{id}/status`（状态流转）、`POST /api/maintenance/orders/{id}/complete`（完成工单、恢复车辆在用态）
- `POST /api/maintenance/trigger-due`（扫描到期计划幂等生成工单）

## 配件

- `POST|GET /api/parts`、`GET /api/parts/low-stock`
- `POST /api/parts/{id}/adjust`（行锁库存调整）、`GET /api/parts/{id}/movements`

## 提醒与报表

- `POST /api/reminders/scan`、`GET /api/reminders`
- `GET /api/reports/fleet-summary|vehicle-utilization|fuel-efficiency|maintenance-cost`
- `GET /api/audit-logs`

## 通用约定

- 分页：`limit`（默认 20，最大 100）、`offset`；响应 `{items,total,limit,page}`
- 过滤：`status`、`keyword`、`vehicle_id`、`driver_id`、`from`、`to`、`abnormal`、`kind`
- 排序：`sort` 白名单字段、`order` asc/desc
- 金额：整数分；油量：整数毫升；里程：整数公里
- 时间：RFC3339，时区 Asia/Shanghai
- 错误：`{"code","message","details"}`，message 为具体原因（如"幂等键不能为空"、"新密码长度不能少于 8 位"），状态码 `400/401/403/404/409/429/500`
- 空列表返回 `[]` 而非 `null`
