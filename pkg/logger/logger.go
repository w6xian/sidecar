package logger

import (
	"os"
	"sync"

	"github.com/w6xian/sidecar/internal/config"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
	atomicLevel  zap.AtomicLevel
)

// Init 初始化全局日志
func Init(profile *config.Logger, hook *lumberjack.Logger) {
	once.Do(func() {

		atomicLevel = zap.NewAtomicLevel()
		encoderConfig := ecszap.EncoderConfig{
			EncodeName:     zapcore.FullNameEncoder,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   ecszap.FullCallerEncoder,
		}
		syncer := []zapcore.WriteSyncer{
			zapcore.AddSync(hook),
		}
		if profile.Stdout {
			syncer = append(syncer, zapcore.AddSync(os.Stdout))
		}
		core := ecszap.NewCore(encoderConfig,
			zapcore.NewMultiWriteSyncer(syncer...),
			zap.InfoLevel)
		globalLogger = zap.New(core, zap.AddCaller())
	})
}

// SetLevel 动态调整日志级别
func SetLevel(level string) {
	var l zapcore.Level
	switch level {
	case "debug":
		l = zapcore.DebugLevel
	case "info":
		l = zapcore.InfoLevel
	case "warn":
		l = zapcore.WarnLevel
	case "error":
		l = zapcore.ErrorLevel
	default:
		l = zapcore.InfoLevel
	}
	atomicLevel.SetLevel(l)
}

// L 获取全局 Logger
func L() *zap.Logger {
	return globalLogger
}

// S 获取全局 SugaredLogger
func S() *zap.SugaredLogger {
	return L().Sugar()
}

// Sync 刷新日志缓冲
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}
