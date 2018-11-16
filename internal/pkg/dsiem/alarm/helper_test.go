package alarm

import (
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

func TestHelper(t *testing.T) {
	initDirAndLog(t)
	t.Logf("Enabling log test mode")
	log.EnableTestingMode()
	a := alarm{}
	apm.Enable(true)
	aLogFile = ""

	tx := apm.StartTransaction("test", "test", nil)
	verifyFuncOutput(t, func() {
		updateElasticsearch(&a, "test", 1, tx)
	}, "failed to update Elasticsearch", true)

}
