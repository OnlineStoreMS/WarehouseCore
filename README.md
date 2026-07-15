# WarehouseCore（仓储中心）

OSMS 平台库存管理中心：仓配商品主档、仓库货位、库存账、盘点、调拨、其他出入库。

| 项 | 值 |
|----|-----|
| Go module | `warehousecore` |
| API | `:8095` |
| Web | `:5180` |
| UserCore app | `warehousecore`（`warehouse:read` / `warehouse:write`） |
| 设计文档 | [ProductCore/docs/WMS_DESIGN.md](../ProductCore/docs/WMS_DESIGN.md) |

## 本地开发

```bash
# API（可用 configs/config.local.yaml + SQLite）
cd WarehouseCore
go run ./cmd/api -config configs/config.local.yaml

# Web
cd web && npm install && npm run dev
```

从 UserCore 应用中心进入时，走 `/auth/callback?token=...`。
