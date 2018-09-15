package main

import (
	"log"
	"os"

	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var oLog logger

func setupLogger(level logrus.Level) {
	oLog.logger = logrus.New()
	oLog.logger.SetLevel(level)

	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	oLog.logger.Formatter = formatter
	oLog.logger.Out = os.Stdout

	// only uses append, no need for lock
	oLog.logger.SetNoLock()

	log.SetOutput(oLog.logger.Writer())

}

func logAndExit(err error) {
	// time.Sleep(5 * time.Minute)
	oLog.logger.Fatalf("%+v", errors.Wrap(err, ""))
}

func logInfo(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	oLog.logger.Info("[" + sID + "] " + msg)
}

func logWarn(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	oLog.logger.Warn("[" + sID + "] " + msg)
}

func logDebug(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	oLog.logger.Debug("[" + sID + "] " + msg)
}
