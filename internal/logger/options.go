package logger

import (
	"sync"
)

var opts *Options
var once sync.Once

type Options struct {
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

// func NewOptions(opt *options.Logger) *Options {
// 	once.Do(func() {
// 		opts = &Options{
// 			FilePath:    opt.FilePath,    //opt.FilePath,
// 			Level:       int8(opt.Level), //int8(zapcore.InfoLevel),
// 			MaxSize:     opt.MaxSize,     //128,
// 			MaxBackups:  opt.MaxBackups,  //30,
// 			MaxAge:      opt.MaxAge,      //7,
// 			Compress:    opt.Compress,    //true,
// 			ServiceName: opt.ServiceName, //"Main",
// 			Stdout:      opt.Stdout,
// 			Filename:    fmt.Sprintf("%s%s", opt.FilePath, opt.Filename),
// 			LocalTime:   opt.LocalTime,
// 			Debug:       opt.Debug == true,
// 		}
// 	})
// 	return opts
// }
