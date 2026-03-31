package options

type RiskData struct {
	Id     string            `json:"id"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
	Time   int64             `json:"time"`
}

type Mysql struct {
	Database     string `json:"database"`
	Protocol     string `json:"protocol"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Pretable     string `json:"pretable"`
	Charset      string `json:"charset"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Maxconnetion int    `json:"maxconnection"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idel_conns"`
	MaxLifetime  int    `json:"max_life_time"`
	DNS          string `json:"dns"`
}

type Tracer struct {
	Enable bool         `json:"enable"`
	Jaeger JaegerConfig `json:"jaeger"`
}

type JaegerConfig struct {
	URL string `json:"url"`
}

type Etcd struct {
	Servers []string `json:"servers"`
	Timeout int      `json:"timeout"`
}

type Nsq struct {
	Server  string `json:"server"`
	Topic   string `json:"topic"`
	Channel string `json:"channel"`
}

type Redis struct {
	Servers []string `json:"servers"`
	Timeout int      `json:"timeout"`
}

type Logger struct {
	FilePath    string `json:"file_path"`    // 日志文件路径
	Level       int8   `json:"level"`        // 日志级别
	MaxSize     int    `json:"max_size"`     // 每个日志文件保存的最大尺寸 单位：M
	MaxBackups  int    `json:"max_backups"`  // 日志文件最多保存多少个备份
	MaxAge      int    `json:"max_age"`      // 文件最多保存多少天
	Compress    bool   `json:"compress"`     // 是否压缩
	ServiceName string `json:"service_name"` // 服务名
	Stdout      bool   `json:"std_out"`
	Filename    string `json:"file_name"`
	LocalTime   bool   `json:"localtime"`
	Debug       bool   `json:"debug"`
}

type Pem struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}
