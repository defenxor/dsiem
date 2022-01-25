package dpluger

import (
	"reflect"
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

	dirs, _, err := siem.LoadDirectivesFromFile("testdata", "dev_out-*.json", false)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(dirs.Dirs) != 3 {
		t.Fatalf("expected 3 directives, but got '%d'", len(dirs.Dirs))
	}
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

	if len(dirs.Dirs) != 3 {
		t.Fatalf("expected 3 directives, but got %d", len(dirs.Dirs))
	}
}

func TestOptionalKingdom(t *testing.T) {
	log.Setup(true)
	const (
		kingdom       = "DEFAULT"
		titleTemplate = "EVENT_TITLE (SRC_IP to DST_IP)"
		priority      = 3
		reliability   = 1
		dirnumber     = 100000
	)

	in := strings.NewReader(`plugin	id	sid	title	category	kingdom
test	1337	1337001	Directive 1	Testing Directive
test	1337	1337002	Directive 2 with Kingdom	Testing Directive 2	TEST
test	1337	1337003	Directive 3	Testing Directive 3`)

	var dirs siem.Directives
	var err error
	dirs, err = createDirective(in, dirs, kingdom, titleTemplate, priority, reliability, dirnumber)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(dirs.Dirs) != 3 {
		t.Fatalf("expected 3 new directives, but got %d", len(dirs.Dirs))
	}

	dir := dirs.Dirs[0]
	if dir.Name != "Directive 1 (SRC_IP to DST_IP)" {
		t.Errorf("expected first directive name to be '%s' but got '%s'", "Directive 1 (SRC_IP to DST_IP)", dir.Name)
	}

	if dir.Category != "Testing Directive" {
		t.Errorf("expected first directive name to be '%s' but got '%s'", "Testing Directive", dir.Category)
	}

	if dir.ID != 100000 {
		t.Errorf("expected first directive ID to be %d but got %d", 100000, dir.ID)
	}

	for _, rule := range dir.Rules {
		var foundSID bool
		for _, id := range rule.PluginSID {
			if id == 1337001 {
				foundSID = true
				break
			}
		}

		if !foundSID {
			t.Errorf("expected plugin rules to contain sid %d", 1337001)
		}

		if rule.PluginID != 1337 {
			t.Errorf("expected rule plugin ID to be %d but got %d", 1337, rule.PluginID)
		}
	}

	if dir.Kingdom != kingdom {
		t.Errorf("expected first directive kingdom to be '%s' but got '%s'", kingdom, dir.Kingdom)
	}

	// check second rule
	dir = dirs.Dirs[1]
	if dir.Name != "Directive 2 with Kingdom (SRC_IP to DST_IP)" {
		t.Errorf("expected second directive name to be '%s' but got '%s'", "Directive 2 with Kingdom (SRC_IP to DST_IP)", dir.Name)
	}

	if dir.Category != "Testing Directive 2" {
		t.Errorf("expected second directive name to be '%s' but got '%s'", "Testing Directive 2", dir.Category)
	}

	if dir.ID != 100001 {
		t.Errorf("expected second directive ID to be %d but got %d", 100001, dir.ID)
	}

	for _, rule := range dir.Rules {
		var foundSID bool
		for _, id := range rule.PluginSID {
			if id == 1337002 {
				foundSID = true
				break
			}
		}

		if !foundSID {
			t.Errorf("expected plugin rules to contain sid %d", 1337001)
		}

		if rule.PluginID != 1337 {
			t.Errorf("expected rule plugin ID to be %d but got %d", 1337, rule.PluginID)
		}
	}

	if dir.Kingdom != "TEST" {
		t.Errorf("expected second directive kingdom to be '%s' but got '%s'", "TEST", dir.Kingdom)
	}

	// check third directive
	dir = dirs.Dirs[2]
	if dir.Name != "Directive 3 (SRC_IP to DST_IP)" {
		t.Errorf("expected third directive name to be '%s' but got '%s'", "Directive 3 (SRC_IP to DST_IP)", dir.Name)
	}

	if dir.Category != "Testing Directive 3" {
		t.Errorf("expected third directive name to be '%s' but got '%s'", "Testing Directive 3", dir.Category)
	}

	if dir.ID != 100002 {
		t.Errorf("expected third directive ID to be %d but got %d", 100000, dir.ID)
	}

	for _, rule := range dir.Rules {
		var foundSID bool
		for _, id := range rule.PluginSID {
			if id == 1337003 {
				foundSID = true
				break
			}
		}

		if !foundSID {
			t.Errorf("expected plugin rules to contain sid %d", 1337003)
		}

		if rule.PluginID != 1337 {
			t.Errorf("expected rule plugin ID to be %d but got %d", 1337, rule.PluginID)
		}
	}

	if dir.Kingdom != kingdom {
		t.Errorf("expected third directive kingdom to be '%s' but got '%s'", kingdom, dir.Kingdom)
	}

}

func TestDuplicatePluginTitle(t *testing.T) {
	log.Setup(true)
	const (
		kingdom       = "DEFAULT"
		titleTemplate = "EVENT_TITLE (SRC_IP to DST_IP)"
		priority      = 3
		reliability   = 1
		dirnumber     = 100000
	)

	in := strings.NewReader(`plugin	id	sid	title	category
test-x	1337	1337001	Test Plugin	Test Category
test-x	1337	1337002	Test Plugin	Test Category
test-x	1337	1337003	Test Plugin X	Test Category X`)

	var dirs siem.Directives
	var err error
	dirs, err = createDirective(in, dirs, kingdom, titleTemplate, priority, reliability, dirnumber)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(dirs.Dirs) != 2 {
		t.Fatalf("expected 2 directive created, but got %d", len(dirs.Dirs))
	}

	dir := dirs.Dirs[0]
	if dir.Name != "Test Plugin (SRC_IP to DST_IP)" {
		t.Errorf("expected directive name to be 'Test Plugin (SRC_IP to DST_IP)' but got '%s'", dir.Name)
	}

	for _, rule := range dir.Rules {
		var found1, found2 bool
		for _, id := range rule.PluginSID {
			if id == 1337001 {
				found1 = true
			}

			if id == 1337002 {
				found2 = true
			}
		}

		if !found1 {
			t.Error("expected plugin sid 1337001 to be found in rule plugin sid list")
		}

		if !found2 {
			t.Error("expected plugin sid 1337002 to be found in rule plugin sid list")
		}
	}

	dir = dirs.Dirs[1]
	if dir.Name != "Test Plugin X (SRC_IP to DST_IP)" {
		t.Errorf("expected directive name to be 'Test Plugin X (SRC_IP to DST_IP)' but got '%s'", dir.Name)
	}

	for _, rule := range dir.Rules {
		var found1 bool
		for _, id := range rule.PluginSID {
			if id == 1337003 {
				found1 = true
			}
		}

		if !found1 {
			t.Error("expected plugin sid 1337001 to be found in rule plugin sid list")
		}
	}
}

func TestMergeUnique(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}

	s3 := mergeUniqueSort(s1, s2)
	expected := []int{1, 2, 3, 4}

	if !reflect.DeepEqual(s3, expected) {
		t.Errorf("expected result to equal the expected, got %#v", s3)
	}
}
