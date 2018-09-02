package main

import (
	"os"

	"github.com/kardianos/osext"
)

func getDir() (string, error) {
	dir, err := osext.ExecutableFolder()
	return dir, err
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
