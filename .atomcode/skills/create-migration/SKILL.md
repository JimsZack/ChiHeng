---
name: create-migration
description: 为 ChiHeng 项目创建新的 SQLite 数据库结构变更。当需要修改数据库结构（新增表、加列、加索引）时使用。遵循 internal/schema 的版本化管理约定。
disable_model_invocation: true
---

# 创建数据库结构变更

本项目数据库结构位于 `internal/schema/`，约定如下：

## 现有结构

- `internal/schema/init.sql` — 初始建表 SQL（`CREATE TABLE IF NOT EXISTS` 幂等语句）
- `internal/schema/schema.go` — 通过 `//go:embed init.sql` 内嵌 SQL
- `SchemaVersion` 常量 — 当前结构版本号（如 `"0001"`）
- `schema_versions` 表 — 记录已应用的结构版本

## 新增变更步骤

1. **确定版本号**：查看 `internal/schema/schema.go` 中 `SchemaVersion` 常量，新版本号 = 当前 + 1（补零到 4 位，如 `0002`）。
2. **更新 SQL 文件**：在 `internal/schema/init.sql` 中追加新语句，使用 `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE ... ADD COLUMN` 等幂等语句。
3. **嵌入并应用**：在 `internal/schema/schema.go` 中：
   - 追加对应的 SQL 语句到 `EnsureSchema` 的初始化流程
   - 更新 `SchemaVersion` 为新版本号
   - 向 `schema_versions` 插入新版本记录
4. **金额/份额字段约定**：金额用 `*_cents INTEGER`，份额用 `*_micros INTEGER`，净值用 `*_nav_micros INTEGER`（见 `internal/domain/money.go` 的 Scale 常量）。
5. **时间字段**：一律 TEXT 存 `time.RFC3339` 格式。
6. **外键**：开启 `PRAGMA foreign_keys = ON`（`EnsureSchema` 已处理），引用用 `REFERENCES 表(id)`。
7. **测试**：运行 `go test ./internal/...` 验证，确认既有测试不受影响。
