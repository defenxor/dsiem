package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zlog *zap.Logger
var enableDebugMessage bool

// Setup initialize logger
func Setup(dbg bool) (err error) {
	enableDebugMessage = dbg
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

//Info log with info level
func Info(m M) {
	if m.DId == 0 && m.CId == 0 && m.BId == "" {
		zlog.Info(m.Msg)
		return
	}
	if m.DId == 0 && m.CId == 0 && m.BId != "" {
		zlog.Info(m.Msg, zap.String("backlog", m.BId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId == "" {
		zlog.Info(m.Msg, zap.Uint64("connId", m.CId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId != "" {
		zlog.Info(m.Msg, zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId == "" {
		zlog.Info(m.Msg, zap.Int("directive", m.DId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId != "" {
		zlog.Info(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		zlog.Info(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	return
}

//Warn log with warn level
func Warn(m M) {
	if m.DId == 0 && m.CId == 0 && m.BId == "" {
		zlog.Warn(m.Msg)
		return
	}
	if m.DId == 0 && m.CId == 0 && m.BId != "" {
		zlog.Warn(m.Msg, zap.String("backlog", m.BId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId == "" {
		zlog.Warn(m.Msg, zap.Uint64("connId", m.CId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId != "" {
		zlog.Warn(m.Msg, zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId == "" {
		zlog.Warn(m.Msg, zap.Int("directive", m.DId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId != "" {
		zlog.Warn(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		zlog.Warn(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	return
}

//Debug log with warn level
func Debug(m M) {
	if !enableDebugMessage {
		return
	}
	if m.DId == 0 && m.CId == 0 && m.BId == "" {
		go zlog.Debug(m.Msg)
		return
	}
	if m.DId == 0 && m.CId == 0 && m.BId != "" {
		go zlog.Debug(m.Msg, zap.String("backlog", m.BId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId == "" {
		go zlog.Debug(m.Msg, zap.Uint64("connId", m.CId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId != "" {
		go zlog.Debug(m.Msg, zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId == "" {
		go zlog.Debug(m.Msg, zap.Int("directive", m.DId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId != "" {
		go zlog.Debug(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		go zlog.Debug(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	return
}

//Error log with warn level
func Error(m M) {
	if m.DId == 0 && m.CId == 0 && m.BId == "" {
		go zlog.Error(m.Msg)
		return
	}
	if m.DId == 0 && m.CId == 0 && m.BId != "" {
		go zlog.Error(m.Msg, zap.String("backlog", m.BId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId == "" {
		go zlog.Error(m.Msg, zap.Uint64("connId", m.CId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId != "" {
		go zlog.Error(m.Msg, zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId == "" {
		go zlog.Error(m.Msg, zap.Int("directive", m.DId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId != "" {
		go zlog.Error(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		go zlog.Error(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
	return
}

/* nice to look at but too expensive
// Info log with info level
func Info(m M) {
	go zlog.Info(m.Msg, parseFields(&m)...)
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
*/
