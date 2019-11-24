package siem

import (
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

type backlogLogger struct {
	ID string
}

func newBacklogLogger(backlogID string) backlogLogger {
	return backlogLogger{
		ID: backlogID,
	}
}

func (b *backlogLogger) info(msg string, connID uint64) {
	log.Info(log.M{Msg: msg, BId: b.ID, CId: connID})
}

func (b *backlogLogger) debug(msg string, connID uint64) {
	log.Debug(log.M{Msg: msg, BId: b.ID, CId: connID})
}

func (b *backlogLogger) warn(msg string, connID uint64) {
	log.Warn(log.M{Msg: msg, BId: b.ID, CId: connID})
}
