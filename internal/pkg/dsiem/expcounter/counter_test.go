package expcounter

import (
	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"time"

	"testing"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/spf13/viper"
)

func TestInit(t *testing.T) {
	// 	modes := []string{"standalone", "cluster-backend", "cluster-frontend"}
	log.Setup(false)
	viper.Set("tags", []string{"0"})
	viper.Set("status", []string{"Open"})
	// server.InitRcCounter()
	alarm.Init("doesntmatter")
	Init("standalone")
	startTicker("standalone", true)
	time.Sleep(6 * time.Second)
}

func TestInit2(t *testing.T) {
	Init("cluster-backend")
	startTicker("cluster-backend", true)
}

func TestInit3(t *testing.T) {
	Init("cluster-frontend")
	startTicker("cluster-frontend", true)
}
