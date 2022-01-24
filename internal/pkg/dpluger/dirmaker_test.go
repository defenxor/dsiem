package dpluger

import (
	"testing"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

func TestCreateDirective(t *testing.T) {
	log.Setup(true)
	const (
		testdata      = "testdata/x_plugin-sid-test.tsv"
		outfile       = "testdata/dev_out-test-x.json"
		kingdom       = "TEST"
		titleTemplate = "EVENT_TITLE (SRC_IP to DST_IP)"
		priority      = 3
		reliability   = 1
		dirnumber     = 100000
	)

	err := CreateDirective(testdata, outfile, kingdom, titleTemplate, priority, reliability, dirnumber)
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO: load the created directive here and validate.
}
