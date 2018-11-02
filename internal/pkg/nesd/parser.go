package nesd

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/gocarina/gocsv/v2"
)

const (
	nessusGlob = "nessus_*.csv"
)

type nessusScans struct {
	entries []nScan
}

// Plugin ID,CVE,CVSS,Risk,Host,Protocol,Port,Name,Synopsis,Description,Solution,See Also,Plugin Output
type nScan struct {
	PluginID     int64   `csv:"Plugin ID"`
	CVE          string  `csv:"CVE"`
	CVSS         float32 `csv:"CVSS"`
	Risk         string  `csv:"Risk"`
	Host         string  `csv:"Host"`
	Protocol     string  `csv:"Protocol"`
	Port         int     `csv:"Port"`
	Name         string  `csv:"Name"`
	Synopsis     string  `csv:"Synopsis"`
	Description  string  `csv:"Description"`
	Solution     string  `csv:"Solution"`
	SeeAlso      string  `csv:"See Also"`
	PluginOutput string  `csv:"Plugin Output"`
}

var vulns nessusScans

// InitCSV read nessus scan results from CSV
func InitCSV(dir string) error {
	csvDir = dir
	p := path.Join(csvDir, nessusGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}

	for i := range files {
		var n nessusScans
		if !fs.FileExist(files[i]) {
			return errors.New("Cannot find " + files[i])
		}
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := ioutil.ReadAll(file)
		err = gocsv.UnmarshalBytes(byteValue, &n.entries)
		if err != nil {
			return err
		}
		for j := range n.entries {
			vulns.entries = append(vulns.entries, n.entries[j])
		}
	}

	total := len(vulns.entries)
	if total == 0 {
		return errors.New("cannot find valid nessus scan results to load from " + dir)
	}
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(total) + " scan entries."})

	return nil
}
