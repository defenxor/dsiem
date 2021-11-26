package dpluger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
)

func TestMerger(t *testing.T) {
	for _, c := range []struct {
		description    string
		existing       siem.Directives
		target         siem.Directives
		expected       siem.Directives
		promptAnswer   bool
		promptExpected bool
	}{
		{
			description: "two different directive set",
			existing: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			target: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			expected: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
		},
		{
			description: "two directive set, with one exactly same directive",
			existing: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       3,
						Name:     "directive-3",
						Priority: 3,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			target: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			expected: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       3,
						Name:     "directive-3",
						Priority: 3,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
		},
		{
			description:    "two directive set, with one conflicting directive, with merge",
			promptExpected: true,
			promptAnswer:   true,
			existing: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       3,
						Name:     "directive-3",
						Priority: 3,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			target: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1-foo",
						Priority: 1,
						Kingdom:  "test-foo",
						Category: "test-foo",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			expected: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1-foo",
						Priority: 1,
						Kingdom:  "test-foo",
						Category: "test-foo",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       3,
						Name:     "directive-3",
						Priority: 3,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
		},
		{
			description:    "two directive set, with one conflicting directive, with not-merge",
			promptExpected: true,
			promptAnswer:   false,
			existing: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       3,
						Name:     "directive-3",
						Priority: 3,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			target: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1-foo",
						Priority: 1,
						Kingdom:  "test-foo",
						Category: "test-foo",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
			expected: siem.Directives{
				Dirs: []siem.Directive{
					{
						ID:       1,
						Name:     "directive-1",
						Priority: 1,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       2,
						Name:     "directive-2",
						Priority: 2,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
					{
						ID:       3,
						Name:     "directive-3",
						Priority: 3,
						Kingdom:  "test",
						Category: "test",
						Rules:    []rule.DirectiveRule{},
					},
				},
			},
		},
	} {
		t.Run(c.description, func(t *testing.T) {
			cmd, reader, transport := &testCommand{}, &testFileReader{}, &testRoundTripper{}

			b1, err := json.Marshal(c.existing)
			if err != nil {
				t.Fatal(err.Error())
			}

			transport.setNextGetResponse(http.StatusOK, b1, nil)

			b2, err := json.Marshal(c.target)
			if err != nil {
				t.Fatal(err.Error())
			}

			reader.nextBytes = b2

			if c.promptExpected {
				cmd.nextBoolResponse = c.promptAnswer
			}

			cfg := MergeConfig{
				Host:       "localhost:9200",
				SourceJSON: "directive_test",
				TargetJSON: "./new_directive_test",
			}

			err = Merge(cmd, cfg, WithCustomFileReader(reader), WithCustomTransport(transport))

			if err != nil {
				t.Fatal(err.Error())
			}

			if c.promptExpected {
				lastPrompt := cmd.lastPrompt
				if lastPrompt == "" {
					t.Errorf("expected some prompt from the command")
				}
			}

			result := transport.PostResponse()

			var dirs siem.Directives
			if err := json.Unmarshal(result, &dirs); err != nil {
				t.Fatalf("invalid response directives, %s", err.Error())
			}

			if err := directivesEqual(dirs, c.expected); err != nil {
				t.Error(err.Error())
			}
		})
	}

}

type testCommand struct {
	nextBoolResponse bool
	lastPrompt       string
}

func (c *testCommand) PromptBool(msg string, def bool) bool {
	c.lastPrompt = msg
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
