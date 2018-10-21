package test

import (
	"dsiem/internal/pkg/shared/fs"
	log "dsiem/internal/pkg/shared/logger"
)

//DirEnv get the root app directory and setup log for testing
func DirEnv() (dir string, err error) {
	dir, err = fs.GetDir(true)
	if err == nil {
		err = log.Setup(false)
	}
	return
}
