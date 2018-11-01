package test

import (
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

//DirEnv get the root app directory and setup log for testing
func DirEnv() (dir string, err error) {
	dir, err = fs.GetDir(true)
	if err == nil {
		err = log.Setup(false)
	}
	return
}
