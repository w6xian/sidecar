package server

import (
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
	"github.com/w6xian/sidecar/pkg/metrics"
)

// RouteEntry 路由条目：一个服务可能有多个 Sidecar 注册
type RouteEntry struct {
	ServiceInfo protocol.ServiceInfo
	ConnIDs     []string // 注册该服务的 Sidecar 连接 ID 列表
	counter     uint64   // 轮询计数器
}

// Router 服务路由表
type Router struct {
	mu     sync.RWMutex
	routes map[string]*RouteEntry // serviceId → RouteEntry

	// pathIndex 路径前缀 → serviceId 映射
	pathIndex map[string]string
}

// NewRouter 创建路由表
func NewRouter() *Router {
	return &Router{
		routes:    make(map[string]*RouteEntry),
		pathIndex: make(map[string]string),
	}
}

// Register 注册服务到路由表
func (r *Router) Register(connID string, services []protocol.ServiceInfo) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var accepted []string

	for _, svc := range services {
		entry, exists := r.routes[svc.ID]
		if !exists {
			entry = &RouteEntry{
				ServiceInfo: svc,
				ConnIDs:     []string{},
			}
			r.routes[svc.ID] = entry
			r.pathIndex[svc.ExposePath] = svc.ID
		}

		// 检查该连接是否已注册过该服务
		found := false
		for _, cid := range entry.ConnIDs {
			if cid == connID {
				found = true
				break
			}
		}
		if !found {
			entry.ConnIDs = append(entry.ConnIDs, connID)
		}

		accepted = append(accepted, svc.ID)
		logger.L().Info("service registered",
			zap.String("service_id", svc.ID),
			zap.String("conn_id", connID),
			zap.String("expose_path", svc.ExposePath),
		)
	}

	metrics.RegisteredServices.Set(float64(len(r.routes)))
	return accepted
}

// Unregister 注销指定服务
func (r *Router) Unregister(connID string, serviceIDs []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, sid := range serviceIDs {
		entry, exists := r.routes[sid]
		if !exists {
			continue
		}
		entry.ConnIDs = removeString(entry.ConnIDs, connID)
		if len(entry.ConnIDs) == 0 {
			delete(r.pathIndex, entry.ServiceInfo.ExposePath)
			delete(r.routes, sid)
		}
	}

	metrics.RegisteredServices.Set(float64(len(r.routes)))
}

// RemoveByConn 移除指定连接的所有服务注册
func (r *Router) RemoveByConn(connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for sid, entry := range r.routes {
		entry.ConnIDs = removeString(entry.ConnIDs, connID)
		if len(entry.ConnIDs) == 0 {
			delete(r.pathIndex, entry.ServiceInfo.ExposePath)
			delete(r.routes, sid)
		}
	}

	metrics.RegisteredServices.Set(float64(len(r.routes)))
}

// MatchByPath 根据请求路径匹配服务（最长前缀匹配）
func (r *Router) MatchByPath(path string) (*RouteEntry, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bestMatch string
	for prefix, sid := range r.pathIndex {
		if len(prefix) > 0 && hasPathPrefix(path, prefix) {
			if len(prefix) > len(bestMatch) {
				bestMatch = prefix
				_ = sid
			}
		}
	}

	if bestMatch == "" {
		return nil, "", false
	}

	sid := r.pathIndex[bestMatch]
	entry := r.routes[sid]
	return entry, bestMatch, true
}

// MatchByServiceID 根据 serviceId 匹配服务
func (r *Router) MatchByServiceID(serviceID string) (*RouteEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.routes[serviceID]
	return entry, exists
}

// PickConn 轮询选择一个连接 ID（负载均衡）
func (r *Router) PickConn(entry *RouteEntry) string {
	if len(entry.ConnIDs) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&entry.counter, 1)
	return entry.ConnIDs[idx%uint64(len(entry.ConnIDs))]
}

// GetAllRoutes 获取所有路由信息（用于调试/管理）
func (r *Router) GetAllRoutes() map[string]*RouteEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*RouteEntry, len(r.routes))
	for k, v := range r.routes {
		result[k] = v
	}
	return result
}

// hasPathPrefix 检查路径是否以指定前缀开头
func hasPathPrefix(path, prefix string) bool {
	if len(path) < len(prefix) {
		return false
	}
	if path[:len(prefix)] != prefix {
		return false
	}
	// 确保前缀匹配在路径边界
	if len(path) > len(prefix) && path[len(prefix)] != '/' {
		return false
	}
	return true
}

// removeString 从切片中移除指定字符串
func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
