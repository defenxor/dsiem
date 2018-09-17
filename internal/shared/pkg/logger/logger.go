package logger

import (
	"os"

	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var oLog *logrus.Logger

// Setup initialize logger
func Setup(enableDebugMessage bool) {
	oLog = logrus.New()
	if enableDebugMessage {
		oLog.SetLevel(logrus.DebugLevel)
	} else {
		oLog.SetLevel(logrus.InfoLevel)
	}

	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	oLog.Formatter = formatter
	oLog.Out = os.Stdout

	// only uses append, no need for lock
	oLog.SetNoLock()

	// log.SetOutput(oLog.Writer())

}

func logAndExit(err error) {
	// time.Sleep(5 * time.Minute)
	oLog.Fatalf("%+v", errors.Wrap(err, ""))
}

// Info log with info level
func Info(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	oLog.Info("[" + sID + "] " + msg)
}

// Warn log with warn level
func Warn(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	oLog.Warn("[" + sID + "] " + msg)
}

// Debug log with debug level
func Debug(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	oLog.Debug("[" + sID + "] " + msg)
}
