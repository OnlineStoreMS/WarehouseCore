# WarehouseCore（仓储中心）

OSMS 平台库存管理中心：仓配商品主档、仓库货位、库存账、盘点、调拨、其他出入库。

| 项 | 值 |
|----|-----|
| Go module | `warehousecore` |
| API | `:8095` |
| Web | `:5180` |
| Docker 镜像 | `warehousecore-api`、`warehousecore-web` |
| UserCore app | `warehousecore`（`warehouse:read` / `warehouse:write`） |
| 设计文档 | [ProductCore/docs/WMS_DESIGN.md](../ProductCore/docs/WMS_DESIGN.md) |
| 平台编排 | `/home/asialeaf/projects/deploy` |

## 本地开发

```bash
# API（可用 configs/config.local.yaml + SQLite）
cd WarehouseCore
go run ./cmd/api -config configs/config.local.yaml

# Web
cd web && npm install && npm run dev
```

从 UserCore 应用中心进入时，走 `/auth/callback?token=...`。

## 数据库

```bash
make init-db APP_PASSWORD=你的密码
make fix-db-perms   # PG15+ public schema 权限修复
```

## Docker / ACR

推送 `main` 后 GitHub Actions 构建并推送：

- `registry.cn-hangzhou.aliyuncs.com/<ns>/warehousecore-api:latest`
- `registry.cn-hangzhou.aliyuncs.com/<ns>/warehousecore-web:latest`

服务器侧：

```bash
cd ~/projects/deploy
make pull-images
docker compose -f docker-compose.yml -f docker-compose.acr.yml up -d warehousecore-api warehousecore-web --no-build
```
