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

package siem

import (
	"path"
	"strings"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"
)

func TestDirectiveInit(t *testing.T) {
	d, err := test.DirEnv()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Using base dir %s", d)
	fDir := path.Join(d, "internal", "pkg", "dsiem", "siem", "fixtures")
	var evtChan chan event.NormalizedEvent
	err = InitDirectives(path.Join(fDir, "directive2"), evtChan)
	if err == nil || !strings.Contains(err.Error(), "Cannot load any directive from") {
		t.Fatal(err)
	}
	err = InitDirectives(path.Join(fDir, "directive1"), evtChan)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 5)

}
