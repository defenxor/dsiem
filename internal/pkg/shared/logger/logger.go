// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package logger

import (
	"bufio"
	"bytes"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zlog *zap.Logger
var enableDebugMessage bool

var enc zapcore.Encoder
var wrt *bufio.Writer
var buffer bytes.Buffer
var zLock = sync.RWMutex{}

// TestMode is a flag for testing mode
var TestMode bool

// Setup initialize logger
func Setup(dbg bool) (err error) {
	zLock.Lock()
	defer zLock.Unlock()
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

//Info logs with info level
func Info(m M) {
	if TestMode {
		zLock.Lock()
		defer zLock.Unlock()
	}

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
		zlog.Info(m.Msg, zap.Uint64("connId", m.CId), zap.String("backlog", m.BId))
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
	if m.DId != 0 && m.CId != 0 && m.BId == "" {
		zlog.Info(m.Msg, zap.Int("directive", m.DId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		zlog.Info(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
}

func InfoMsg(msg string) {
	Info(M{Msg: msg})
}

//Warn logs with warn level
func Warn(m M) {
	if TestMode {
		zLock.Lock()
		defer zLock.Unlock()
	}

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
		zlog.Warn(m.Msg, zap.Uint64("connId", m.CId), zap.String("backlog", m.BId))
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
	if m.DId != 0 && m.CId != 0 && m.BId == "" {
		zlog.Warn(m.Msg, zap.Int("directive", m.DId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		zlog.Warn(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
}

//Debug logs with debug level
func Debug(m M) {
	if !enableDebugMessage {
		return
	}
	if TestMode {
		zLock.Lock()
		defer zLock.Unlock()
	}

	if m.DId == 0 && m.CId == 0 && m.BId == "" {
		zlog.Debug(m.Msg)
		return
	}
	if m.DId == 0 && m.CId == 0 && m.BId != "" {
		zlog.Debug(m.Msg, zap.String("backlog", m.BId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId == "" {
		zlog.Debug(m.Msg, zap.Uint64("connId", m.CId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId != "" {
		zlog.Debug(m.Msg, zap.Uint64("connId", m.CId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId == "" {
		zlog.Debug(m.Msg, zap.Int("directive", m.DId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId != "" {
		zlog.Debug(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId == "" {
		zlog.Debug(m.Msg, zap.Int("directive", m.DId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		zlog.Debug(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
}

//Error logs with error level
func Error(m M) {
	if TestMode {
		zLock.Lock()
		defer zLock.Unlock()
	}
	if m.DId == 0 && m.CId == 0 && m.BId == "" {
		zlog.Error(m.Msg)
		return
	}
	if m.DId == 0 && m.CId == 0 && m.BId != "" {
		zlog.Error(m.Msg, zap.String("backlog", m.BId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId == "" {
		zlog.Error(m.Msg, zap.Uint64("connId", m.CId))
		return
	}
	if m.DId == 0 && m.CId != 0 && m.BId != "" {
		zlog.Error(m.Msg, zap.Uint64("connId", m.CId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId == "" {
		zlog.Error(m.Msg, zap.Int("directive", m.DId))
		return
	}
	if m.DId != 0 && m.CId == 0 && m.BId != "" {
		zlog.Error(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId == "" {
		zlog.Error(m.Msg, zap.Int("directive", m.DId), zap.Uint64("connId", m.CId))
		return
	}
	if m.DId != 0 && m.CId != 0 && m.BId != "" {
		zlog.Error(m.Msg, zap.Int("directive", m.DId), zap.String("backlog", m.BId), zap.Uint64("connId", m.CId))
		return
	}
}

// CaptureZapOutput returns output of zap logger so that it can be used
// in tests
func CaptureZapOutput(funcToRun func()) string {
	zLock.Lock()
	buffer.Reset()
	zLock.Unlock()
	funcToRun()
	zLock.Lock()
	wrt.Flush()
	zLock.Unlock()
	return buffer.String()
}

// EnableTestingMode set zap for testing, should be called before CaptureZapOutput
func EnableTestingMode() {
	TestMode = true
	enc = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	wrt = bufio.NewWriter(&buffer)
	zlog = zap.New(
		zapcore.NewCore(enc, zapcore.AddSync(wrt), zapcore.DebugLevel))
}

/* nice to look at but more expensive?
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
