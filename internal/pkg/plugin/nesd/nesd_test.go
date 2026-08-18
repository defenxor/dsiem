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
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/defenxor/dsiem/pkg/vuln"
)

// ensure the Nesd fits the vuln.Checker interface
var _ vuln.Checker = &Nesd{}

type vulnSource struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Plugin  string `json:"plugin"`
	Config  string `json:"config"`
}

func prepareTestNesd(rt *dummyRoundTripper) *Nesd {
	return &Nesd{
		Client: &http.Client{
			Transport: rt,
		},
	}
}

type dummyRoundTripper struct {
	nextResponse *http.Response
	nextError    error
}

func (d *dummyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return d.nextResponse, d.nextError
}

func (d *dummyRoundTripper) setNextResponse(status int, b []byte) {
	d.nextError = nil
	d.nextResponse = &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

func (d *dummyRoundTripper) setNextError(err error) {
	d.nextResponse = nil
	d.nextError = err
}

func TestNesd(t *testing.T) {
	for _, tt := range []struct {
		description   string
		vuln          vulnSource
		httpResponse  []byte
		httpError     error
		httpStatus    int
		expectedFound bool
		expectedError bool
	}{
		{
			description: "match 10.0.0.1:22 - Not Found",
			vuln: vulnSource{
				Name:    "Nessus",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"http://localhost:8081/?ip=${ip}&port=${port}\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  []byte("no vulnerability found\n"),
			expectedFound: false,
		},
		{
			description: "match 192.168.225.196:80 - Not Found (None Risk)",
			vuln: vulnSource{
				Name:    "Nessus",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"http://localhost:8081/?ip=${ip}&port=${port}\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  []byte(`[{"cve": "CVE-1999-0524", "name": "ICMP Timestamp Request Remote Date Disclosure", "risk": "None"}]`),
			expectedFound: false,
		},
		{
			description: "match 192.168.225.196:80 - Not Found (Low Risk)",
			vuln: vulnSource{
				Name:    "Nessus",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"http://localhost:8081/?ip=${ip}&port=${port}\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  []byte(`[{"cve": "CVE-1999-0524", "name": "ICMP Timestamp Request Remote Date Disclosure", "risk": "Low"}]`),
			expectedFound: false,
		},
		{
			description: "match 192.168.225.196:22 - Found High Risk",
			vuln: vulnSource{
				Name:    "Nessus",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"http://localhost:8081/?ip=${ip}&port=${port}\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  []byte(`[{"cve": "CVE-1999-0524", "name": "ICMP Timestamp Request Remote Date Disclosure", "risk": "High"}]`),
			expectedFound: true,
		},
		{
			description: "config error - missing URL",
			vuln: vulnSource{
				Name:    "Nessus-ERror",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  nil,
			httpError:     errors.New("test"),
			expectedFound: false,
			expectedError: true,
		},
		{
			description: "config error - missing IP Port",
			vuln: vulnSource{
				Name:    "Nessus-ERror",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"http://localhost:8081/?\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  nil,
			httpError:     errors.New("test"),
			expectedFound: false,
			expectedError: true,
		},
		{
			description: "invalid response",
			vuln: vulnSource{
				Name:    "Nessus",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"http://localhost:8081/?ip=${ip}&port=${port}\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  []byte(`{"foo":"bar"}`),
			expectedFound: false,
			expectedError: true,
		},
		{
			description: "non-JSON response",
			vuln: vulnSource{
				Name:    "Nessus",
				Type:    "Ip-Port",
				Plugin:  "Nesd",
				Enabled: true,
				Config:  "{ \"url\" : \"http://localhost:8081/?ip=${ip}&port=${port}\" }",
			},
			httpStatus:    http.StatusOK,
			httpResponse:  []byte(``),
			expectedFound: false,
			expectedError: true,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			rt := &dummyRoundTripper{}

			n := prepareTestNesd(rt)
			if err := n.Initialize([]byte(tt.vuln.Config)); err != nil {
				t.Fatal(err)
			}

			if tt.httpError != nil {
				rt.setNextError(tt.httpError)
			} else {
				rt.setNextResponse(tt.httpStatus, tt.httpResponse)
			}
			found, _, err := n.CheckIPPort(context.Background(), "10.0.0.1", 80)
			if tt.expectedError {
				if err == nil {
					t.Fatal("expected to error due to mistake in config")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}

				if found != tt.expectedFound {
					t.Fatalf("expecting found %t, got %t", tt.expectedFound, found)
				}
			}

		})
	}
}
