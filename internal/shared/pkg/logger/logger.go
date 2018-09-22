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
		cfg.DisableCaller = true
		zlog, err = cfg.Build()
	} else {
		cfg := zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.DisableCaller = true
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
	zlog.Info(m.Msg, parseFields(&m)...)
}

// Warn log with info level
func Warn(m M) {
	zlog.Warn(m.Msg, parseFields(&m)...)
}

// Debug log with info level
func Debug(m M) {
	zlog.Debug(m.Msg, parseFields(&m)...)
}

func parseFields(m *M) (f []zapcore.Field) {
	f = []zapcore.Field{}
	switch {
	case m.DId != 0:
		f = append(f, zap.Int("directive", m.DId))
	case m.CId != 0:
		f = append(f, zap.Uint64("connId", m.CId))
	case m.BId != "":
		f = append(f, zap.String("backlog", m.BId))
	}
	return
}
