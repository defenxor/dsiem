package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zlog *zap.Logger

// Setup initialize logger
func Setup(enableDebugMessage bool) (err error) {
	if enableDebugMessage {
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.DisableStacktrace = true
		cfg.DisableCaller = true
		zlog, err = cfg.Build(zap.AddStacktrace(zap.ErrorLevel))
	} else {
		cfg := zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.DisableCaller = true
		cfg.OutputPaths = []string{"stdout"}
		cfg.ErrorOutputPaths = []string{"stderr"}
		zlog, err = cfg.Build()
	}
	if err == nil {
		zlog.Sync()
	}
	return
}

// M defines the type for log messages
type M struct {
	Msg string // the message
	DId int    // directive ID
	BId string // backlog ID
	CId uint64 // conn ID
}

// Info log with info level
func Info(m M) {
	go zlog.Info(m.Msg, parseFields(&m)...)
}

// Warn log with info level
func Warn(m M) {
	go zlog.Warn(m.Msg, parseFields(&m)...)
}

// Debug log with info level
func Debug(m M) {
	go zlog.Debug(m.Msg, parseFields(&m)...)
}

// Error log with error level
func Error(m M) {
	go zlog.Error(m.Msg, parseFields(&m)...)
}

func parseFields(m *M) (f []zapcore.Field) {
	f = []zapcore.Field{}
	if m.DId != 0 {
		f = append(f, zap.Int("directive", m.DId))
	}
	if m.CId != 0 {
		f = append(f, zap.Uint64("connId", m.CId))
	}
	if m.BId != "" {
		f = append(f, zap.String("backlog", m.BId))
	}
	return
}
