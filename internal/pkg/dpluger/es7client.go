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

	elastic7 "github.com/olivere/elastic/v7"
)

type es7Client struct {
	client *elastic7.Client
}

func (es *es7Client) Init(esURL string) (err error) {
	es.client, err = elastic7.NewSimpleClient(elastic7.SetURL(esURL))
	return
}

func (es *es7Client) CollectPair(plugin Plugin, confFile, sidSource, esFilter, titleSource, categorySource string, shouldCollectCategory bool) (c tsvRef, err error) {
	size := 1000
	c.init(plugin.Name, confFile)
	var finalAgg, subSubTerm *elastic7.TermsAggregation
	rootTerm := elastic7.NewTermsAggregation().Field(titleSource).Size(size)
	subTerm := elastic7.NewTermsAggregation().Field(sidSource)
	finalAgg = rootTerm.SubAggregation("subterm", subTerm)
	if shouldCollectCategory {
		subSubTerm = elastic7.NewTermsAggregation().Field(categorySource)
		finalAgg = finalAgg.SubAggregation("subSubTerm", subSubTerm)
	}

	query := elastic7.NewBoolQuery()
	if esFilter != "" {
		coll := strings.Split(esFilter, ";")
		for _, v := range coll {
			s := strings.Split(v, "=")
			if len(s) != 2 {
				err = errors.New("Cannot split the ES filter term")
				return
			}
			query = query.Must(elastic7.NewTermQuery(s[0], s[1]))
		}
	} else {
		query = query.Must(elastic7.NewMatchAllQuery())
	}

	ctx := context.Background()
	searchResult, err := es.client.Search().
		Index(plugin.Index).
		Query(query).
		Aggregation("finalAgg", finalAgg).
		Pretty(true).
		Do(ctx)
	if err != nil {
		return
	}
	agg, found := searchResult.Aggregations.Terms("finalAgg")
	if !found {
		err = errors.New("cannot find aggregation finalAgg in ES query result")
		return
	}
	count := len(agg.Buckets)
	if count == 0 {
		err = errors.New("cannot find matching entry in field " + sidSource + " on index " + plugin.Index)
		return
	}
	fmt.Println("Found", count, "uniq "+sidSource+".")
	nID, err := strconv.Atoi(plugin.Fields.PluginID)
	if err != nil {
		return
	}

	for _, lvl1Bucket := range agg.Buckets {
		subterm, found := lvl1Bucket.Terms("subterm")
		if !found {
			continue
		}
		for _, lvl2Bucket := range subterm.Buckets {
			sKey := lvl1Bucket.Key.(string)
			nKey := int(lvl2Bucket.Key.(float64))
			// fmt.Println("item1:", sKey, "item2:", nKey)
			if shouldCollectCategory {
				subSubTerm, found2 := lvl1Bucket.Terms("subSubTerm")
				if !found2 {
					continue
				}
				for _, lvl3Bucket := range subSubTerm.Buckets {
					sCat := lvl3Bucket.Key.(string)
					_ = c.upsert(plugin.Name, nID, &nKey, sCat, sKey)
					break
				}
			} else {
				_ = c.upsert(plugin.Name, nID, &nKey, categorySource, sKey)
			}
			break
		}
	}
	return
}

func (es *es7Client) Collect(plugin Plugin, confFile, sidSource, esFilter, categorySource string, shouldCollectCategory bool) (c tsvRef, err error) {

	size := 1000
	c.init(plugin.Name, confFile)
	var subTerm *elastic7.TermsAggregation
	terms := elastic7.NewTermsAggregation().Field(sidSource).Size(size)
	if shouldCollectCategory {
		subTerm = elastic7.NewTermsAggregation().Field(categorySource)
		terms = terms.SubAggregation("subTerm", subTerm)
	}

	query := elastic7.NewBoolQuery()
	if esFilter != "" {
		coll := strings.Split(esFilter, ";")
		for _, v := range coll {
			s := strings.Split(v, "=")
			if len(s) != 2 {
				err = errors.New("Cannot split the ES filter term")
				return
			}
			query = query.Must(elastic7.NewTermQuery(s[0], s[1]))
		}
	} else {
		query = query.Must(elastic7.NewMatchAllQuery())
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
		if !shouldCollectCategory {
			// increase SID counter only if the last entry
			if shouldIncrease := c.upsert(plugin.Name, nID, &newSID, categorySource, t); shouldIncrease {
				newSID++
			}
		} else {
			subterm, found := titleBucket.Terms("subTerm")
			if !found {
				continue
			}
			for _, lvl2Bucket := range subterm.Buckets {
				sCat := lvl2Bucket.Key.(string)
				// increase SID counter only if the last entry
				if shouldIncrease := c.upsert(plugin.Name, nID, &newSID, sCat, t); shouldIncrease {
					newSID++
				}
				break
			}
		}
	}
	return
}

func (es *es7Client) ValidateIndex(index string) (err error) {
	var exists bool
	exists, err = es.client.IndexExists(index).Do(context.Background())
	if err == nil && !exists {
		err = errors.New("Index " + index + " does not exist")
	}
	return
}

func (es *es7Client) IsESFieldExist(index string, field string) (exist bool, err error) {
	var countResult int64
	existQuery := elastic7.NewExistsQuery(field)
	countService := elastic7.NewCountService(es.client)
	countResult, err = countService.Index(index).
		Query(existQuery).
		Pretty(true).
		Do(context.Background())
	if countResult > 0 {
		exist = true
	}
	return
}
