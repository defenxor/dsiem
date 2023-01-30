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

package ip

import (
	"testing"
)

func TestIP(t *testing.T) {
	type ipTests struct {
		ip        string
		expected  bool
		shouldErr bool
	}
	tbl := []ipTests{
		{"127.0.0.1", true, false},
		{"10.0.0.1", true, false},
		{"192.168.0.1", true, false},
		{"172.16.0.1", true, false},
		{"8.8.8.8", false, false},
		{"not-an-ip", false, true},
		{"fd12:3456:789a:1::1", true, false},
		{"fb00:3456:789a:1::1", false, false},
	}

	for _, tt := range tbl {
		actual, err := IsPrivateIP(tt.ip)
		if err != nil && !tt.shouldErr {
			t.Errorf("IsPrivateIP %v, expected err: %v, actual: %v", tt.ip, tt.shouldErr, err)
		}
		if actual != tt.expected {
			t.Errorf("IsPrivateIP %v, expected: %v, actual: %v", tt.ip, tt.expected, actual)
		}
	}
}
