// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package dpluger

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	elastic6 "github.com/olivere/elastic"
)

type es6Client struct {
	client *elastic6.Client
}

func (es *es6Client) Init(esURL string) (err error) {
	es.client, err = elastic6.NewSimpleClient(elastic6.SetURL(esURL))
	return
}
func (es *es6Client) Collect(plugin Plugin, confFile, sidSource, esFilter string) (c tsvRef, err error) {

	size := 1000
	c.init(plugin.Name, confFile)
	terms := elastic6.NewTermsAggregation().Field(sidSource).Size(size)
	var query elastic6.Query
	if esFilter != "" {
		s := strings.Split(esFilter, "=")
		if len(s) != 2 {
			err = errors.New("Cannot split the ES filter term")
			return
		}
		query = elastic6.NewTermsQuery(s[0], s[1])
	} else {
		query = elastic6.NewMatchAllQuery()
	}

	ctx := context.Background()
	searchResult, err := es.client.Search().
		Index(plugin.Index).
		Query(query).
		Aggregation("uniqTerm", terms).
		Pretty(true).
		Do(ctx)
	if err != nil {
		return
	}
	agg, found := searchResult.Aggregations.Terms("uniqTerm")
	if !found {
		err = errors.New("cannot find aggregation uniqTerm in ES query result")
		return
	}
	count := len(agg.Buckets)
	if count == 0 {
		err = errors.New("cannot find matching entry in field " + sidSource + " on index " + plugin.Index)
		return
	}
	fmt.Println("Found", count, "uniq "+sidSource+".")
	newSID := 1
	nID, err := strconv.Atoi(plugin.Fields.PluginID)
	if err != nil {
		return
	}
	for _, titleBucket := range agg.Buckets {
		t := titleBucket.Key.(string)
		// fmt.Println("found title:", t)
		// increase SID counter only if the last entry
		if shouldIncrease := c.upsert(plugin.Name, nID, &newSID, t); shouldIncrease {
			newSID++
		}
	}
	return
}

func (es *es6Client) ValidateIndex(index string) (err error) {
	var exists bool
	exists, err = es.client.IndexExists(index).Do(context.Background())
	if err == nil && !exists {
		err = errors.New("Index " + index + " does not exist")
	}
	return
}

func (es *es6Client) IsESFieldExist(index string, field string) (exist bool, err error) {
	var countResult int64
	existQuery := elastic6.NewExistsQuery(field)
	countService := elastic6.NewCountService(es.client)
	countResult, err = countService.Index(index).
		Query(existQuery).
		Pretty(true).
		Do(context.Background())
	if countResult > 0 {
		exist = true
	}
	return
}
