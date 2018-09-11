package main

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/kardianos/osext"
)

func getDir() (string, error) {
	dir, err := osext.ExecutableFolder()

	if devEnv == true {
		dir = "/go/src/siem"
	}

	return dir, err
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func isDigit(v string) bool {
	if _, err := strconv.ParseInt(v, 10, 64); err == nil {
		return true
	}
	return false
}

func reftoDigit(v string) (int64, error) {
	i := strings.Index(v, ":")
	if i == -1 {
		return 0, errors.New("not a reference")
	}
	v = strings.Trim(v, ":")
	return strconv.ParseInt(v, 10, 64)
}

func isIPinCIDR(ip string, netcidr string, connID uint64) (found bool) {
	// first convert to slice, because netcidr maybe in a form of "cidr1,cidr2..."
	cleaned := strings.Replace(netcidr, ",", " ", -1)
	cidrSlice := strings.Fields(cleaned)

	found = false
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}
	ipB, _, err := net.ParseCIDR(ip)
	if err != nil {
		logWarn("Unable to parse IP address: "+ip+". Make sure the plugin is configured correctly!", connID)
		return
	}

	for _, v := range cidrSlice {
		if !strings.Contains(v, "/") {
			v = v + "/32"
		}
		_, ipnetA, err := net.ParseCIDR(v)
		if err != nil {
			logWarn("Unable to parse CIDR address: "+v+". Make sure the directive is configured correctly!", connID)
			return
		}
		if ipnetA.Contains(ipB) {
			found = true
			break
		}
	}
	return
}

func caseInsensitiveContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
}

func logInfo(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	logger.Info("[" + sID + "] " + msg)
}

func logWarn(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	logger.Warn("[" + sID + "] " + msg)
}

func appendStringUniq(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func removeDuplicatesUnordered(elements []string) []string {
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
