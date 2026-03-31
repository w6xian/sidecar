package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ConnectedSidecars 当前已连接的 Sidecar 数量
	ConnectedSidecars = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sidecar_connected_total",
		Help: "Number of currently connected sidecar clients",
	})

	// RegisteredServices 当前已注册的服务数量
	RegisteredServices = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sidecar_registered_services_total",
		Help: "Number of currently registered services",
	})

	// ProxyRequestsTotal 代理请求总数
	ProxyRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sidecar_proxy_requests_total",
		Help: "Total number of proxy requests",
	}, []string{"service_id", "protocol", "method", "status"})

	// ProxyRequestDuration 代理请求耗时
	ProxyRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sidecar_proxy_request_duration_seconds",
		Help:    "Proxy request duration in seconds",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	}, []string{"service_id", "protocol"})

	// WebSocketErrors WebSocket 错误计数
	WebSocketErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sidecar_websocket_errors_total",
		Help: "Total number of WebSocket errors",
	}, []string{"type"})

	// HeartbeatTotal 心跳计数
	HeartbeatTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sidecar_heartbeat_total",
		Help: "Total number of heartbeats",
	}, []string{"direction"})
)
