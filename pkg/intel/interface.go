// Package intel provides entry point for threat intel plugins
package intel

import "context"

// Checker defines the behaviour that must be implemented by an intel plugin
type Checker interface {
	CheckIP(ctx context.Context, ip string) (found bool, results []Result, err error)
	Initialize(config []byte) error
}

// Result defines the struct that must be returned by an intel plugin
type Result struct {
	Provider string `json:"provider"`
	Term     string `json:"term"`
	Result   string `json:"result"`
}
