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
	"fmt"
	"reflect"
	"testing"
)

func TestAppenUniq(t *testing.T) {
	type s1 struct {
		input    string
		expected []string
	}
	tbl1 := []s1{
		{"a", []string{"a"}},
		{"a", []string{"a"}},
		{"b", []string{"a", "b"}},
		{"c", []string{"a", "b", "c"}},
	}

	actual := []string{}
	for _, tt := range tbl1 {
		actual = AppendUniq(actual, tt.input)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("AppendUniq: input %v, expected %v, actual %v", tt.input, tt.expected, actual)
		}
	}
}

func TestCaseInsensitiveContains(t *testing.T) {
	type s1 struct {
		text     string
		term     string
		expected bool
	}
	tbl1 := []s1{
		{"test string", "test", true},
		{"TEST string", "test", true},
		{"TEST string", "strong", false},
	}

	for _, tt := range tbl1 {
		actual := CaseInsensitiveContains(tt.text, tt.term)
		if actual != tt.expected {
			t.Errorf("CaseInsensitiveContains: text %v, term %v, expected %v, actual %v",
				tt.text, tt.term, tt.expected, actual)
		}
	}
}

func TestIsInCSVList(t *testing.T) {
	type s1 struct {
		text     string
		term     string
		expected bool
	}
	tbl1 := []s1{
		{"10.0.0.1, 10.0.0.2", "10.0", false},
		{"10.0.0.1, 10.0.0.2", "10.0.0.1", true},
		{"", "", false},
	}

	for _, tt := range tbl1 {
		actual := IsInCSVList(tt.text, tt.term)
		if actual != tt.expected {
			t.Errorf("IsInCSVList: text %v, term %v, expected %v, actual %v",
				tt.text, tt.term, tt.expected, actual)
		}
	}
}

func TestCsvToSlice(t *testing.T) {
	type s1 struct {
		text     string
		expected []string
	}
	tbl1 := []s1{
		{"1,2,3", []string{"1", "2", "3"}},
		{"123", []string{"123"}},
		{"", []string{}},
	}

	for _, tt := range tbl1 {
		actual := CsvToSlice(tt.text)
		if !reflect.DeepEqual(actual, tt.expected) {
			fmt.Println(actual)
			fmt.Println(tt.expected)
			t.Errorf("CsvToSlice: text %v,  expected %v, actual %v",
				tt.text, tt.expected, actual)
		}
	}
}

func TestRefToDigit(t *testing.T) {
	type s1 struct {
		text       string
		expected   int64
		expectedOk bool
	}
	tbl1 := []s1{
		{"noref", 0, false},
		{"", 0, false},
		{":1", 1, true},
		{":123", 123, true},
	}

	for _, tt := range tbl1 {
		actual, ok := RefToDigit(tt.text)
		if ok != tt.expectedOk {
			t.Errorf("RefToDigit: text %v, expected Ok value: %v, actual: %v", tt.text, tt.expectedOk, ok)
		} else {
			if actual != tt.expected {
				t.Errorf("RefToDigit: text %v,  expected %v, actual %v",
					tt.text, tt.expected, actual)
			}
		}
	}
}

func TestTimeStampToUnix(t *testing.T) {
	type s1 struct {
		text      string
		expected  int64
		shouldErr bool
	}
	tbl1 := []s1{
		{"non-date", 0, true},
		{"2018-10-21T16:32:12+07:00", 1540114332, false},
	}

	for _, tt := range tbl1 {
		actual, err := TimeStampToUnix(tt.text)
		if err != nil && !tt.shouldErr {
			t.Errorf("TimeStampToUnix: text %v, expected err: %v, actual: %v", tt.text, tt.shouldErr, err)
		} else {
			if actual != tt.expected {
				t.Errorf("TimeStampToUnix: text %v,  expected %v, actual %v",
					tt.text, tt.expected, actual)
			}
		}
	}
}

func TestRemoveDuplicatesUnordered(t *testing.T) {
	type s1 struct {
		elements []string
		expected []string
	}
	tbl1 := []s1{
		{[]string{"1", "1", "2"}, []string{"2", "1"}},
		{[]string{"1", "2"}, []string{"2", "1"}},
		{[]string{}, []string{}},
	}

	for _, tt := range tbl1 {
		actual := RemoveDuplicatesUnordered(tt.elements)
		if !sameStringSlice(actual, tt.expected) {
			t.Errorf("RemoveDuplicatesUnordered: elements %v,  expected %v, actual %v",
				tt.elements, tt.expected, actual)
		}
	}
}

func TestUniqStringSlice(t *testing.T) {
	type s1 struct {
		list     string
		expected []string
	}
	tbl1 := []s1{
		{"1,2,3", []string{"1", "2", "3"}},
		{"1,2,2", []string{"1", "2"}},
	}

	for _, tt := range tbl1 {
		actual := UniqStringSlice(tt.list)
		if !sameStringSlice(actual, tt.expected) {
			fmt.Println(actual)
			fmt.Println(tt.expected)
			t.Errorf("UniqStringSlice: list %v,  expected %v, actual %v",
				tt.list, tt.expected, actual)
		}
	}
}

func TestRemoveElementUnlessEmpty(t *testing.T) {
	type s1 struct {
		elements []string
		target   string
		expected []string
	}
	tbl1 := []s1{
		{[]string{"1"}, "1", []string{"1"}},
		{[]string{"1", "2", "3"}, "1", []string{"2", "3"}},
		{[]string{"1", "2", "3"}, "4", []string{"1", "2", "3"}},
	}

	for _, tt := range tbl1 {
		actual := RemoveElementUnlessEmpty(tt.elements, tt.target)
		if !sameStringSlice(actual, tt.expected) {
			fmt.Println(actual)
			fmt.Println(tt.expected)
			t.Errorf("RemoveElementUnlessEmpty: elements %v,  target %v, expected %v, actual %v",
				tt.elements, tt.target, tt.expected, actual)
		}
	}
}

// https://stackoverflow.com/questions/36000487/check-for-equality-on-slices-without-order
func sameStringSlice(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(x))
	for _, _x := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	return len(diff) == 0
}

func TestTrimLeftChar(t *testing.T) {
	type s1 struct {
		text     string
		expected string
	}
	tbl1 := []s1{
		{"test string", "est string"},
		{"TEST", "EST"},
		{"", ""},
	}

	for _, tt := range tbl1 {
		actual := TrimLeftChar(tt.text)
		if actual != tt.expected {
			t.Errorf("TrimLeftChar: text %v, expected %v, actual %v",
				tt.text, tt.expected, actual)
		}
	}
}
