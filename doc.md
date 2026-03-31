# Sidecar 模式内网穿透服务 需求文档

> **版本**：v1.0  
> **日期**：2026-03-30  
> **语言**：Go  
> **状态**：草稿

---

## 1. 项目背景

在实际业务场景中，开发者或企业内部经常需要将内网（局域网）中运行的服务（HTTP、gRPC 等）暴露给外部客户端访问（如手机小程序、第三方系统等）。传统方案（如 frp、ngrok）通常是通用工具，缺乏灵活的协议扩展能力与企业级管控能力。

本项目借鉴 **Sidecar 代理模式**（源自服务网格架构），在本地部署一个轻量级 Sidecar Proxy（基于 Envoy 思想、Go 自研实现），通过与云端控制节点建立 **WebSocket 长连接**，实现内外网服务的透明转发与统一管控。

---

## 2. 核心目标

| 目标 | 描述 |
|------|------|
| 内网服务外部可达 | 手机小程序等外部客户端可通过公网 URL 访问内网 HTTP/gRPC 服务 |
| 协议无关 | 支持 HTTP、gRPC（HTTP/2）及未来扩展（TCP 流等） |
| 零侵入 | 内网服务无需改造，Sidecar 以旁路方式运行 |
| 集中管控 | 服务端 Envoy 统一管理所有已注册的内网 Sidecar |
| 安全可靠 | 连接鉴权、传输加密、访问控制 |

---

## 3. 系统架构

### 3.1 整体架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                         外部网络（公网）                           │
│                                                                    │
│   手机小程序 / 浏览器 / 第三方系统                                  │
│         │  HTTP / gRPC 请求                                        │
│         ▼                                                          │
│  ┌─────────────────────────────────┐                               │
│  │        服务端 Envoy (Server)     │  ← 公网可达，部署于云服务器    │
│  │  - HTTP/gRPC 入口网关            │                               │
│  │  - 服务注册中心                  │                               │
│  │  - WebSocket 控制通道管理        │                               │
│  │  - 请求路由 & 负载均衡           │                               │
│  └─────────────┬───────────────────┘                               │
│                │ WebSocket 长连接（控制 + 数据复用）                 │
└────────────────┼─────────────────────────────────────────────────┘
                 │
┌────────────────┼─────────────────────────────────────────────────┐
│                │           内网（局域网）                           │
│                ▼                                                    │
│  ┌─────────────────────────────────┐                               │
│  │      本地 Sidecar Proxy (Client) │  ← 内网部署，随服务启动       │
│  │  - WebSocket 客户端              │                               │
│  │  - 服务注册 & 心跳               │                               │
│  │  - 请求接收 & 本地转发           │                               │
│  │  - 响应回传                      │                               │
│  └─────────────┬───────────────────┘                               │
│                │ 本地 HTTP/gRPC 调用                                │
│                ▼                                                    │
│  ┌─────────────────────────────────┐                               │
│  │       内网业务服务               │                               │
│  │  - HTTP Service (e.g. :8080)    │                               │
│  │  - gRPC Service (e.g. :9090)    │                               │
│  └─────────────────────────────────┘                               │
└──────────────────────────────────────────────────────────────────┘
```

### 3.2 核心组件

| 组件 | 角色 | 部署位置 |
|------|------|---------|
| **Server Envoy**（服务端节点） | 公网入口、路由分发、连接管理 | 云服务器 |
| **Local Sidecar**（本地节点） | 内网代理、请求转发、服务注册 | 内网机器 |
| **配置文件**（config.yaml） | 定义本地服务列表、服务端地址、鉴权信息 | 本地 |

---

## 4. 详细功能需求

### 4.1 Local Sidecar（本地 Sidecar Proxy）

#### 4.1.1 服务注册与发现

- 启动时读取配置文件，解析本地服务列表
- 通过 WebSocket 连接服务端 Envoy
- 发送 **RegisterRequest** 消息，将本地服务元信息注册到服务端
- 支持服务心跳（默认每 30s 发送一次 Ping），断线自动重连（指数退避）

#### 4.1.2 请求接收与转发

- 接收服务端 Envoy 通过 WebSocket 下发的 **ProxyRequest** 消息
- 根据消息中的 `serviceId` + `protocol`，定位本地对应服务的地址和端口
- 在本地发起真实的 HTTP 或 gRPC 请求
- 将响应封装为 **ProxyResponse** 消息，通过 WebSocket 回传给服务端

#### 4.1.3 协议支持

| 协议 | 说明 |
|------|------|
| HTTP/HTTPS | 支持 GET/POST/PUT/DELETE/PATCH 等标准方法，透传 Headers/Body |
| gRPC（HTTP/2） | 支持 Unary RPC，流式 RPC（Server/Client/Bi-directional Streaming）扩展支持 |

#### 4.1.4 本地配置文件（config.yaml）

```yaml
sidecar:
  server_addr: "wss://your-server.com/sidecar/connect"   # 服务端 WebSocket 地址
  token: "your-auth-token"                                # 鉴权 Token
  reconnect_interval: 5s                                  # 重连间隔基础值
  max_reconnect_interval: 60s                             # 最大重连间隔

services:
  - id: "user-service"
    name: "用户服务"
    protocol: "http"
    local_addr: "http://127.0.0.1:8080"
    expose_path: "/api/user"                              # 服务端暴露路径前缀

  - id: "order-grpc"
    name: "订单 gRPC 服务"
    protocol: "grpc"
    local_addr: "127.0.0.1:9090"
    expose_path: "/grpc/order"
```

---

### 4.2 Server Envoy（服务端节点）

#### 4.2.1 WebSocket 控制通道

- 监听 WebSocket 端点（如 `/sidecar/connect`）
- 对每个连接进行 Token 鉴权
- 维护已连接的 Sidecar 列表及其注册的服务信息
- 支持并发多个 Sidecar 同时接入（连接池管理）

#### 4.2.2 HTTP 入口网关

- 监听公网 HTTP 端口（如 `:443` / `:8443`）
- 根据请求路径，匹配已注册的服务路由规则
- 生成唯一 **requestId**，封装为 **ProxyRequest** 消息
- 通过对应 Sidecar 的 WebSocket 连接下发请求
- 等待 **ProxyResponse**（设置超时，默认 30s）
- 将响应返回给外部调用方

#### 4.2.3 gRPC 入口网关

- 监听公网 gRPC 端口（如 `:9443`）
- 实现通用 gRPC 代理（基于 `grpc.UnknownServiceHandler`）
- 将 gRPC 帧序列化后通过 WebSocket 转发至对应 Sidecar
- 接收 Sidecar 返回的 gRPC 响应帧，回写给调用方

#### 4.2.4 服务路由管理

- 内存维护路由表（serviceId → Sidecar WebSocket 连接）
- Sidecar 断连时，自动从路由表移除相关服务
- 支持同一服务多个 Sidecar 注册（负载均衡，轮询策略）

---

### 4.3 消息协议设计

所有 WebSocket 消息使用 **JSON 或 Protobuf** 编码（默认 JSON，生产推荐 Protobuf）。

#### 消息类型枚举

| 类型 | 方向 | 说明 |
|------|------|------|
| `REGISTER` | Local → Server | 注册服务 |
| `REGISTER_ACK` | Server → Local | 注册确认 |
| `PROXY_REQUEST` | Server → Local | 转发请求 |
| `PROXY_RESPONSE` | Local → Server | 转发响应 |
| `HEARTBEAT` | 双向 | 心跳保活 |
| `UNREGISTER` | Local → Server | 注销服务 |
| `ERROR` | 双向 | 错误通知 |

#### RegisterRequest 结构

```json
{
  "type": "REGISTER",
  "token": "your-auth-token",
  "services": [
    {
      "id": "user-service",
      "name": "用户服务",
      "protocol": "http",
      "expose_path": "/api/user"
    }
  ]
}
```

#### ProxyRequest 结构（HTTP）

```json
{
  "type": "PROXY_REQUEST",
  "request_id": "uuid-xxxx",
  "service_id": "user-service",
  "protocol": "http",
  "http": {
    "method": "GET",
    "path": "/api/user/profile?id=123",
    "headers": {
      "Authorization": "Bearer xxx",
      "Content-Type": "application/json"
    },
    "body": ""
  }
}
```

#### ProxyResponse 结构（HTTP）

```json
{
  "type": "PROXY_RESPONSE",
  "request_id": "uuid-xxxx",
  "protocol": "http",
  "http": {
    "status_code": 200,
    "headers": {
      "Content-Type": "application/json"
    },
    "body": "{\"name\":\"张三\",\"age\":25}"
  }
}
```

---

## 5. 非功能需求

### 5.1 安全性

| 需求 | 描述 |
|------|------|
| 传输加密 | WebSocket 使用 WSS（TLS 1.2+） |
| 鉴权 | Sidecar 连接时携带 Token，服务端验证有效性（支持 JWT 扩展） |
| 访问控制 | 服务端支持白名单 IP 过滤，限制可注册的 Sidecar 来源 |
| 请求签名 | （可选）ProxyRequest 携带 HMAC 签名，防止中间人篡改 |

### 5.2 可靠性

| 需求 | 描述 |
|------|------|
| 断线重连 | 本地 Sidecar 断线后，自动指数退避重连 |
| 请求超时 | 服务端对每个 ProxyRequest 设置响应超时（默认 30s），超时返回 504 |
| 并发处理 | 服务端支持单个 Sidecar 连接上的请求并发处理（multiplexing） |
| 优雅退出 | Sidecar 关闭前发送 UNREGISTER 消息，服务端清理路由 |

### 5.3 性能

| 指标 | 目标 |
|------|------|
| 单连接并发请求数 | ≥ 100 并发（通过 requestId 多路复用） |
| 额外延迟（P99） | ≤ 50ms（不含内网服务本身耗时） |
| 内存占用（Sidecar） | ≤ 50MB |

### 5.4 可观测性

- 服务端提供 `/metrics` 接口（Prometheus 格式），暴露：
  - 已注册 Sidecar 数量
  - 各服务请求 QPS、错误率
  - WebSocket 连接状态
- 结构化日志（JSON 格式），支持日志级别动态调整
- 请求链路 TraceID 透传（X-Trace-Id Header）

---

## 6. 技术选型

| 模块 | 技术 | 说明 |
|------|------|------|
| 开发语言 | Go 1.22+ | 高并发，丰富的网络库 |
| WebSocket 库 | `gorilla/websocket` | 成熟稳定 |
| HTTP 框架 | `net/http` 标准库 + `chi` 路由 | 轻量、高性能 |
| gRPC | `google.golang.org/grpc` | 官方库 |
| 配置解析 | `viper` | 支持 YAML/ENV/动态配置 |
| 日志 | `uber-go/zap` | 高性能结构化日志 |
| 序列化 | `encoding/json` + 可选 `protobuf` | 开发阶段 JSON，生产 Protobuf |
| 构建 & 发布 | `Makefile` + `Docker` | 多平台交叉编译 |

---

## 7. 目录结构

```
sidecar-proxy/
├── cmd/
│   ├── server/          # 服务端 Envoy 入口
│   │   └── main.go
│   └── sidecar/         # 本地 Sidecar 入口
│       └── main.go
├── internal/
│   ├── server/          # 服务端核心逻辑
│   │   ├── gateway.go   # HTTP/gRPC 入口网关
│   │   ├── hub.go       # WebSocket 连接管理
│   │   ├── router.go    # 服务路由表
│   │   └── handler.go   # 请求分发处理
│   ├── sidecar/         # 本地 Sidecar 核心逻辑
│   │   ├── client.go    # WebSocket 客户端
│   │   ├── forwarder.go # 本地请求转发
│   │   └── register.go  # 服务注册
│   └── protocol/        # 消息协议定义
│       ├── message.go
│       └── types.go
├── config/
│   ├── server.yaml      # 服务端配置示例
│   └── sidecar.yaml     # 本地 Sidecar 配置示例
├── pkg/
│   ├── auth/            # 鉴权模块
│   ├── logger/          # 日志封装
│   └── metrics/         # 监控指标
├── deploy/
│   ├── Dockerfile.server
│   ├── Dockerfile.sidecar
│   └── docker-compose.yml
├── docs/
│   └── architecture.md
├── go.mod
├── go.sum
└── Makefile
```

---

## 8. 核心流程时序图

### 8.1 Sidecar 注册流程

```
Local Sidecar          Server Envoy
     │                      │
     │── WS Connect ────────▶│  (携带 Token)
     │                      │── 验证 Token
     │◀── WS Connected ──────│
     │                      │
     │── REGISTER ──────────▶│  (服务列表)
     │                      │── 写入路由表
     │◀── REGISTER_ACK ──────│
     │                      │
     │── HEARTBEAT(30s) ────▶│
     │◀── HEARTBEAT ─────────│
```

### 8.2 外部请求转发流程

```
外部客户端          Server Envoy         Local Sidecar       内网服务
    │                   │                    │                  │
    │── HTTP GET ───────▶│                   │                  │
    │                   │── 路由匹配          │                  │
    │                   │── PROXY_REQUEST ──▶│                  │
    │                   │                   │── HTTP Request ──▶│
    │                   │                   │◀── HTTP Response ─│
    │                   │◀── PROXY_RESPONSE ─│                  │
    │◀── HTTP 200 ───────│                   │                  │
```

---

## 9. 里程碑计划

| 阶段 | 内容 | 周期 |
|------|------|------|
| **M1** | 协议定义、WebSocket 连接管理、服务注册/心跳 | 第 1-2 周 |
| **M2** | HTTP 请求转发（完整链路跑通） | 第 3-4 周 |
| **M3** | gRPC 请求转发（Unary RPC） | 第 5-6 周 |
| **M4** | 鉴权、TLS、配置管理、日志 | 第 7-8 周 |
| **M5** | 监控指标、多 Sidecar 负载均衡、gRPC 流式支持 | 第 9-10 周 |
| **M6** | 性能压测、文档完善、Docker 镜像发布 | 第 11-12 周 |

---

## 10. 风险与注意事项

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| WebSocket 连接被中间网关断开 | 内网服务不可达 | 心跳保活 + 断线重连 + 代理配置说明 |
| gRPC 流式代理复杂度高 | 延期 | M3 优先 Unary，流式进入 M5 |
| 高并发下 requestId 匹配性能 | 吞吐下降 | 使用 sync.Map 或 channel map 管理 pending 请求 |
| Token 泄露风险 | 未授权访问 | Token 轮换机制 + HTTPS/WSS 强制 |
| 大 Body 传输内存占用 | OOM | 流式传输 + Body 大小限制（默认 32MB） |

---

## 11. 附录：参考项目

- [frp](https://github.com/fatedier/frp) — 内网穿透参考
- [Envoy Proxy](https://www.envoyproxy.io/) — Sidecar 架构参考
- [ngrok](https://github.com/inconshreveable/ngrok) — 隧道模型参考
- [cloudflared](https://github.com/cloudflare/cloudflared) — WebSocket 隧道参考

---

*文档结束 | 如有调整请更新版本号并注明变更内容*
