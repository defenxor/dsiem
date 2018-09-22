package xcorrelator

import (
	"regexp"
	"strings"
)

func matcherRegexIntel(body []byte, provider string, term string, strRegex []string) (found bool, results []IntelResult) {
	vResult := string(body)
	// loop over strRegex, applying it one by one to vResult
	for _, v := range strRegex {
		if strings.HasPrefix(v, "match:") {
			r := strings.Split(v, ":")
			re := regexp.MustCompile(r[len(r)-1])
			s := re.FindAllString(vResult, -1)
			if s == nil {
				vResult = ""
				break
			}
			vResult = s[len(s)-1]
		}
		if strings.HasPrefix(v, "remove:") {
			r := strings.Split(v, ":")
			re := regexp.MustCompile(r[len(r)-1])
			s := re.ReplaceAllLiteralString(vResult, "")
			if s == "" {
				vResult = ""
				break
			}
			vResult = s
		}
	}
	vResult = strings.Trim(vResult, " ")
	if vResult != "" {
		results = append(results, IntelResult{provider, term, vResult})
		found = true
	}
	return
}

func matcherRegexVuln(body []byte, provider string, term string, strRegex []string) (found bool, results []VulnResult) {
	vResult := string(body)
	// loop over strRegex, applying it one by one to vResult
	for _, v := range strRegex {
		if strings.HasPrefix(v, "match:") {
			r := strings.Split(v, ":")
			re := regexp.MustCompile(r[len(r)-1])
			s := re.FindAllString(vResult, -1)
			if s == nil {
				vResult = ""
				break
			}
			vResult = s[len(s)-1]
		}
		if strings.HasPrefix(v, "remove:") {
			r := strings.Split(v, ":")
			re := regexp.MustCompile(r[len(r)-1])
			s := re.ReplaceAllLiteralString(vResult, "")
			if s == "" {
				vResult = ""
				break
			}
			vResult = s
		}
	}
	vResult = strings.Trim(vResult, " ")
	if vResult != "" {
		results = append(results, VulnResult{provider, term, vResult})
		found = true
	}
	return
}
