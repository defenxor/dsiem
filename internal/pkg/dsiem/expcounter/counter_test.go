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

package expcounter

import (
	"time"

	"testing"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/spf13/viper"
)

func TestInit(t *testing.T) {
	// 	modes := []string{"standalone", "cluster-backend", "cluster-frontend"}
	log.Setup(false)
	viper.Set("tags", []string{"0"})
	viper.Set("status", []string{"Open"})
	// server.InitRcCounter()
	// alarm.Init("doesntmatter", false)
	Init("standalone")
	startTicker("standalone", true)
	time.Sleep(6 * time.Second)
}

func TestInit2(t *testing.T) {
	Init("cluster-backend")
	startTicker("cluster-backend", true)
}

func TestInit3(t *testing.T) {
	Init("cluster-frontend")
	startTicker("cluster-frontend", true)
}
