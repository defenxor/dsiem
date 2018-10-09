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
