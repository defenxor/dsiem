// Package vuln provides entry point for vulnerability lookup plugins
package vuln

import "context"

// Checker defines dispatch to extensions
type Checker interface {
	CheckIPPort(ctx context.Context, ip string, port int) (found bool, results []Result, err error)
	Initialize(config []byte) error
}

// Result defines
type Result struct {
	Provider string `json:"provider"`
	Term     string `json:"term"`
	Result   string `json:"result"`
}
