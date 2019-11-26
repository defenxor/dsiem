// Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.
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

package server

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
	xc "github.com/defenxor/dsiem/internal/pkg/dsiem/xcorrelator"
)

func isCfgFileNameValid(filename string) (ok bool) {
	r, err := regexp.Compile(`^[a-zA-Z0-9]([a-zA-Z0-9_-]+)?.json$`)
	if err != nil {
		return
	}
	ok = r.MatchString(filename)
	return
}

func isUploadContentValid(filename string, content []byte) (err error) {
	e := errors.New("content doesn't have a valid entry")
	switch {
	case strings.HasPrefix(filename, "assets_"):
		var v asset.NetworkAssets
		err = json.Unmarshal(content, &v)
		if err == nil && len(v.NetworkAssets) == 0 {
			err = e
		}
	case strings.HasPrefix(filename, "directives_"):
		var v siem.Directives
		err = json.Unmarshal(content, &v)
		if err == nil && len(v.Dirs) == 0 {
			err = e
		}
	case strings.HasPrefix(filename, "intel_"):
		var v xc.IntelSources
		err = json.Unmarshal(content, &v)
		if err == nil && len(v.IntelSources) == 0 {
			err = e
		}
	case strings.HasPrefix(filename, "vuln_"):
		var v xc.VulnSources
		err = json.Unmarshal(content, &v)
		if err == nil && len(v.VulnSources) == 0 {
			err = e
		}
	default:
	}
	return
}
