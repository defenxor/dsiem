package fs

import (
	"errors"
	"os"
	"path"

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
		g := os.Getenv("GOPATH")
		if g == "" {
			return "", errors.New("cannot find $GOPATH env variable")
		}
		dir = path.Join(g, "src", "github.com", "defenxor", "dsiem")
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

// EnsureDir creates directory if it doesnt exist
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, os.FileMode(0700))
}
