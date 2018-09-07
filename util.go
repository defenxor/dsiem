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

	// only during debugging
	dir = "/home/mmta/go/src/siem"

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

func isIPinCIDR(ip string, netcidr string) bool {
	if ip == "" || netcidr == "" {
		return false
	}
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}
	if !strings.Contains(netcidr, "/") {
		netcidr = netcidr + "/32"
	}
	_, ipnetA, _ := net.ParseCIDR(netcidr)
	ipB, _, _ := net.ParseCIDR(ip)

	return ipnetA.Contains(ipB)
}

func logInfo(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	logger.Info("[" + sID + "] " + msg)
}

func logWarn(msg string, connID uint64) {
	sID := strconv.Itoa(int(connID))
	logger.Warn("[" + sID + "] " + msg)
}
