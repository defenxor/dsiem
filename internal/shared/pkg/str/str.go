package str

import (
	"strings"
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
