package asset

import (
	"path"
	"reflect"
	"testing"

	"github.com/dsiem/internal/pkg/shared/test"
)

func TestInit(t *testing.T) {

	d, err := test.DirEnv()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Using base dir %s", d)
	fDir := path.Join(d, "internal", "pkg", "dsiem", "asset", "fixtures")
	err = Init(path.Join(fDir, "asset2"))
	if err == nil {
		t.Fatal(err)
	}
	assets = networkAssets{}
	err = Init(path.Join(fDir, "asset3"))
	if err == nil {
		t.Fatal(err)
	}
	assets = networkAssets{}
	err = Init(path.Join(fDir, "asset4"))
	if err == nil {
		t.Fatal(err)
	}
	assets = networkAssets{}
	err = Init(path.Join(fDir, "assetX"))
	if err == nil {
		t.Fatal(err)
	}
	assets = networkAssets{}
	err = Init(path.Join(fDir, "asset1"))
	if err != nil {
		t.Fatal(err)
	}
}
func TestAsset(t *testing.T) {

	privIP := "192.168.0.1"
	privNet := "192.168.0.0/16"
	privNetName := "192-168-Net"
	pubIP := "8.8.8.200"
	if ok, err := IsInHomeNet(privIP); !ok {
		t.Error(err)
	}
	if ok, err := IsInHomeNet(pubIP); ok {
		t.Error(err)
	}
	if GetName(privNet) == privNetName {
		t.Errorf("Cannot find name for %s", privIP)
	}
	if GetName(privIP) == "Firewall" {
		t.Errorf("Cannot find name for %s", privIP)
	}
	if GetValue(privIP) != 5 {
		t.Errorf("Cannot get correct asset value for %s", privIP)
	}
	if GetValue(pubIP) != 0 {
		t.Errorf("Cannot get correct asset value for %s", privIP)
	}
	net := GetAssetNetworks(privIP)
	expected := []string{"192.168.0.0/16"}
	if !reflect.DeepEqual(net, expected) {
		t.Errorf("expected %v, obtained %v", expected, net)
	}
	net = GetAssetNetworks(pubIP)
	expected = []string{}
	if !reflect.DeepEqual(net, expected) {
		t.Errorf("expected %v, obtained %v", expected, net)
	}
}
