package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/kardianos/osext"
)

func getDir() (string, error) {
	dir, err := osext.ExecutableFolder()

	if devEnv == true {
		dir = "/home/mmta/go/src/ossim-converter"
	}

	return dir, err
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func insertDirectivesXML(filename string) error {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, `<?xml version="1.0" encoding="UTF-8"?>`) {
			lines[i] = `<?xml version="1.0" encoding="UTF-8"?>` + "\n<directives>"
			break
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(filename, []byte(output), 0644)
	return err
}

func appendToFile(s string, filename string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s + "\n")
	return err
}

func writeToFile(s string, filename string) error {
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s + "\n")
	return err
}
