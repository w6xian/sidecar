package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

/**
 * 获取日志
 * filePath 日志文件路径
 * level 日志级别
 * maxSize 每个日志文件保存的最大尺寸 单位：M
 * maxBackups 日志文件最多保存多少个备份
 * maxAge 文件最多保存多少天
 * compress 是否压缩
 * serviceName 服务名
 */

func newLoggerInstance(o *Options) *zap.Logger {

	// First, define our level-handling logic.
	//highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
	//	return lvl >= zapcore.ErrorLevel
	//})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	//topicDebugging := zapcore.AddSync(io.Discard)
	//topicErrors := zapcore.AddSync(io.Discard)

	// High-priority output should also go to standard error, and low-priority
	// output should also go to standard out.
	consoleDebugging := zapcore.Lock(os.Stdout)
	//consoleErrors := zapcore.Lock(os.Stderr)

	hook := &lumberjack.Logger{
		Filename:   o.Filename,
		MaxSize:    o.MaxSize,
		MaxBackups: o.MaxBackups,
		MaxAge:     o.MaxAge,
		Compress:   o.Compress, // disabled by default
	}

	fileErrors := zapcore.AddSync(hook)

	// Optimize the Kafka output for machine consumption and the console output
	// for human operators.
	//kafkaEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	ec := zap.NewDevelopmentEncoderConfig()
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	ec.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(ec)
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	cores := make([]zapcore.Core, 0, 2)
	cores = append(cores,
		//zapcore.NewCore(kafkaEncoder, topicErrors, highPriority),
		//zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		//zapcore.NewCore(kafkaEncoder, topicDebugging, lowPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority))

	if o.Stdout {
		cores = append(cores, zapcore.NewCore(fileEncoder, fileErrors, lowPriority))
	}

	core := zapcore.NewTee(cores...)

	options := make([]zap.Option, 0, 5)
	if o.Debug {
		// 开启开发模式，堆栈跟踪
		options = append(options, zap.AddCaller(), zap.Development())
	}
	// 设置初始化字段,如：添加一个服务器名称
	options = append(options, zap.Fields(zap.String("serviceName", o.ServiceName)))

	logger := zap.New(core, options...)
	defer logger.Sync()
	return logger
}
