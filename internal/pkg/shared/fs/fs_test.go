package fs

import (
	"os"
	"path"
	"testing"
)

func TestFS(t *testing.T) {
	_, err := GetDir(true)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Setenv("GOPATH", ""); err != nil {
		t.Fatal(err)
	}
	_, err = GetDir(true)
	if err == nil {
		t.Fatal("expected error due to missing GOPATH")
	}

	tmpDir := path.Join(os.TempDir(), "dsiem")
	if err := EnsureDir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := path.Join(tmpDir, "file.txt")
	if err := AppendToFile("test", tmpFile); err != nil {
		t.Fatal(err)
	}

	if !FileExist(tmpFile) {
		t.Fatal("test file" + tmpFile + " doesnt exist")
	}

	if err := AppendToFile("test", "/proc"); err == nil {
		t.Fatal("o rly?")
	}
	if err := OverwriteFile("test", tmpFile); err != nil {
		t.Fatal(err)
	}
	if err := OverwriteFile("test", "/proc"); err == nil {
		t.Fatal("o rly?")
	}
}
