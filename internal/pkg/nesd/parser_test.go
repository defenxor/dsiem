package nesd

import (
	"os"
	"path"
	"testing"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

var csvInitialized bool

func TestInitCSV(t *testing.T) {

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	log.Setup(true)
	err = InitCSV(`/\/\/\/`)
	if err == nil {
		t.Error("Expected error due to bad path")
	}
	csvDir := path.Join(dir, "fixtures")
	err = InitCSV(csvDir)
	if err == nil {
		t.Error("Expected error due to empty result")
	}
	csvDir = path.Join(dir, "fixtures", "example1")
	err = InitCSV(csvDir)
	if err == nil {
		t.Fatal("expected parsing error")
	}
	csvDir = path.Join(dir, "fixtures", "example2")
	err = InitCSV(csvDir)
	if err != nil {
		t.Fatal(err)
	}
	csvInitialized = true

}
