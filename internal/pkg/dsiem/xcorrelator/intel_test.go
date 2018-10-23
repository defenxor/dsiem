package xcorrelator

import (
	"context"
	"dsiem/internal/pkg/shared/apm"
	"dsiem/internal/pkg/shared/ip"
	"dsiem/internal/pkg/shared/test"
	"dsiem/pkg/intel"
	"os"
	"path"
	"reflect"
	"testing"
)

type intelTests struct {
	ip            string
	expectedFound bool
	expectedRes   []intel.Result
}

var tblIntel = []intelTests{
	{"10.0.0.1", false, nil},
	{"not-an-ip", false, nil},
	{"10.0.0.2", true, []intel.Result{{"Dummy", "10.0.0.2", "Detected in DB"}}},
	{"10.0.0.2", true, []intel.Result{{"Dummy", "10.0.0.2", "Detected in DB"}}},
}

type Dummy struct{}

func (d Dummy) Initialize(b []byte) (err error) {
	return
}

func (d Dummy) CheckIP(ctx context.Context, ipstr string) (found bool, results []intel.Result, err error) {
	_, err = ip.IsPrivateIP(ipstr)
	if err != nil {
		return
	}
	for _, v := range tblIntel {
		if ipstr == v.ip {
			return v.expectedFound, v.expectedRes, nil
		}
	}
	return
}

func TestIntel(t *testing.T) {
	_, err := test.DirEnv()
	if err != nil {
		t.Fatal(err)
	}

	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	apm.Enable(true)
	intel.RegisterExtension(new(Dummy), "Dummy")

	intelFileGlob = "intel_dummy.json"
	confDir := path.Join(d, "fixtures")
	if err = InitIntel(confDir, 0); err != nil {
		t.Fatal("Cannot init intel")
	}

	for _, tt := range tblIntel {
		_, _ = CheckIntelIP(tt.ip, 0)
		found, res := CheckIntelIP(tt.ip, 0)
		if found != tt.expectedFound {
			t.Errorf("Intel: %v, expected found %v, actual %v", tt.ip, tt.expectedFound, found)
		}
		if !reflect.DeepEqual(res, tt.expectedRes) {
			t.Errorf("Intel: %v, expected result %v, actual %v", tt.ip, tt.expectedRes, res)
		}
	}

	//	CheckIntelIP()

}
