package ip

import "net"

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

// IsPrivateIP check if IP is in private range
func IsPrivateIP(ip string) bool {
	ipn := net.ParseIP(ip)
	for _, block := range privateIPBlocks {
		if block.Contains(ipn) {
			return true
		}
	}
	return false
}
