// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package asset

import (
	"path"
	"reflect"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/shared/test"
)

func TestInit(t *testing.T) {

	d, err := test.DirEnv(false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Using base dir %s", d)
	fDir := path.Join(d, "internal", "pkg", "dsiem", "asset", "fixtures")
	err = Init(path.Join(fDir, "asset2"))
	if err == nil {
		t.Fatal(err)
	}
	assets = NetworkAssets{}
	err = Init(path.Join(fDir, "asset3"))
	if err == nil {
		t.Fatal(err)
	}
	assets = NetworkAssets{}
	err = Init(path.Join(fDir, "asset4"))
	if err == nil {
		t.Fatal(err)
	}
	assets = NetworkAssets{}
	err = Init(path.Join(fDir, "assetX"))
	if err == nil {
		t.Fatal(err)
	}
	assets = NetworkAssets{}
	err = Init(path.Join(fDir, "asset1"))
	if err != nil {
		t.Fatal(err)
	}
	whitelisted, err := IsWhiteListed("192.168.0.2")
	if err != nil {
		t.Fatal(err)
	}
	if !whitelisted {
		t.Fatal("Expected 192.168.0.2 to be in whitelist")
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
		t.Errorf("Cannot find name for %s", privNet)
	}
	if GetName(privIP) != "firewall" {
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

	privIP2 := "2002:c0a8:0001:0:0:0:0:1"
	privNet2 := "2002:c0a8:1::/64"
	privNetName2 := "2002:c0a8:1::/64-Net"
	if ok, err := IsInHomeNet(privIP2); !ok {
		t.Error(err)
	}
	if GetName(privNet2) == privNetName2 {
		t.Errorf("Cannot find name for %s", privNet2)
	}
	if GetName(privIP2) != "firewall-ipv6" {
		t.Errorf("Cannot find name for %s", privIP2)
	}
	if GetValue(privIP2) != 5 {
		t.Errorf("Cannot get correct asset value for %s", privIP2)
	}
	net2 := GetAssetNetworks(privIP2)
	expected2 := []string{"2002:c0a8:1::/64"}
	if !reflect.DeepEqual(net2, expected2) {
		t.Errorf("expected %v, obtained %v", expected2, net2)
	}

}
