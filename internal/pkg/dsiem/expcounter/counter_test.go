package expcounter

import (
	"dsiem/internal/pkg/dsiem/alarm"
	"dsiem/internal/pkg/dsiem/server"
	log "dsiem/internal/pkg/shared/logger"
	"testing"

	"github.com/spf13/viper"
)

func TestInit(t *testing.T) {
	// 	modes := []string{"standalone", "cluster-backend", "cluster-frontend"}
	log.Setup(false)
	viper.Set("tags", []string{"0"})
	viper.Set("status", []string{"Open"})
	server.InitRcCounter()
	alarm.Init("doesntmatter")
	Init("standalone")
	startTicker("standalone", true)
}

func TestInit2(t *testing.T) {
	Init("cluster-backend")
	startTicker("cluster-backend", true)
}

func TestInit3(t *testing.T) {
	Init("cluster-frontend")
	startTicker("cluster-frontend", true)
}
