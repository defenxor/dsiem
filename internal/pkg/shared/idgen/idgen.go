package idgen

import (
	"github.com/teris-io/shortid"
)

var sid *shortid.Shortid
var initErr error

func init() {
	sid, initErr = shortid.New(1, shortid.DEFAULT_ABC, 2342)
}

// GenerateID creates random shortid
func GenerateID() (id string, err error) {
	if initErr != nil {
		return "", initErr
	}
	return sid.Generate()
}
