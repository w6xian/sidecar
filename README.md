# Sidecar Proxy — 内网穿透服务

基于 WebSocket 长连接的 Sidecar 模式内网穿透服务，用 Go 1.22+ 实现。

## 架构概览

```
外部客户端  ─── HTTP/gRPC ──→  Server Envoy（公网）
                                     │  WebSocket 长连接
                               Local Sidecar（内网）
                                     │  本地调用
                               内网业务服务
```

## 项目结构

```
sidecar/
├── cmd/
│   ├── server/main.go        # 服务端入口
│   └── sidecar/main.go       # 客户端入口
├── internal/
│   ├── protocol/             # 消息协议定义
│   │   ├── types.go
│   │   └── message.go
│   ├── server/               # 服务端核心
│   │   ├── hub.go            # WebSocket 连接管理
│   │   ├── router.go         # 服务路由表
│   │   ├── handler.go        # 请求分发处理
│   │   └── gateway.go        # HTTP/gRPC 入口网关
│   └── sidecar/              # 客户端核心
│       ├── client.go         # WebSocket 客户端 + 心跳 + 重连
│       ├── register.go       # 服务注册/注销
│       └── forwarder.go      # 本地请求转发
├── config/
│   ├── server.yaml           # 服务端配置
│   └── sidecar.yaml          # 客户端配置
├── pkg/
│   ├── auth/auth.go          # Token 鉴权 + IP 白名单
│   ├── logger/logger.go      # zap 结构化日志
│   └── metrics/metrics.go   # Prometheus 指标
├── deploy/
│   ├── Dockerfile.server
│   ├── Dockerfile.sidecar
│   ├── docker-compose.yml
│   └── prometheus.yml
├── go.mod
├── go.sum
└── Makefile
```

## 快速开始

### 1. 编译

```bash
# 编译所有二进制
make all

# 仅编译服务端
make build-server

# 仅编译客户端
make build-sidecar
```

### 2. 配置服务端

编辑 `config/server.yaml`：

```yaml
server:
  http_addr: ":8443"
  grpc_addr: ":9443"
  proxy_timeout: 30s
auth:
  tokens:
    - "your-secret-token"
```

### 3. 启动服务端

```bash
./bin/server -config config/server.yaml
```

### 4. 配置 Sidecar 客户端

编辑 `config/sidecar.yaml`，指向你的本地服务：

```yaml
sidecar:
  server_addr: "ws://your-server.com:8443/sidecar/connect"
  token: "your-secret-token"
services:
  - id: "user-service"
    protocol: "http"
    local_addr: "http://127.0.0.1:8080"
    expose_path: "/api/user"
```

### 5. 启动客户端

```bash
./bin/sidecar -config config/sidecar.yaml
```

### 6. 访问内网服务

```bash
curl http://your-server.com:8443/api/user/profile?id=123
```

## Docker Compose 部署

```bash
cd deploy
docker compose up -d
```

## API 端点

| 端点 | 说明 |
|------|------|
| `GET /health` | 健康检查 |
| `GET /metrics` | Prometheus 指标 |
| `GET /admin/routes` | 当前注册的服务路由 |
| `WS /sidecar/connect` | Sidecar WebSocket 连接端点 |
| `/* `（其他路径）| 代理转发到匹配的内网服务 |

## 消息协议

WebSocket 消息为 JSON 格式，类型包括：

| 类型 | 方向 | 说明 |
|------|------|------|
| `REGISTER` | Client → Server | 注册服务 |
| `REGISTER_ACK` | Server → Client | 注册确认 |
| `PROXY_REQUEST` | Server → Client | 转发请求 |
| `PROXY_RESPONSE` | Client → Server | 转发响应 |
| `HEARTBEAT` | 双向 | 心跳保活（30s） |
| `UNREGISTER` | Client → Server | 注销服务 |
| `ERROR` | 双向 | 错误通知 |

## 技术栈

- **语言**：Go 1.22+
- **WebSocket**：gorilla/websocket
- **HTTP 路由**：chi v5
- **gRPC**：google.golang.org/grpc
- **配置**：viper
- **日志**：uber-go/zap（结构化 JSON）
- **监控**：Prometheus client_golang
