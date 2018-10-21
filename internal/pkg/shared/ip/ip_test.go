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
