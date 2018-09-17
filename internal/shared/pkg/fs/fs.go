package fs

import (
	"os"

	"github.com/kardianos/osext"
)

// FileExist check if path exist
func FileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetDir returns the program root directory
func GetDir(devEnv bool) (string, error) {
	dir, err := osext.ExecutableFolder()

	if devEnv == true {
		// check both docker internal location or dev local pc location
		dir = "/go/src/dsiem"
		if !FileExist(dir + "/conf/assets.json") {
			dir = "/home/mmta/go/src/dsiem"
		}
	}

	return dir, err
}

// AppendToFile write s to the end of filename
func AppendToFile(s string, filename string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s + "\n")
	return err
}

// OverwriteFile truncate filename and write s into it
func OverwriteFile(s string, filename string) error {
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s + "\n")
	return err
}
