# Sidecar 项目记忆

## 项目概况
- **名称**: Sidecar Proxy - 内网穿透服务
- **语言**: Go 1.22+
- **架构**: WebSocket 长连接 + 服务端网关 + 内网 Sidecar 客户端
- **协议**: JSON 格式的 WebSocket 消息
- **技术栈**: gorilla/websocket, chi v5, gRPC, viper, zap, Prometheus

## 目录结构
- `cmd/` - 入口程序（server, sidecar）
- `internal/server/` - 服务端核心（hub, router, handler, gateway）
- `internal/sidecar/` - 客户端核心（client, register, forwarder）
- `internal/protocol/` - WebSocket 消息协议定义
- `config/` - 配置文件
- `deploy/` - Docker 部署配置

## 2026-03-31 功能扩展：连接别名路由

### 新增功能
实现了基于连接别名的路由功能，格式为 `/{alias}/*`，允许直接访问内网服务的所有路径。

### 协议变更
1. `protocol.RegisterRequest` 新增 `ConnAlias string` 字段
2. `protocol.RegisterACK` 新增 `ConnAlias string` 字段

### 服务端改动
1. `server/Hub`:
   - `SidecarConn` 新增 `Alias string` 字段
   - `Hub` 新增 `aliases sync.Map` 存储 alias→connID 映射
   - 新增 `GetConnByAlias()` 和 `AliasExists()` 方法

2. `server/Handler`:
   - 注册时处理 `ConnAlias`，冲突时自动使用 UUID
   - ACK 中回传实际分配的别名

3. `server/Gateway`:
   - 新增 `proxyByAliasHandler()` 处理 `/{alias}/*` 路由
   - 路由优先级：别名路由 > 服务路径路由

### 客户端改动
1. `sidecar/SidecarConfig` 新增 `ConnAlias string` 字段
2. `sidecar/RegisterService` 发送注册时带上 `ConnAlias`
3. `sidecar/HandleRegisterACK` 打印分配的别名

### 配置文件
`config/sidecar.yaml` 新增可选字段：
```yaml
sidecar:
  conn_alias: "my-machine"  # 可选，不指定则自动生成 UUID
```

### 路由优先级
1. `/sidecar/connect` - WebSocket 连接
2. `/health`, `/metrics`, `/admin/routes` - 管理端点
3. `/{alias}/*` - 连接别名路由（新增）
4. `/*` - 服务路径通配路由（原有）

### 使用示例
```bash
# 设置别名后，直接访问内网服务任意路径
curl http://server:8443/my-machine/any/path/you/want

# 原有的服务路径路由仍然有效
curl http://server:8443/api/user/profile
```

### Bug 修复
1. **Prometheus label 不匹配**：修复 `ProxyRequestDuration` Histogram 的 label 数量问题，移除多余的 `method` label
2. **ServiceID 查找失败**：别名路由正确使用连接注册的第一个 HTTP 服务的 `ServiceID`，而非直接使用 `connID`

### 测试状态
✅ 编译通过，功能实现完成，已修复所有已知 bug

## 2026-04-01 功能扩展：三类控制指令

### 新增指令
1. **更新指令**（`UPDATE` / `UPDATE_ACK`）：服务端推送更新包地址 → 客户端下载、SHA256 校验、替换二进制、重启
2. **Lua 脚本执行**（`EXEC_LUA` / `EXEC_LUA_RESULT`）：服务端下发脚本 → 客户端用 `github.com/w6xian/gua` 执行 → 返回输出
3. **文件上传**（`UPLOAD_FILE` / `UPLOAD_FILE_DATA`）：服务端请求日志文件 → 客户端读取（支持 tail 行数）→ base64 回传

### 管理 API 端点（POST）
- `/admin/cmd/update/{connID}` — 推送更新
- `/admin/cmd/exec-lua/{connID}` — 执行 Lua
- `/admin/cmd/upload-file/{connID}` — 上传文件

### 新增文件
- `internal/sidecar/cmd_update.go`
- `internal/sidecar/cmd_lua.go`
- `internal/sidecar/cmd_upload.go`

### 编译状态
✅ `go build ./...` 通过
