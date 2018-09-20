package idgen

import (
	"github.com/teris-io/shortid"
)

var sid *shortid.Shortid

// GenerateID creates random shortid
func GenerateID() (id string, err error) {
	if sid == nil {
		sid, err = shortid.New(1, shortid.DEFAULT_ABC, 2342)
	}
	return sid.Generate()
}
