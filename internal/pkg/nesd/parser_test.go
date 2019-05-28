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

package nesd

import (
	"os"
	"path"
	"testing"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

var csvInitialized bool

func TestInitCSV(t *testing.T) {

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	log.Setup(true)
	err = InitCSV(`/\/\/\/`)
	if err == nil {
		t.Error("Expected error due to bad path")
	}
	csvDir := path.Join(dir, "fixtures")
	err = InitCSV(csvDir)
	if err == nil {
		t.Error("Expected error due to empty result")
	}
	csvDir = path.Join(dir, "fixtures", "example1")
	err = InitCSV(csvDir)
	if err == nil {
		t.Fatal("expected parsing error")
	}
	csvDir = path.Join(dir, "fixtures", "example2")
	err = InitCSV(csvDir)
	if err != nil {
		t.Fatal(err)
	}
	csvInitialized = true

}
