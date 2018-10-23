package xcorrelator

import (
	"context"
	"dsiem/internal/pkg/shared/apm"
	"dsiem/internal/pkg/shared/ip"
	"dsiem/internal/pkg/shared/test"

	"dsiem/pkg/vuln"
	"os"
	"path"
	"reflect"
	"testing"
)

type vulnTests struct {
	ip            string
	port          int
	expectedFound bool
	expectedRes   []vuln.Result
}

var tblVuln = []vulnTests{
	{"10.0.0.1", 80, false, nil},
	{"not-an-ip", 80, false, nil},
	{"10.0.0.2", 80, true, []vuln.Result{{"Dummy", "10.0.0.2", "Detected in DB"}}},
	{"10.0.0.2", 80, true, []vuln.Result{{"Dummy", "10.0.0.2", "Detected in DB"}}},
}

type DummyV struct{}

func (d DummyV) Initialize(b []byte) (err error) {
	return
}

func (d DummyV) CheckIPPort(ctx context.Context, ipstr string, port int) (found bool, results []vuln.Result, err error) {
	_, err = ip.IsPrivateIP(ipstr)
	if err != nil {
		return
	}
	for _, v := range tblVuln {
		if ipstr == v.ip && port == v.port {
			return v.expectedFound, v.expectedRes, nil
		}
	}
	return
}

func TestVuln(t *testing.T) {
	_, err := test.DirEnv()
	if err != nil {
		t.Fatal(err)
	}

	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	apm.Enable(true)
	vuln.RegisterExtension(new(DummyV), "DummyV")

	vulnFileGlob = "vuln_dummy.json"
	confDir := path.Join(d, "fixtures")
	if err = InitVuln(confDir, 0); err != nil {
		t.Fatal("Cannot init vuln")
	}

	for _, tt := range tblVuln {
		_, _ = CheckVulnIPPort(tt.ip, tt.port)
		found, res := CheckVulnIPPort(tt.ip, tt.port)
		if found != tt.expectedFound {
			t.Errorf("Vuln: %v %v, expected found %v, actual %v", tt.ip, tt.port, tt.expectedFound, found)
		}
		if !reflect.DeepEqual(res, tt.expectedRes) {
			t.Errorf("Vuln: %v %v, expected result %v, actual %v", tt.ip, tt.port, tt.expectedRes, res)
		}
	}

	//	CheckIntelIP()

}
