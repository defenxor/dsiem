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

package alarm

import (
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

func TestHelper(t *testing.T) {
	initDirAndLog(t)
	t.Logf("Enabling log test mode")
	log.EnableTestingMode()
	a := alarm{Risk: 1}
	apm.Enable(true)
	// tmpLog := path.Join(os.TempDir(), "foo.log")
	fWriter.Init("", 10)

	tx := apm.StartTransaction("test", "test", nil, nil)
	verifyFuncOutput(t, func() {
		updateElasticsearch(&a, "test", 1, tx)
	}, "failed to update Elasticsearch", true)

}
