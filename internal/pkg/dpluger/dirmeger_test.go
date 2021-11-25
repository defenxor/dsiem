package dpluger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
)

type testCommand struct {
	nextBoolResponse bool
}

func (c *testCommand) PromptBool(string) bool {
	return c.nextBoolResponse
}

func (c *testCommand) Log(msg string) {
	log.Println(msg)
}

func (c *testCommand) SetNextBoolResponse(res bool) {
	c.nextBoolResponse = res
}

type testRoundTripper struct {
	nextResponse *http.Response
	nextError    error
	postBody     []byte
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet {
		return t.getResponse()
	}

	if req.Method == http.MethodPost {
		defer req.Body.Close()
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		t.postBody = b
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     http.StatusText(http.StatusOK),
		}, nil
	}

	return nil, fmt.Errorf("unknown http method '%s'", req.Method)
}

func (t *testRoundTripper) getResponse() (*http.Response, error) {
	if t.nextError != nil {
		return nil, t.nextError
	}

	if t.nextResponse == nil {
		return nil, fmt.Errorf("no next response set")
	}

	return t.nextResponse, nil
}

func (t *testRoundTripper) setNextGetResponse(statuscode int, response []byte, err error) {
	if err != nil {
		t.nextError = err
		return
	}

	t.nextResponse = &http.Response{
		StatusCode: statuscode,
		Status:     http.StatusText(statuscode),
		Body:       io.NopCloser(bytes.NewReader(response)),
	}
}

func (t *testRoundTripper) PostResponse() []byte {
	return t.postBody
}

type testFileReader struct {
	nextBytes []byte
	nextError error
}

func (t *testFileReader) Read(string) ([]byte, error) {
	if t.nextError != nil {
		return nil, t.nextError
	}

	return t.nextBytes, nil
}

func TestMerger(t *testing.T) {
	for _, c := range []struct {
		description string
		dir1        siem.Directives
		dir2        siem.Directives
		expected    siem.Directives
	}{
		// TODO: add test cases
	} {
		t.Run(c.description, func(t *testing.T) {
			cmd, reader, transport := &testCommand{}, &testFileReader{}, &testRoundTripper{}

			b1, err := json.Marshal(c.dir1)
			if err != nil {
				t.Fatal(err.Error())
			}

			transport.setNextGetResponse(http.StatusOK, b1, nil)

			b2, err := json.Marshal(c.dir2)
			if err != nil {
				t.Fatal(err.Error())
			}

			reader.nextBytes = b2

			cfg := MergeConfig{
				// TODO: fill the config
			}

			err = Merge(cmd, cfg, WithCustomFileReader(reader), WithCustomTransport(transport))

			if err != nil {
				t.Fatal(err.Error())
			}

			result := transport.PostResponse()

			var dirs siem.Directives
			if err := json.Unmarshal(result, &dirs); err != nil {
				t.Fatalf("invalid response directives, %s", err.Error())
			}

			if err := compareDirectives(dirs, c.expected); err != nil {
				t.Error(err.Error())
			}
		})
	}

}

func compareDirectives(dir1, dir2 siem.Directives) error {
	panic("not implemented")
}
