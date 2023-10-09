package dpluger

import (
	"encoding/json"
	"io"
	"os"
	"math"
	"strconv"
	"path/filepath"
	"errors"

	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
)

// SplitDirective Split single directive json file into multiple json files
func SplitDirective(target string, suffix string, count int, delete bool) (err error) {
	directiveFile, err := os.Open(target)
	if err != nil {
		return err
	}

	defer directiveFile.Close()

	byteValue, _ := io.ReadAll(directiveFile);
	var directives siem.Directives
	json.Unmarshal(byteValue, &directives)

	length := float64(len(directives.Dirs))
	files := int(math.Ceil(float64(length/float64(count))))

	if files < 2 {
		err = errors.New("Cannot split into single file, the target directive only contains " + strconv.Itoa(int(length)) + " directive, but the splitted directive count is set to " + strconv.Itoa(count) + ", \nuse -n flags to define another splitted item count, or use --help to show available flags")
		return
	}

	for i := 0; i < files; i++ {
		d := siem.Directives{}
		start := i * count
		end := start + count
		if end > int(length) {
			end = int(length)
		}
		
		d.Dirs = directives.Dirs[start: end]

		b, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return err
		}

		err = fs.OverwriteFile(string(b), getFilename(target, suffix, i))
	}

	if delete {
		err = os.Remove(target)
	}
	
	return
}

func getFilename(filename string, suffix string, n int) (filenameRet string) {
	ext := filepath.Ext(filename)
	name := filename[0:len(filename)-len(ext)]
	filenameRet = name + suffix + strconv.Itoa(n + 1) + ext
	return
}
