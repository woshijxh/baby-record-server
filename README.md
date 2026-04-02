# 宝宝记录小程序后端

这是一个基于 `Gin + GORM + MySQL` 的宝宝记录服务，提供微信小程序首版所需的登录、家庭、宝宝档案、记录和统计接口。

## 本地运行

1. 复制环境变量：

```bash
cp .env.example .env
```

2. 创建数据库并执行 [sql/001_init.sql](/Users/jiaoxinghui/Documents/codex/baby-record-server/sql/001_init.sql)，或直接使用 `AUTO_MIGRATE=true` 让服务自动建表。

3. 启动服务：

```bash
go run ./cmd/server
```

默认地址是 `http://127.0.0.1:8080`。

## 本地微信登录

开发期默认 `WECHAT_MOCK=true`，后端会根据 `wx.login` 返回的 code 生成一个稳定的 mock `openid`，这样即使没配微信正式 `AppID/Secret` 也能联调。

## 主要接口

- `POST /api/auth/wx-login`
- `POST /api/family`
- `POST /api/family/join`
- `GET /api/family/current`
- `GET /api/babies/current`
- `POST /api/babies`
- `PATCH /api/babies/:id`
- `GET /api/dashboard`
- `GET /api/records`
- `POST /api/records`
- `PATCH /api/records/:id`
- `DELETE /api/records/:id`
- `GET /api/stats`
