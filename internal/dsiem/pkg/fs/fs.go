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
		dir = "/go/src/siem"
		if !FileExist(dir + "/conf/assets.json") {
			dir = "/home/mmta/go/src/siem2/src-local"
		}
	}

	return dir, err
}
