package idgen

import (
	"github.com/teris-io/shortid"
)

var sid, _ = shortid.New(1, shortid.DEFAULT_ABC, 2342)

// GenerateID creates random shortid
func GenerateID() (id string, err error) {
	return sid.Generate()
}
