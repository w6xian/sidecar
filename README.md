# Sidecar Proxy

> 基于 WebSocket 长连接的内网穿透服务，借鉴服务网格 Sidecar 代理模式，将内网 HTTP/gRPC 服务安全地暴露给外部网络。

## 架构概览

```
┌──────────────────────────────────────────────────────────┐
│                    外部网络（公网）                         │
│                                                            │
│   手机小程序 / 浏览器 / 第三方系统                          │
│         │                                                  │
│         ▼                                                  │
│  ┌──────────────────────────────────┐                      │
│  │       服务端 (Server)             │  ← 部署于云服务器     │
│  │  - HTTP :8443 / gRPC :9443       │                      │
│  │  - 服务注册中心 & 路由分发        │                      │
│  │  - WebSocket 控制通道            │                      │
│  │  - 管理指令下发                   │                      │
│  └──────────┬───────────────────────┘                      │
│             │ WebSocket 长连接                              │
└─────────────┼──────────────────────────────────────────────┘
              │
┌─────────────┼──────────────────────────────────────────────┐
│             ▼        内网（局域网）                           │
│  ┌──────────────────────────────────┐                      │
│  │     Sidecar 客户端 (Client)       │  ← 部署于内网机器     │
│  │  - 服务注册 & 心跳保活            │                      │
│  │  - 请求接收 & 本地转发            │                      │
│  │  - 执行管理指令                   │                      │
│  └──────────┬───────────────────────┘                      │
│             │                                                  │
│             ▼                                                  │
│  ┌──────────────────────────────────┐                      │
│  │      内网业务服务                 │                      │
│  │  HTTP :8080 / gRPC :9090          │                      │
│  └──────────────────────────────────┘                      │
└──────────────────────────────────────────────────────────┘
```

**一句话说明**：Server 部署在公网，Sidecar 部署在内网，两者通过 WebSocket 长连接通信。外部请求打到 Server，Server 通过 WebSocket 转发到内网 Sidecar，Sidecar 再访问本地服务。

## 快速开始

### 编译

```bash
# 编译服务端和客户端
make all

# 或分别编译
make build-server    # → bin/server
make build-sidecar   # → bin/sidecar

# 多平台交叉编译
make release
```

### 运行

**1. 启动服务端（公网机器）**

```bash
./bin/server -config config/server.yaml
```

`config/server.yaml`：

```yaml
server:
  http_addr: ":8443"     # HTTP 网关端口
  grpc_addr: ":9443"     # gRPC 网关端口
  proxy_timeout: 30s     # 代理超时

auth:
  tokens:
    - "your-auth-token"  # 客户端连接 Token
  ip_whitelist: []       # IP 白名单，空则不限制

log:
  level: "info"          # debug / info / warn / error
```

**2. 启动 Sidecar 客户端（内网机器）**

```bash
./bin/sidecar -config config/sidecar.yaml
```

`config/sidecar.yaml`：

```yaml
sidecar:
  server_addr: "ws://your-server:8443/sidecar/connect"  # 生产环境用 wss://
  app_id: "your-app-id"                                  # 应用 ID（鉴权用）
  app_sec: "your-app-sec"                                # 应用密钥（鉴权用）
  app_sn: "your-sn"                                      # 序列号（鉴权用）
  reconnect_interval: 5s                                  # 重连基础间隔
  max_reconnect_interval: 60s                             # 最大重连间隔
  conn_alias: "my-machine"                               # 可选：连接别名

services:
  - id: "web-service"
    name: "Web 服务"
    protocol: "http"
    local_addr: "http://127.0.0.1:8080"
    expose_path: "/api/web"

  - id: "order-grpc"
    name: "订单 gRPC 服务"
    protocol: "grpc"
    local_addr: "127.0.0.1:9090"
    expose_path: "/grpc/order"

log:
  level: "info"
```

**3. 访问内网服务**

启动后，内网服务即可通过公网访问：

```bash
# 通过 expose_path 路由
curl http://your-server:8443/api/web/users

# 通过连接别名路由（任意路径透传）
curl http://your-server:8443/my-machine/any/path
```

### Docker 部署

```bash
docker-compose -f deploy/docker-compose.yml up -d
```

### 注册为系统服务（Windows/Linux）

```bash
./bin/sidecar install    # 注册开机自启服务
./bin/sidecar uninstall  # 卸载
```

## 路由方式

服务端支持两种路由方式，可同时使用：

| 路由方式 | 格式 | 说明 |
|---------|------|------|
| **服务路径路由** | `/{expose_path}/*` | 在配置中定义 `expose_path`，精确匹配路径前缀 |
| **连接别名路由** | `/{alias}/*` | 在配置中定义 `conn_alias`，任意路径透传 |

**路由优先级**（从高到低）：

1. `/sidecar/connect` — WebSocket 连接端点
2. `/health` — 健康检查
3. `/metrics` — Prometheus 监控指标
4. `/admin/*` — 管理 API
5. `/{alias}/*` — 连接别名路由
6. `/*` — 服务路径通配路由

## 管理 API

所有管理 API 均通过 HTTP 端口（默认 `:8443`）访问。

### 获取 connID

管理指令中的 `{connID}` 参数，可通过以下接口获取：

### 1. 查看所有在线 Sidecar 连接

```
GET /admin/sidecars
```

**响应示例**：

```json
{
  "total": 2,
  "sidecars": [
    {
      "conn_id": "550e8400-e29b-41d4-a716-446655440000",
      "alias": "my-machine",
      "services": [
        {
          "id": "web-service",
          "name": "Web 服务",
          "protocol": "http",
          "expose_path": "/api/web"
        }
      ]
    },
    {
      "conn_id": "660f9511-f30c-52e5-b827-557766551111",
      "alias": "",
      "services": [...]
    }
  ]
}
```

### 2. 查看已注册的服务路由

```
GET /admin/routes
```

**响应示例**：

```json
[
  {
    "service_id": "web-service",
    "name": "Web 服务",
    "protocol": "http",
    "expose_path": "/api/web",
    "sidecar_count": 1
  }
]
```

> 注意：此接口仅显示基于 `expose_path` 的服务路由，不显示连接别名路由。

### 3. 推送版本更新

```
POST /admin/cmd/update/{connID}
```

**功能**：向指定 Sidecar 推送更新包，客户端会下载新版本、校验 SHA256、替换二进制并自动重启。

**请求参数**（JSON Body）：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `version` | string | 是 | 目标版本号，如 `"1.2.3"` |
| `down_url` | string | 是 | 新版本下载地址 |
| `checksum` | string | 否 | 文件 SHA256 校验值，不填则跳过校验 |
| `force` | bool | 否 | 是否强制更新（不等待空闲） |

**请求示例**：

```bash
curl -X POST http://localhost:8443/admin/cmd/update/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.2.3",
    "down_url": "https://releases.example.com/sidecar-linux-amd64",
    "checksum": "a1b2c3d4e5f6...",
    "force": false
  }'
```

**响应示例**：

```json
{
  "success": true,
  "message": "update applied, restarting",
  "version": "1.2.3"
}
```

**执行流程**：
1. 客户端下载新版本到临时文件
2. 可选：SHA256 校验文件完整性
3. 备份当前二进制 → 替换为新版本
4. 回告服务端成功 → 自动重启

### 4. 执行 Lua 脚本

```
POST /admin/cmd/exec-lua/{connID}
```

**功能**：在 Sidecar 目标机上执行 Lua 脚本，支持直接传入脚本内容或指定本地文件路径。

**请求参数**（JSON Body）：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `script` | string | 二选一 | Lua 脚本内容（直接传入） |
| `file` | string | 二选一 | 本地脚本文件路径（从 Sidecar 本地读取） |
| `timeout` | int | 否 | 执行超时秒数，默认 30s |
| `env` | object | 否 | 传入 Lua 的全局变量，如 `{"key": "value"}` |

**请求示例**：

```bash
# 直接传脚本内容
curl -X POST http://localhost:8443/admin/cmd/exec-lua/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "script": "print(\"hello from sidecar\")\nfor i=1,3 do print(i) end",
    "timeout": 10
  }'

# 执行本地脚本文件
curl -X POST http://localhost:8443/admin/cmd/exec-lua/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "file": "/opt/scripts/health_check.lua",
    "env": {"SERVICE_NAME": "web"},
    "timeout": 60
  }'
```

**响应示例**：

```json
{
  "success": true,
  "output": "hello from sidecar\n1\n2\n3\n",
  "error": "",
  "elapsed": 5
}
```

### 5. 上传文件（读取 Sidecar 本地文件）

```
POST /admin/cmd/upload-file/{connID}
```

**功能**：读取 Sidecar 目标机上的文件内容并回传到服务端。常用于远程读取日志文件。

**请求参数**（JSON Body）：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | 是 | 要读取的本地文件路径 |
| `max_bytes` | int64 | 否 | 最多读取字节数，默认读取全部（上限 32MB） |
| `tail` | int | 否 | 只读取末尾 N 行，0 表示读全部 |

**请求示例**：

```bash
# 读取日志文件最后 100 行
curl -X POST http://localhost:8443/admin/cmd/upload-file/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/var/log/app.log",
    "tail": 100
  }'

# 读取文件前 1MB
curl -X POST http://localhost:8443/admin/cmd/upload-file/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/etc/config.yaml",
    "max_bytes": 1048576
  }'
```

**响应示例**：

```json
{
  "success": true,
  "path": "/var/log/app.log",
  "data": "base64-encoded-file-content...",
  "size": 4096,
  "error": "",
  "truncated": false
}
```

### 6. 写入文件（远程写文件到 Sidecar）

```
POST /admin/cmd/write-file/{connID}
```

**功能**：将文件内容写入 Sidecar 目标机的指定路径。支持大文件分块传输，服务端自动分块，客户端按序写入并逐块确认。

**请求参数**（JSON Body）：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | 是 | 目标文件路径（Sidecar 本地） |
| `data` | string | 是 | 文件完整内容（**base64 编码**） |
| `perm` | uint32 | 否 | 文件权限，默认 0644（十进制 420） |
| `chunk_size` | int | 否 | 每块字节数，默认 256KB |

**请求示例**：

```bash
# 写入配置文件
curl -X POST http://localhost:8443/admin/cmd/write-file/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/etc/myapp/config.yaml",
    "data": "'"$(base64 -w 0 config.yaml)"'",
    "perm": 420
  }'
```

> **注意**：`data` 字段必须是 base64 编码的完整文件内容。

**响应示例**：

```json
{
  "success": true,
  "path": "/etc/myapp/config.yaml",
  "bytes_written": 1024,
  "chunks": 4,
  "error": ""
}
```

**执行流程**：
1. 服务端将文件按 `chunk_size` 分块
2. 逐块发送到客户端，每块等待 ACK 确认后才发下一块（串行保序）
3. 第一块（seq=0）创建文件并自动创建目录
4. 最后一块写入后关闭文件，回传总字节数

### 7. 执行命令（远程执行可执行文件）

```
POST /admin/cmd/exec/{connID}
```

**功能**：在 Sidecar 目标机上执行指定的可执行文件或系统命令，支持传参、环境变量注入、工作目录指定、超时控制和 stdin 输入。

**请求参数**（JSON Body）：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `command` | string | 是 | 可执行文件路径或命令名 |
| `args` | string[] | 否 | 命令参数列表 |
| `env` | object | 否 | 环境变量（追加到系统环境变量），如 `{"KEY": "VALUE"}` |
| `dir` | string | 否 | 工作目录，空则使用当前目录 |
| `timeout` | int | 否 | 超时秒数，默认 60s，超时自动 kill 进程 |
| `stdin` | string | 否 | 标准输入内容 |

**请求示例**：

```bash
# 执行系统命令
curl -X POST http://localhost:8443/admin/cmd/exec/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "command": "ls",
    "args": ["-la", "/tmp"],
    "timeout": 10
  }'

# 执行自定义程序并传参
curl -X POST http://localhost:8443/admin/cmd/exec/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "command": "/usr/local/bin/myapp",
    "args": ["--port", "8080", "--mode", "prod"],
    "env": {"APP_ENV": "production", "DB_HOST": "localhost"},
    "dir": "/opt/myapp",
    "timeout": 30
  }'

# 带标准输入
curl -X POST http://localhost:8443/admin/cmd/exec/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "command": "cat",
    "stdin": "hello world",
    "timeout": 5
  }'
```

**响应示例**：

```json
{
  "success": true,
  "output": "total 64\ndrwxrwxr-x  2 root root 4096 Jan 1 00:00 .\n...",
  "error": "",
  "exit_code": 0,
  "elapsed": 12
}
```

**错误响应示例**：

```json
{
  "success": false,
  "output": "",
  "error": "execution timeout after 30s",
  "exit_code": -1,
  "elapsed": 30045
}
```

### 8. 健康检查

```
GET /health
```

**响应示例**：

```json
{
  "status": "healthy",
  "time": "2026-04-01T06:00:00Z"
}
```

### 9. Prometheus 监控指标

```
GET /metrics
```

暴露 Prometheus 格式的监控指标，可配合 Grafana 仪表盘使用。

## 错误码说明

管理 API 的 HTTP 错误码：

| 状态码 | 含义 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 404 | 连接不存在 / 服务未找到 |
| 500 | 命令执行失败（如写入文件失败） |
| 502 | Sidecar 连接异常或通信超时 |
| 504 | 代理请求超时 |

## WebSocket 消息协议

所有 WebSocket 消息使用 JSON 编码，通用信封格式：

```json
{
  "type": "MESSAGE_TYPE",
  "request_id": "uuid-xxxx",
  "app_id": "your-app-id",
  "sign": "base64-signature",
  "timestamp": 1712000000,
  "payload": { ... }
}
```

### 消息类型一览

| 类型 | 方向 | 说明 |
|------|------|------|
| `REGISTER` | Local → Server | 注册服务 |
| `REGISTER_ACK` | Server → Local | 注册确认 |
| `PROXY_REQUEST` | Server → Local | 转发请求 |
| `PROXY_RESPONSE` | Local → Server | 转发响应 |
| `HEARTBEAT` | 双向 | 心跳保活（每 30s） |
| `UNREGISTER` | Local → Server | 注销服务 |
| `ERROR` | 双向 | 错误通知 |
| `UPDATE` | Server → Local | 推送更新指令 |
| `UPDATE_ACK` | Local → Server | 更新执行结果 |
| `EXEC_LUA` | Server → Local | 执行 Lua 脚本 |
| `EXEC_LUA_RESULT` | Local → Server | Lua 执行结果 |
| `UPLOAD_FILE` | Server → Local | 请求上传文件 |
| `UPLOAD_FILE_DATA` | Local → Server | 文件数据 |
| `WRITE_FILE_CHUNK` | Server → Local | 写文件分块 |
| `WRITE_FILE_ACK` | Local → Server | 写文件确认 |
| `EXEC_CMD` | Server → Local | 执行命令 |
| `EXEC_CMD_RESULT` | Local → Server | 命令执行结果 |

## 可观测性

### 链路追踪

每个请求自动注入 `X-Trace-Id` Header，贯穿整个调用链（外部 → Server → Sidecar → 内网服务），方便排查问题。

### Prometheus 指标

主要指标：

- `connected_sidecars` — 在线 Sidecar 连接数
- `registered_services` — 已注册服务数
- `proxy_request_duration_seconds` — 代理请求耗时（Histogram）
- `proxy_requests_total` — 代理请求总数（Counter）
- `heartbeat_total` — 心跳计数
- `websocket_errors` — WebSocket 错误数

### 日志

- 使用 `uber-go/zap` 结构化日志
- 支持文件轮转（`lumberjack`）
- 级别可配置：`debug` / `info` / `warn` / `error`

## 技术栈

| 模块 | 技术 |
|------|------|
| 语言 | Go 1.22+ |
| WebSocket | gorilla/websocket |
| HTTP 路由 | chi v5 |
| gRPC | google.golang.org/grpc |
| 配置管理 | spf13/viper |
| 日志 | uber-go/zap + lumberjack |
| 监控 | prometheus/client_golang |
| CLI | spf13/cobra |
| Lua 执行 | w6xian/gua |

## 目录结构

```
sidecar/
├── cmd/
│   ├── server/              # 服务端入口
│   │   ├── main.go
│   └── root.go
│   └── sidecar/             # 客户端入口
│       ├── main.go
│       ├── root.go
│       └── install.go       # 系统服务注册
├── internal/
│   ├── server/              # 服务端核心
│   │   ├── gateway.go       # HTTP/gRPC 入口网关 & 管理 API
│   │   ├── handler.go       # WebSocket 连接处理 & 指令分发
│   │   ├── hub.go           # 连接管理（连接池、别名映射）
│   │   └── router.go        # 服务路由表（路径匹配、负载均衡）
│   ├── sidecar/             # 客户端核心
│   │   ├── client.go        # WebSocket 客户端 & 消息循环
│   │   ├── forwarder.go     # 本地请求转发（HTTP/gRPC）
│   │   ├── register.go      # 服务注册
│   │   ├── cmd_update.go    # 更新指令处理
│   │   ├── cmd_lua.go       # Lua 脚本执行
│   │   ├── cmd_upload.go    # 文件上传（读取本地文件）
│   │   ├── cmd_writefile.go # 文件写入（分块传输）
│   │   └── cmd_exec.go      # 命令执行
│   └── protocol/            # WebSocket 消息协议定义
│       ├── types.go         # 消息类型枚举
│       └── message.go       # 消息载荷结构体
├── config/
│   ├── server.yaml          # 服务端配置
│   └── sidecar.yaml         # 客户端配置
├── deploy/
│   ├── Dockerfile.server
│   ├── Dockerfile.sidecar
│   ├── docker-compose.yml
│   └── prometheus.yml
├── pkg/
│   ├── auth/                # 鉴权模块
│   ├── logger/              # 日志封装
│   └── metrics/             # Prometheus 指标
├── Makefile
├── go.mod
└── go.sum
```

## License

Private
