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

package str

import (
	"strconv"
	"strings"
	"time"
)

// AppendUniq append string to slice if it its not there yet
func AppendUniq(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

// CaseInsensitiveContains perform case-insensitive search of substr in s
func CaseInsensitiveContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
}

// IsInCSVList find term in s, where s is in the form of "string, string,string ..."
func IsInCSVList(s string, term string) (found bool) {
	// first convert to slice, because netcidr maybe in a form of "cidr1,cidr2..."
	sSlice := CsvToSlice(s)
	for _, v := range sSlice {
		if v != term {
			continue
		}
		found = true
		break
	}
	return
}

// CsvToSlice convert s to []string; where s is in the form of string, string, string
func CsvToSlice(s string) []string {
	cleaned := strings.Replace(s, ",", " ", -1)
	sSlice := strings.Fields(cleaned)
	return sSlice
}

// RefToDigit convert references in rules like :1 :2 :3 to 1 2 3
func RefToDigit(v string) (ret int64, ok bool) {
	i := strings.Index(v, ":")
	if i == -1 {
		return
	}
	v = strings.Trim(v, ":")
	ret, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		ok = true
	}
	return
}

// TimeStampToUnix converts s in RFC3339 format to epoch
func TimeStampToUnix(s string) (int64, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// RemoveDuplicatesUnordered remove duplicates from elements
func RemoveDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

// UniqStringSlice removes duplicate from a comma separated list into result slice
func UniqStringSlice(cslist string) (result []string) {
	s := strings.Split(cslist, ",")
	result = RemoveDuplicatesUnordered(s)
	return
}

// RemoveElementUnlessEmpty remove string element from slice unless doing so
// will result in an empty slice. This assumes entries in slice are uniq.
func RemoveElementUnlessEmpty(slice []string, target string) []string {
	if len(slice) == 1 {
		return slice
	}
	for i, v := range slice {
		if v == target {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// TrimLeftChar remove the first char in string and trim any whitespaces from the result
func TrimLeftChar(s string) string {
	for i := range s {
		if i > 0 {
			return strings.TrimSpace(s[i:])
		}
	}
	return s[:0]
}
