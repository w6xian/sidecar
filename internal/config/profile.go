package config

import (
	"sync"
	"time"

	"github.com/w6xian/sidecar/internal/utils/id"
	"github.com/w6xian/sqlm"
)

var opt *Profile
var once sync.Once

/**
   http_addr: ":8443"     # HTTP 入口网关监听地址
  grpc_addr: ":9443"     # gRPC 入口网关监听地址
  proxy_timeout: 30s     # 代理请求超时
*/

type ServerConfig struct {
	HTTPAddr     string        `mapstructure:"http_addr"`
	GRPCAddr     string        `mapstructure:"grpc_addr"`
	ProxyTimeout time.Duration `mapstructure:"proxy_timeout"`
}

/*
server_addr: "ws://localhost:8443/sidecar/connect"  # 服务端 WebSocket 地址（生产使用 wss://）
token: "your-auth-token"                            # 鉴权 Token
reconnect_interval: 5s                              # 重连间隔基础值
max_reconnect_interval: 60s                         # 最大重连间隔
conn_alias: "abc"
*/
type SidecarConfig struct {
	ServerAddr           string        `mapstructure:"server_addr"`
	AppSn                string        `mapstructure:"app_sn"`
	AppId                string        `mapstructure:"app_id"`
	AppSec               string        `mapstructure:"app_sec"`
	ReconnectInterval    time.Duration `mapstructure:"reconnect_interval"`
	MaxReconnectInterval time.Duration `mapstructure:"max_reconnect_interval"`
	ConnAlias            string        `mapstructure:"conn_alias"`
}

/*
auth:

	tokens:
	  - "your-auth-token"  # 允许的 Sidecar 连接 Token
	ip_whitelist: []       # IP 白名单，为空则不限制
*/
type AuthConfig struct {
	Tokens      []string `mapstructure:"tokens"`
	IPWhitelist []string `mapstructure:"ip_whitelist"`
}

/*
id: "order-grpc"
name: "订单 gRPC 服务"
protocol: "grpc"
local_addr: "127.0.0.1:9090"
expose_path: "/grpc/order"
*/
type ServiceConfig struct {
	Id         string `mapstructure:"id"`
	Name       string `mapstructure:"name"`
	Protocol   string `mapstructure:"protocol"`
	LocalAddr  string `mapstructure:"local_addr"`
	ExposePath string `mapstructure:"expose_path"`
}

type Logger struct {
	FilePath    string `mapstructure:"file_path"`    // 日志文件路径
	Level       int8   `mapstructure:"level"`        // 日志级别
	MaxSize     int    `mapstructure:"max_size"`     // 每个日志文件保存的最大尺寸 单位：M
	MaxBackups  int    `mapstructure:"max_backups"`  // 日志文件最多保存多少个备份
	MaxAge      int    `mapstructure:"max_age"`      // 文件最多保存多少天
	Compress    bool   `mapstructure:"compress"`     // 是否压缩
	ServiceName string `mapstructure:"service_name"` // 服务名
	Stdout      bool   `mapstructure:"std_out"`
	Filename    string `mapstructure:"file_name"`
	LocalTime   bool   `mapstructure:"localtime"`
	Debug       bool   `mapstructure:"debug"`
}

type Profile struct {
	Id       string
	OpenId   string
	AppId    string
	DeviceId string

	Language string          `mapstructure:"language"`
	Timezone string          `mapstructure:"timezone"`
	LogLevel string          `mapstructure:"log_level"`
	Store    *sqlm.Server    `mapstructure:"store"`
	Sidecar  *SidecarConfig  `mapstructure:"sidecar"`
	Services []ServiceConfig `mapstructure:"services"`
	Logger   *Logger         `mapstructure:"logger"`
	Mode     string          `mapstructure:"mode"`
	Server   *ServerConfig   `mapstructure:"server"`
	Auth     *AuthConfig     `mapstructure:"auth"`
}

func NewProfile() *Profile {
	once.Do(func() {
		appId := "SC_SiDur7xmUAXY6pUtCYvsZz"
		// 创建连接
		opt = &Profile{
			Id: id.ShortID(),
			// 每台机器都有一个唯一的DeviceId
			DeviceId: id.DeviceId(appId),
			// sst cashier app id
			AppId:    appId,
			Language: "zh-CN",
			Timezone: "Asia/Shanghai",
			LogLevel: "info",
			Mode:     "prod",
			Store: &sqlm.Server{
				Database:     "cloud",
				Host:         "127.0.0.1",
				Port:         3306,
				Protocol:     "mysql",
				Pretable:     "mi_",
				Charset:      "utf8mb4",
				MaxOpenConns: 64,
				MaxIdleConns: 64,
				Username:     "root",
				Password:     "1Qazxsw2",
				MaxLifetime:  int(time.Second) * 60,
				DSN:          "sqlm_demo.db",
			},
			Logger: &Logger{
				ServiceName: "",   // 服务名
				FilePath:    "./", // 日志文件路径
				Filename:    "cash.log",
				Level:       -1,
				MaxSize:     500,   // 每个日志文件保存的最大尺寸 单位：M
				MaxBackups:  2024,  // 日志文件最多保存多少个备份
				MaxAge:      180,   // 文件最多保存多少天
				Compress:    false, // 是否压缩
				Stdout:      true,
				LocalTime:   true,
				Debug:       false,
			},
		}
	})
	return opt
}
