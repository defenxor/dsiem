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

package pprof

import (
	"testing"
)

func TestPProf(t *testing.T) {
	prof := []string{"cpu", "memory", "mutex", "block"}

	for _, p := range prof {
		f, err := GetProfiler(p)
		if err != nil {
			t.Fatal("Cannot start profiler: " + err.Error())
		}
		f.Stop()
	}
	_, err := GetProfiler("invalid")
	if err == nil {
		t.Fatal("invalid profiler should results in non-nil err")
	}
}
