// short for "log"
package logger

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ilog *Logger
var lg sync.Once

type StdLog interface {
	Debug(s string, args ...zapcore.Field)
	Info(s string, args ...zapcore.Field)
	Warn(s string, args ...zapcore.Field)
	Error(s string, args ...zapcore.Field)
	Panic(s string, args ...zapcore.Field)
	Fatal(s string, args ...zapcore.Field)
}

type Logger struct {
	logger  *zap.Logger
	options *Options
}

// NewLogger 统一添加标示
func NewLogger(opt ...*Options) StdLog {
	lg.Do(func() {
		l := len(opt)
		if l <= 0 {
			panic("初始Logger出错")
		}
		ilog = &Logger{
			options: opt[0],
			logger:  newLoggerInstance(opt[0]).With(zap.String("type", "Logger")),
		}
	})
	return ilog
}

func (l Logger) Debug(s string, args ...zap.Field) {
	l.logger.Debug(s, args...)
}

func (l Logger) Info(s string, args ...zap.Field) {
	l.logger.Info(s, args...)
}
func (l Logger) Warn(s string, args ...zap.Field) {
	l.logger.Warn(s, args...)
}
func (l Logger) Error(s string, args ...zap.Field) {
	l.logger.Error(s, args...)
}
func (l Logger) Panic(s string, args ...zap.Field) {
	l.logger.Panic(s, args...)
}
func (l Logger) Fatal(s string, args ...zap.Field) {
	l.logger.Fatal(s, args...)
}

type LogLevel int

const (
	DEBUG     = LogLevel(7)
	INFO      = LogLevel(6)
	NOTICE    = LogLevel(5)
	WARN      = LogLevel(4)
	ERROR     = LogLevel(3)
	CRITICAL  = LogLevel(2)
	ALERT     = LogLevel(1)
	EMERGENCY = LogLevel(0)
)

func Logf(msgLevel LogLevel, s string, args ...interface{}) {
	s = fmt.Sprintf(s, args...)
	if ilog == nil {
		fmt.Print(s)
	}
	switch msgLevel {
	case DEBUG:
		ilog.Debug(s)
	case INFO:
		ilog.Info(s)
	case WARN:
		ilog.Warn(s)
	case ERROR:
		ilog.Error(s)
	case CRITICAL:
		ilog.Panic(s)
	case ALERT:
		ilog.Fatal(s)
	}
}

func L7Debug(s string, args ...any) {
	ilog.Debug(fmt.Sprintf(s, args...))
}
func L6Info(s string, args ...any) {
	ilog.Info(fmt.Sprintf(s, args...))
}
func L5Notice(s string, args ...any) {
	ilog.Info(fmt.Sprintf(s, args...))
}
func L4Warn(s string, args ...any) {
	ilog.Warn(fmt.Sprintf(s, args...))
}
func L3Error(s string, args ...any) {
	ilog.Error(fmt.Sprintf(s, args...))
}
func L2Critical(s string, args ...any) {
	ilog.Panic(fmt.Sprintf(s, args...))
}
func L1Alert(s string, args ...any) {
	ilog.Fatal(fmt.Sprintf(s, args...))
}
func L0Emergency(s string, args ...any) {
	Panic(fmt.Errorf(s, args...))
}
