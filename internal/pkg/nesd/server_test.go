// Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.
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
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

func TestServer(t *testing.T) {
	ctx := context.Background()
	log.Setup(true)
	err := Start("", 0)
	if err == nil {
		t.Fatal("expected error due to bad port")
	}

	err = Start("", 8085)
	if err == nil {
		t.Fatal("expected error due to bad address")
	}

	if !csvInitialized {
		dir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		csvDir := path.Join(dir, "fixtures", "example2")
		InitCSV(csvDir)
	}

	errs := make(chan error, 1)

	go func() {
		e := Start("127.0.0.1", 8085)
		if e.Error() == "http: Server closed" {
			e = nil
		}
		errs <- e
	}()

	time.Sleep(time.Second * 2)

	select {
	case res := <-errs:
		if res != nil {
			t.Fatal(res)
		}
	default:
	}

	defer httpSrv.Shutdown(ctx)

	url := "http://127.0.0.1:8085/?"
	httpTest(t, url, "GET", "", 400)
	httpTest(t, url+"ip=foo&port=bar", "GET", "", 418)
	httpTest(t, url+"ip=192.168.225.196&port=0", "GET", "", 418)
	httpTest(t, url+"ip=192.168.225.196&port=31337", "GET", "", 200)
	httpTest(t, url+"ip=192.168.225.196&port=80", "GET", "", 200)
	httpTest(t, url+"ip=192.168.225.196&port=161", "GET", "", 200)

}

func httpTest(t *testing.T, url, method, data string, expectedStatusCode int) {
	_, code, err := httpClient(url, method, data)
	if err != nil {
		t.Fatal("Error received from httpClient", url, ":", err)
	}
	if code != expectedStatusCode && expectedStatusCode != 500 {
		t.Fatal("Received", code, "from", url, "expected", expectedStatusCode)
	}
	if code != expectedStatusCode && expectedStatusCode == 500 {
		fmt.Println("Flaky server test result detected, for", url, "retrying for 3 times every sec ..")
		for i := 0; i < 10; i++ {
			fmt.Println("attempt ", i+1, "..")
			_, code, err := httpClient(url, method, data)
			if err != nil {
				t.Fatal("Flaky test workaround receive error from httpClient", url, ":", err)
			}
			if code == expectedStatusCode {
				return
			}
			time.Sleep(time.Second)
		}
		t.Fatal("Flaky test received", code, "from", url, "expected", expectedStatusCode)
	}
}

func httpClient(url, method, data string) (out string, statusCode int, err error) {
	client := &http.Client{}
	r := strings.NewReader(data)
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	out = string(body)
	statusCode = resp.StatusCode
	return
}
