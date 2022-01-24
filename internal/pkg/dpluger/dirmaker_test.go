package dpluger

import (
	"strings"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
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

func TestDirectiveDoubleQuoteTitle(t *testing.T) {
	log.Setup(true)
	const (
		kingdom       = "DEFAULT"
		titleTemplate = "EVENT_TITLE (SRC_IP to DST_IP)"
		priority      = 3
		reliability   = 1
		dirnumber     = 100000
	)

	in := strings.NewReader(`plugin	id	sid	title	category
suricata	9001	2009477	SQLBrute SQL Scan Detected	Attempted Information Leak
suricata	9001	2009040	SQLNinja MSSQL User Scan"; content:"?param=a	Attempted Information Leak
suricata	9001	2009041	SQLNinja MSSQL Database User Rights Scan	Attempted Information Leak`)

	var dirs siem.Directives
	var err error
	dirs, err = createDirective(in, dirs, kingdom, titleTemplate, priority, reliability, dirnumber)
	if err != nil {
		t.Fatal(err.Error())
	}

	_ = dirs
}
