package dpluger

import (
	"encoding/json"
	"os"
	"io/ioutil"
	"math"
	"strconv"
	"path/filepath"

	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
)

func SplitDirective(target string, suffix string, count int) (err error) {
	directiveFile, err := os.Open(target)
	if err != nil {
		return err
	}

	defer directiveFile.Close()

	byteValue, _ := ioutil.ReadAll(directiveFile);
	var directives siem.Directives
	json.Unmarshal(byteValue, &directives)

	length := len(directives.Dirs)
	files := int(math.Ceil(float64(length/count)))

	for i := 0; i < files; i++ {
		d := siem.Directives{}
		d.Dirs = directives.Dirs[i * count: i * count + count]

		b, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return err
		}

		err = fs.OverwriteFile(string(b), getFilename(target, suffix, i))
	}
	
	return
}

func getFilename(filename string, suffix string, n int) (filenameRet string) {
	ext := filepath.Ext(filename)
	name := filename[0:len(filename)-len(ext)]
	filenameRet = name + suffix + strconv.Itoa(n + 1) + ext
	return
}