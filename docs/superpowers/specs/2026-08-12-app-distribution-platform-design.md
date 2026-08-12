# App 分发平台 设计文档

日期：2026-08-12
状态：已确认

## 1. 概述

面向个人/小团队的自建 app 分发平台，类似简化版蒲公英/fir.im。开发者上传安装包，测试/同事通过链接或二维码下载安装。

**核心定位：** 极简、单二进制一体式部署、存储可切换（本地 / 腾讯云 COS）。

**支持平台：** Android (APK)、iOS (IPA，仅文件下载)、桌面端 (EXE/DMG)。

## 2. 技术选型

| 层 | 选型 |
|----|------|
| 后端语言 | Go 1.24 |
| Web 框架 | 标准库 `net/http`（自定义中间件封装，见 §10） |
| ORM | GORM |
| 数据库 | `github.com/libtnb/sqlite`（基于 modernc.org/sqlite，纯 Go 无 CGO） |
| 文件存储 | Storage 接口抽象：LocalStorage / COSStorage |
| 对象存储 SDK | 腾讯云 cos-go-sdk |
| 前端 | Vue 3 + Vite + TypeScript（现有脚手架），vue-router，axios/fetch |
| 部署 | 单二进制一体式：后端内嵌编译后的 Vue 静态文件（go:embed） |

## 3. 总体架构

```
┌─────────────── 单二进制 ───────────────┐
│  Go 后端 (net/http)                     │
│  ├── REST API                          │
│  ├── 内嵌 Vue 静态前端 (go:embed)        │
│  ├── GORM + libtnb/sqlite              │
│  └── Storage 接口                       │
│       ├── LocalStorage (本地磁盘)        │
│       └── COSStorage (腾讯云 COS)       │
└────────────────────────────────────────┘
```

部署形态：后端服务一个进程监听端口，同时提供 API 与静态前端；数据库为单个 sqlite 文件；本地存储为配置目录下文件。

## 4. 数据模型（GORM Model）

### User（管理员）
- `id` int
- `username` string，唯一
- `password_hash` string（sha256 + salt）
- `created_at` time

### App（应用）
- `id` int
- `name` string
- `icon` string（可选，图标 URL）
- `description` string
- `created_at` time

### Channel（渠道）
- `id` int
- `app_id` int，外键
- `name` string（如 test / release）
- `created_at` time

### Version（版本 / 安装包）
- `id` int
- `app_id` int，外键
- `channel_id` int，外键
- `version_name` string（如 1.0.0）
- `version_code` int（如 100）
- `file_type` string（apk / ipa / exe / dmg）
- `file_name` string（原始文件名）
- `file_size` int64
- `storage_key` string（存储 key，规则见 §5）
- `storage_backend` string（local / cos，记录实际存储后端）
- `sha256` string（上传时计算，供客户端校验）
- `changelog` text（更新日志）
- `access_mode` string（public / password / expiry）
- `password_hash` string（access_mode=password 时的密码哈希）
- `expires_at` time（access_mode=expiry 时的过期时间）
- `enabled` bool（false 表示下架）
- `download_count` int
- `install_count` int
- `created_at` time

### DownloadLog（下载记录）
- `id` int
- `version_id` int，外键
- `ip` string
- `user_agent` string
- `created_at` time

## 5. 存储抽象

### Storage 接口

```go
type Storage interface {
    Save(ctx, key string, r io.Reader) (size int64, err error)   // 写文件
    Open(ctx, key string) (io.ReadCloser, error)                 // 读文件（本地代理用）
    Delete(ctx, key string) error                                // 删文件
    DownloadURL(ctx, key string, filename string, expire time.Duration) (string, error)
    // 返回下载 URL：本地 → 后端代理路径；COS → 预签名 URL
}
```

### LocalStorage
- 文件存于配置目录（默认 `./data/files`）
- `DownloadURL` 返回后端路由 `/api/v1/files/{key}`，由后端 `Open` 流式输出

### COSStorage
- 使用腾讯云 `cos-go-sdk`
- 上传：后端流式 PUT 到 COS
- `DownloadURL` 返回预签名 URL（有效期内直连下载，不占服务器带宽）

### 文件 key 规则
`{app_id}/{version_id}/{original_filename}`，上传时由后端统一生成并写入 `Version.storage_key`，同时计算 `sha256` 入库。

## 6. 配置（JSON）

配置文件默认路径 `config.json`，也可用环境变量覆盖关键项。

```json
{
  "server": {
    "addr": ":8080"
  },
  "database": {
    "dsn": "./data/app.db"
  },
  "storage": {
    "backend": "local",
    "local": { "dir": "./data/files" },
    "cos": {
      "secret_id": "...",
      "secret_key": "...",
      "bucket": "app-dist-xxxx",
      "region": "ap-guangzhou",
      "base_url": "https://app-dist-xxxx.cos.ap-guangzhou.myqcloud.com"
    }
  },
  "jwt": {
    "secret": "...",
    "expire": "24h"
  }
}
```

## 7. API 设计

REST API，前缀 `/api/v1`。管理员接口需 `Authorization: Bearer <token>`（JWT）。

### 认证
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/login` | 账号密码登录，返回 JWT |

### 管理端（需登录）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/admin/apps` | 应用列表 / 创建应用 |
| PUT/DELETE | `/admin/apps/{id}` | 修改 / 删除应用 |
| GET/POST | `/admin/channels` | 渠道列表 / 创建渠道 |
| POST | `/admin/versions` | 上传安装包（multipart：文件 + 版本信息 + 访问控制） |
| PUT | `/admin/versions/{id}` | 更新版本信息（changelog、访问模式、下架等） |
| DELETE | `/admin/versions/{id}` | 删除版本（可选连带删存储文件） |
| GET | `/admin/versions/{id}/stats` | 下载/安装统计 |

### 公开/下载端
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/apps` | 应用列表（含最新版本摘要） |
| GET | `/apps/{id}` | 应用详情 + 版本列表 |
| POST | `/versions/{id}/verify` | 校验访问权限（密码模式提交密码） |
| GET | `/versions/{id}/install` | 安装上报 + 返回下载 URL（内部校验访问控制） |
| GET | `/versions/{id}/download` | 返回下载 URL（记 download_count） |
| GET | `/files/{key}` | 本地存储文件流式代理（COS 场景不用） |

### 下载流程
1. 用户访问下载页 → 前端根据 `access_mode` 渲染（public 直接下载按钮 / password 弹窗 / expiry 过期提示）
2. 下载请求到达后端 → 服务端校验访问控制（密码、有效期、下架状态），不信任前端
3. 校验通过 → 根据 `storage_backend` 返回本地代理 URL 或 COS 预签名 URL
4. 下载计数 +1，写 `DownloadLog`
5. 安装统计：下载页「安装」按钮先调 `/install`（install_count +1）再触发下载

## 8. 前端

Vue 3 单应用，`vue-router` 路由分两块。

### 公开区（无需登录）
- `/` — 应用列表看板（图标 + 名称 + 最新版本）
- `/app/:id` — 应用详情：渠道 → 版本列表（版本号、更新日志、大小、下载按钮）
- 下载交互：public 直接下载；password 弹窗校验；expiry 过期提示
- 桌面端/APK 显示二维码（扫码安装）+ 直接下载按钮

### 管理区（需登录）
- `/login` — 登录页
- `/admin` — 应用管理：新建/编辑/删除应用
- `/admin/app/:id` — 渠道管理 + 版本列表（上传、编辑、下架、删除、看统计）
- `/admin/upload` — 上传版本表单（文件 + 版本号 + changelog + 访问模式/密码/有效期 + 渠道）

### 技术栈
现有 Vite + Vue 3 + TS，加 `vue-router` + axios/fetch。管理区用路由守卫检查 JWT。开发时 Vite proxy 把 `/api` 转发到后端。

## 9. 安全

- 管理员密码、下载密码均用 **sha256 + salt** 哈希存储（小团队内网工具，不引入 bcrypt 依赖）
- 管理接口 JWT 认证
- 下载访问控制服务端强制校验（密码、有效期、下架状态）
- 本地存储路径防目录穿越：key 白名单校验（只允许 `数字/数字/文件名` 结构）
- 上传计算 `sha256` 入库，下载暴露给客户端校验完整性
- 密码验证接口简单限流，登录失败记录

## 10. 错误处理与 Web 封装

使用标准库 `net/http`，配合统一的中间件与响应封装。

### 响应封装

统一 JSON 响应结构：`{ "code": 0, "msg": "ok", "data": ... }`（`code` 为业务码，`0` 表示成功）。

```go
type Middleware func(http.HandlerFunc) http.HandlerFunc

func Chain(mws ...Middleware) Middleware {
    return func(next http.HandlerFunc) http.HandlerFunc {
        for i := len(mws) - 1; i >= 0; i-- {
            next = mws[i](next)
        }
        return next
    }
}

func SendJson(w http.ResponseWriter, data any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    enc := json.NewEncoder(w)
    enc.Encode(map[string]any{
        "code": 0,
        "msg":  "ok",
        "data": data,
    })
}

func SendError(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    enc := json.NewEncoder(w)
    enc.Encode(map[string]any{
        "code": code,
        "msg":  msg,
    })
}

func SendStatus(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(code)
    enc := json.NewEncoder(w)
    enc.Encode(map[string]any{
        "code": code,
        "msg":  msg,
    })
}
```

### 中间件

- 请求日志中间件、recovery（panic 恢复）中间件、JWT 认证中间件，均用 `Middleware` + `Chain` 组合
- 业务错误用 `SendError` 返回业务码（HTTP 200），HTTP 状态错误（如 404/500）用 `SendStatus` 返回真实状态码
- 存储错误映射为对应业务码/状态码

## 11. 测试

- Storage 接口两个实现（local/cos）单元测试（COS 用 mock SDK）
- 访问控制逻辑单元测试（密码/有效期/下架）
- Handler 集成测试：sqlite 内存库跑 上传 → 列表 → 下载 全流程
