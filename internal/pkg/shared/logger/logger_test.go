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

package logger

import (
	"strings"
	"testing"
)

func TestLog(t *testing.T) {

	if err := Setup(false); err != nil {
		t.Fatal(err)
	}
	Debug(M{})

	if err := Setup(true); err != nil {
		t.Fatal(err)
	}

	text := "test"
	i := 1
	s := "s"
	n := uint64(1)
	msgs := []M{
		{Msg: text},
		{Msg: text, DId: i},
		{Msg: text, BId: s},
		{Msg: text, CId: n},
		{Msg: text, DId: i, BId: s},
		{Msg: text, DId: i, CId: n},
		{Msg: text, BId: s, CId: n},
		{Msg: text, DId: i, BId: s, CId: n},
	}

	for _, m := range msgs {
		EnableTestingMode()
		o := CaptureZapOutput(func() {
			Info(m)
		})
		if !strings.Contains(o, "INFO") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
		o = CaptureZapOutput(func() {
			Warn(m)
		})
		if !strings.Contains(o, "WARN") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
		o = CaptureZapOutput(func() {
			Debug(m)
		})
		if !strings.Contains(o, "DEBUG") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
		o = CaptureZapOutput(func() {
			Error(m)
		})
		if !strings.Contains(o, "ERROR") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
	}
}
