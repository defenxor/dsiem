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

package nesd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	nesdsrv "github.com/defenxor/dsiem/internal/pkg/nesd"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

type vulnSource struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Plugin  string `json:"plugin"`
	Config  string `json:"config"`
}

type vulnSources struct {
	VulnSources []vulnSource `json:"vuln_sources"`
}

func startNesd(d string, wait time.Duration) error {
	ch := make(chan error)
	go func() {
		defer close(ch)

		csvDir := path.Join(d, "fixtures")
		err := nesdsrv.InitCSV(csvDir)
		if err != nil {
			ch <- fmt.Errorf("cannot read Nessus CSV from %s, %s", csvDir, err)
			return
		}

		if err := nesdsrv.Start("127.0.0.1", 8081); err != nil {
			ch <- fmt.Errorf("cannot start server, %s", err)
			return
		}
	}()

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case err := <-ch:
		return err
	}
}

func TestNesd(t *testing.T) {

	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	log.Setup(true)

	if err := startNesd(d, time.Second); err != nil {
		t.Fatal(err)
	}

	cfg := path.Join(d, "fixtures", "vuln_nessus.json")
	var vs vulnSources
	file, err := os.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)
	err = json.Unmarshal(byteValue, &vs)
	if err != nil {
		t.Fatal(err)
	}

	n := Nesd{}
	if err = n.Initialize([]byte(vs.VulnSources[0].Config)); err != nil {
		t.Fatal(err)
	}

	found, _, err := n.CheckIPPort(context.Background(), "10.0.0.1", 80)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("Expected to not find a match")
	}

	found, _, err = n.CheckIPPort(context.Background(), "192.168.225.196", 80)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("Expected to find a match")
	}

	found, _, err = n.CheckIPPort(context.Background(), "192.168.225.196", 22)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("Expected to not find a match (only Low risks found)")
	}

	// for error in config

	n = Nesd{}
	if err = n.Initialize([]byte(vs.VulnSources[1].Config)); err != nil {
		t.Fatal(err)
	}
	found, _, err = n.CheckIPPort(context.Background(), "10.0.0.1", 80)
	if err == nil {
		t.Fatal("expected to error due to mistake in config")
	}

	n = Nesd{}
	if err = n.Initialize([]byte(vs.VulnSources[2].Config)); err != nil {
		t.Fatal(err)
	}
	found, _, err = n.CheckIPPort(context.Background(), "10.0.0.1", 80)
	if err == nil {
		t.Fatal("expected to error due to mistake in config")
	}

}
