package dpluger

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"
)

const multipleResultsResponse = `
{
	"test-2020.02.13": {
		"mappings": {}
	},
	"test-2020.02.14": {
	  "mappings": {
		"test": {
		  "full_name": "test",
		  "mapping": {
			"test": {
			  "type": "text",
			  "fields": {
				"keyword": {
				  "type": "keyword",
				  "ignore_above": 256
				}
			  }
			}
		  }
		}
	  }
	}
  }
`

const fieldExistResponse = `
{
	"test-2020.02.14": {
	  "mappings": {
		"test": {
		  "full_name": "test",
		  "mapping": {
			"test": {
			  "type": "text",
			  "fields": {
				"keyword": {
				  "type": "keyword",
				  "ignore_above": 256
				}
			  }
			}
		  }
		}
	  }
	}
  }
`

const fieldExistMultipleLevelResponse = `
{
	"test-2020.02.14": {
	  "mappings": {
		"foo.bar": {
		  "full_name": "foo.bar",
		  "mapping": {
			"bar": {
			  "type": "text",
			  "fields": {
				"keyword": {
				  "type": "keyword",
				  "ignore_above": 256
				}
			  }
			}
		  }
		}
	  }
	}
  }
`

const fieldNotExistResponse = `
{
	"test-2020.02.14": {
	  "mappings": {}
	}
}`

const fieldExistNoKeywordResponse = `
{
	"suricata-2020.02.14": {
	  "mappings": {
		"foo": {
		  "full_name": "foo",
		  "mapping": {
			"foo": {
			  "type": "date"
			}
		  }
		}
	  }
	}
  }`

type TestTransport struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (t *TestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTrip(req)
}

func TestParseGetField(t *testing.T) {
	for _, c := range []struct {
		name       string
		fieldType  string
		hasKeyword bool
		exist      bool
		response   string
		fieldName  string
	}{
		{
			name:       "single level",
			response:   fieldExistResponse,
			hasKeyword: true,
			exist:      true,
			fieldType:  "text",
			fieldName:  "test",
		},
		{
			name:       "multiple levels",
			response:   fieldExistMultipleLevelResponse,
			hasKeyword: true,
			exist:      true,
			fieldType:  "text",
			fieldName:  "foo.bar",
		},
		{
			name:       "not exist",
			response:   fieldNotExistResponse,
			hasKeyword: false,
			exist:      false,
			fieldName:  "foo.bar",
		},
		{
			name:       "exist but no keyword",
			response:   fieldExistNoKeywordResponse,
			fieldType:  "date",
			hasKeyword: false,
			exist:      true,
			fieldName:  "foo",
		},
		{
			name:       "mulitple returns",
			response:   multipleResultsResponse,
			fieldType:  "text",
			hasKeyword: true,
			exist:      true,
			fieldName:  "test",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			cl := &http.Client{
				Transport: &TestTransport{
					roundTrip: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(c.response)),
						}, nil
					},
				},
			}

			escl, err := elastic7.NewSimpleClient(
				elastic7.SetURL("http://localhost:9200"),
				elastic7.SetHttpClient(cl),
			)

			if err != nil {
				t.Fatal(err.Error())
			}

			ecl := &es7Client{
				client: escl,
			}

			ft, haskeyword, err := ecl.FieldType(context.Background(), "test", c.fieldName)
			if !c.exist {
				if err != ErrFieldMappingNotExist {
					t.Fatalf("expected error '%s', got '%s'", ErrFieldMappingNotExist.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error, %s", err.Error())
				}

				if haskeyword != c.hasKeyword {
					t.Errorf("expected haskeyword to be %t got %t", c.hasKeyword, haskeyword)
				}

				if ft != c.fieldType {
					t.Errorf("expected field type to be %s got %s", c.fieldType, ft)
				}
			}
		})
	}

}
