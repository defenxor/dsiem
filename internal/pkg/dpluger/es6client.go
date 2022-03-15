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

func (es *es6Client) CollectPair(plugin Plugin, confFile, sidSource, esFilter, titleSource, categorySource string, shouldCollectCategory bool) (c tsvRef, err error) {
	size := 1000
	c.init(plugin.Name, confFile)
	var finalAgg, subSubTerm *elastic6.TermsAggregation
	rootTerm := elastic6.NewTermsAggregation().Field(titleSource).Size(size)
	subTerm := elastic6.NewTermsAggregation().Field(sidSource)
	finalAgg = rootTerm.SubAggregation("subterm", subTerm)
	if shouldCollectCategory {
		subSubTerm = elastic6.NewTermsAggregation().Field(categorySource)
		finalAgg = finalAgg.SubAggregation("subSubTerm", subSubTerm)
	}

	query := elastic6.NewBoolQuery()
	if esFilter != "" {
		filters := strings.Split(esFilter, ";")
		for _, filter := range filters {
			s := strings.Split(filter, "=")
			if len(s) != 2 {
				return tsvRef{}, fmt.Errorf("invalid ES filter term, '%s', expected pair of strings with '=' delimitier", filter)
			}
			query = query.Must(elastic6.NewTermQuery(s[0], s[1]))
		}
	} else {
		query = query.Must(elastic6.NewMatchAllQuery())
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

	roots, found := searchResult.Aggregations.Terms("finalAgg")
	if !found {
		err = errors.New("cannot find aggregation finalAgg in ES query result")
		return
	}
	count := len(roots.Buckets)
	if count == 0 {
		err = errors.New("cannot find matching entry in field " + sidSource + " on index " + plugin.Index)
		return
	}

	fmt.Printf("found %d unique '%s'\n", count, sidSource)
	nID, err := strconv.Atoi(plugin.Fields.PluginID)
	if err != nil {
		return
	}

	for _, rootBucket := range roots.Buckets {
		sidlist, found := rootBucket.Terms("subterm")
		if !found {
			continue
		}

		for _, sidBucket := range sidlist.Buckets {
			root := rootBucket.Key.(string)
			sid, err := toInt(sidBucket.Key)
			if err != nil {
				return c, fmt.Errorf("invalid signature ID, %s", err.Error())
			}
			// fmt.Println("item1:", sKey, "item2:", nKey)
			if shouldCollectCategory {
				subSubTerm, found2 := rootBucket.Terms("subSubTerm")
				if !found2 {
					continue
				}
				for _, lvl3Bucket := range subSubTerm.Buckets {
					sCat := lvl3Bucket.Key.(string)
					_ = c.upsert(plugin.Name, nID, &sid, sCat, root)
					break
				}
			} else {
				_ = c.upsert(plugin.Name, nID, &sid, categorySource, root)
			}
			break
		}
	}
	return
}

func (es *es6Client) Collect(plugin Plugin, confFile, sidSource, esFilter, categorySource string, shouldCollectCategory bool) (c tsvRef, err error) {

	size := 1000
	c.init(plugin.Name, confFile)
	var subTerm *elastic6.TermsAggregation
	terms := elastic6.NewTermsAggregation().Field(sidSource).Size(size)
	if shouldCollectCategory {
		subTerm = elastic6.NewTermsAggregation().Field(categorySource)
		terms = terms.SubAggregation("subTerm", subTerm)
	}

	query := elastic6.NewBoolQuery()
	if esFilter != "" {
		filters := strings.Split(esFilter, ";")
		for _, filter := range filters {
			s := strings.Split(filter, "=")
			if len(s) != 2 {
				return tsvRef{}, fmt.Errorf("invalid ES filter term, '%s', expected pair of strings with '=' delimitier", filter)
			}
			query = query.Must(elastic6.NewTermQuery(s[0], s[1]))
		}
	} else {
		query = query.Must(elastic6.NewMatchAllQuery())
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

	fmt.Printf("found %d unique '%s'\n", count, sidSource)
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

func (es *es6Client) FieldType(ctx context.Context, index string, field string) (string, bool, error) {
	m, err := elastic6.NewGetFieldMappingService(es.client).
		Field(field).
		Index(index).
		Do(ctx)

	if err != nil {
		return "", false, err
	}

	var fiedMapping map[string]interface{}
	var ok bool
MAPPING_SEARCH:
	for _, v := range m {
		fm, fmok := v.(map[string]interface{})["mappings"].(map[string]interface{})
		if !fmok {
			continue
		}

		for _, val := range fm {
			// get the first child that has the mapping for the field
			fiedMapping, ok = val.(map[string]interface{})[field].(map[string]interface{})
			if ok {
				break MAPPING_SEARCH
			}
		}

	}

	if !ok || fiedMapping == nil {
		return "", false, ErrFieldMappingNotExist
	}

	levels := strings.Split(field, ".")
	level := levels[len(levels)-1]

	mapping, ok := fiedMapping["mapping"].(map[string]interface{})[level].(map[string]interface{})
	if !ok || mapping == nil {
		return "", false, ErrFieldMappingNotExist
	}

	fieldType, ok := mapping["type"].(string)
	if !ok {
		return "", false, ErrFieldMappingNotExist
	}

	var iskeyword bool
	keyword, ok := mapping["fields"].(map[string]interface{})
	if ok && keyword != nil {
		kword, ok := keyword["keyword"].(map[string]interface{})
		if ok && kword != nil {
			ktype, ok := kword["type"].(string)
			if ok && ktype == "keyword" {
				iskeyword = true
			}
		}
	}

	return fieldType, iskeyword, nil
}
