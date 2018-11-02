package dpluger

import (
	"errors"
	"strings"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/olivere/elastic"
)

// esCollector is the interface for querying elasticsearch summaries
type esCollector interface {
	Init(esURL string) (err error)
	Collect(plugin Plugin, confFile, sidSource string) (c tsvRef, err error)
	ValidateIndex(index string) (err error)
	IsESFieldExist(index string, field string) (exist bool, err error)
}

func newESCollector(esURL string) (collector esCollector, err error) {
	esVersion := 0
	c, err := elastic.NewSimpleClient(elastic.SetURL(esURL))
	if err != nil {
		return
	}
	ver, err := c.ElasticsearchVersion(esURL)
	if err != nil {
		return
	}
	log.Info(log.M{Msg: "Found ES version " + ver})
	if strings.HasPrefix(ver, "6") {
		esVersion = 6
		collector = &es6Client{}
	}
	if strings.HasPrefix(ver, "5") {
		esVersion = 5
		collector = &es5Client{}
	}
	if esVersion == 0 {
		err = errors.New("Unsupported ES version (" + ver + "), currently only ver 5.x and 6.x are supported.")
		return
	}
	err = collector.Init(esURL)
	return
}
