package test

import (
	"dsiem/internal/pkg/shared/fs"
	log "dsiem/internal/pkg/shared/logger"
	"testing"
)

//DirEnv get the root app directory and setup log for testing
func DirEnv(t *testing.T) (dir string) {
	dir, err := fs.GetDir(true)
	if err != nil {
		t.Fatal(err)
	}
	err = log.Setup(false)
	if err != nil {
		t.Fatal(err)
	}
	return
}
