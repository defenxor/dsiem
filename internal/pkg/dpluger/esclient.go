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
	"fmt"
	"strings"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

// esCollector is the interface for querying elasticsearch summaries
type esCollector interface {
	Init(esURL string) (err error)

	// Collect unqiue list of plugin SIDs from field marked as 'SID' field, with optional 'Category' field. Returns TSV reference, and any occurred error.
	Collect(plugin Plugin, confFile, sidSource, esFilter, categorySource string, shouldCollectCategory bool) (c tsvRef, err error)

	// Collect unique list of plugin SIDs from fields marked as 'title' field and 'SID' field, with optional 'Category' field. Returns TSV reference, and any occurred error.
	CollectPair(plugin Plugin, confFile, sidSource, esFilter, titleSource, categorySource string, shouldCollectCategory bool) (c tsvRef, err error)

	ValidateIndex(index string) (err error)
	IsESFieldExist(index string, field string) (exist bool, err error)
	FieldType(ctx context.Context, index, field string) (fieldType string, hasKeyword bool, err error)
}

func newESCollector(esURL string) (esCollector, error) {
	var esVersion int
	ver, err := elasticsearchVersion(esURL)
	if err != nil {
		return nil, err
	}

	log.InfoMsg(fmt.Sprintf("Found ES version '%s'", ver))
	if strings.HasPrefix(ver, "7") {
		esVersion = 7
		collector = &es7Client{}
	}

	if strings.HasPrefix(ver, "6") {
		esVersion = 6
		collector = &es6Client{}
	}

	if strings.HasPrefix(ver, "5") {
		esVersion = 5
		collector = &es5Client{}
	}

	if esVersion == 0 {
		return nil, fmt.Errorf("unsupported es version '%s', currently only ver 5.x, 6.x, 7.x are supported", ver)
	}

	if err := collector.Init(esURL); err != nil {
		return nil, err
	}

	return collector, err
}
