package main

import (
	"log"
	"os"

	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func setupLogger() {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	// formatter := &logrus.JSONFormatter{}
	logger.Formatter = formatter
	logger.Out = os.Stdout

	// use logrus for standard log output, those chatty 3rd-party libs ..
	log.SetOutput(logger.Writer())
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func logAndExit(err error) {
	// time.Sleep(5 * time.Minute)
	logger.Fatalf("%+v", errors.Wrap(err, ""))
}

func getDir() (string, error) {
	dir, err := osext.ExecutableFolder()
	return dir, err
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}