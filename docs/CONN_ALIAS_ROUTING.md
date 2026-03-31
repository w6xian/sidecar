# 连接别名路由功能

## 功能说明

Sidecar 连接支持通过"连接别名"进行路由，格式为 `/{alias}/*`。这样可以让每个 Sidecar 连接直接暴露其内网服务的所有路径，而无需在服务配置中指定具体的 `expose_path`。

## 使用方式

### 客户端配置

在 `config/sidecar.yaml` 中设置 `conn_alias` 字段：

```yaml
# Sidecar 客户端配置
sidecar:
  server_addr: "ws://localhost:8443/sidecar/connect"
  token: "your-auth-token"
  reconnect_interval: 5s
  max_reconnect_interval: 60s
  conn_alias: "my-machine"  # 设置连接别名（可选）
```

### 访问方式

设置 `conn_alias: "my-machine"` 后，可以通过以下方式访问内网服务：

```bash
# 所有请求都会转发到该连接，路径直接透传
curl http://localhost:8443/my-machine/api/v1/users
curl http://localhost:8443/my-machine/any/path/you/want
```

### 注意事项

1. **别名冲突**：如果多个连接使用相同的 `conn_alias`，后续连接会被拒绝使用该别名，服务端会自动使用 UUID 作为别名
2. **不指定别名**：如果不设置 `conn_alias` 或留空，服务端会自动生成 UUID 作为别名
3. **服务路径路由不受影响**：原有的 `/api/user` 等基于 `expose_path` 的路由方式仍然有效，两者可以共存
4. **重连后别名保持**：客户端指定别名后，断线重连会使用相同的别名

## 技术实现

### 协议变更

1. **RegisterRequest** 增加 `ConnAlias` 字段（客户端请求中携带）
2. **RegisterACK** 增加 `ConnAlias` 字段（服务端确认实际使用的别名）

### 服务端处理

1. **Hub** 维护 `alias → connID` 映射
2. **Gateway** 新增 `/{alias}/*` 路由，优先级高于通配路由 `/*`
3. **Handler** 注册时检查别名冲突，冲突时自动使用 UUID

### 路由优先级

1. `/sidecar/connect` - WebSocket 连接端点
2. `/health` - 健康检查
3. `/metrics` - Prometheus 指标
4. `/admin/routes` - 路由信息查看
5. `/{alias}/*` - 连接别名路由（新增）
6. `/*` - 服务路径通配路由（原有）

## 示例

### 示例 1：指定别名访问

**客户端配置：**
```yaml
sidecar:
  conn_alias: "prod-db"
```

**访问方式：**
```bash
# 直接访问 prod-db 上的任何服务
curl http://server:8443/prod-db/metrics
curl http://server:8443/prod-db/admin/status
```

### 示例 2：多服务共存

**Sidecar 1 配置：**
```yaml
sidecar:
  conn_alias: "user-service"
services:
  - id: "user-api"
    name: "用户服务"
    protocol: "http"
    local_addr: "http://127.0.0.1:8080"
    expose_path: "/api/user"
```

**访问方式（两种方式都支持）：**
```bash
# 方式1：通过别名（访问 8080 端口的任何路径）
curl http://server:8443/user-service/api/v1/health

# 方式2：通过服务路径（只能访问配置的 expose_path）
curl http://server:8443/api/user/profile
```

### 示例 3：不指定别名

**客户端配置：**
```yaml
sidecar:
  conn_alias: ""  # 或不设置该字段
```

服务端会自动生成 UUID 作为别名，例如：
```bash
# 服务端日志显示分配的别名
INFO: services registered successfully assigned_alias=550e8400-e29b-41d4-a716-446655440000

# 使用分配的 UUID 访问
curl http://server:8443/550e8400-e29b-41d4-a716-446655440000/any/path
```

## API 查看

可以通过 `/admin/routes` 查看当前注册的所有服务：

```bash
curl http://localhost:8443/admin/routes
```

响应示例：
```json
[
  {
    "service_id": "user-service",
    "name": "用户服务",
    "protocol": "http",
    "expose_path": "/api/user",
    "sidecar_count": 1
  }
]
```

注意：`/admin/routes` 只显示基于 `expose_path` 的服务注册信息，不显示连接别名。连接别名由 Hub 直接管理，用于 `/{alias}/*` 路由。
