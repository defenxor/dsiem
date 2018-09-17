package tsv

import (
	"encoding/csv"
	"errors"
	"golang.org/x/text/unicode/norm"
	"io"
	"reflect"
	"strconv"
)

// Parser has information for parser
type Parser struct {
	Headers    []string
	Reader     *csv.Reader
	Data       interface{}
	ref        reflect.Value
	indices    []int // indices is field index list of header array
	structMode bool
	normalize  norm.Form
}

// NewStructModeParser creates new TSV parser with given io.Reader as struct mode
func NewParser(reader io.Reader, data interface{}) (*Parser, error) {
	r := csv.NewReader(reader)
	r.Comma = '\t'

	// first line should be fields
	headers, err := r.Read()

	if err != nil {
		return nil, err
	}

	for i, header := range headers {
		headers[i] = header
	}

	p := &Parser{
		Reader:     r,
		Headers:    headers,
		Data:       data,
		ref:        reflect.ValueOf(data).Elem(),
		indices:    make([]int, len(headers)),
		structMode: false,
		normalize:  -1,
	}

	// get type information
	t := p.ref.Type()

	for i := 0; i < t.NumField(); i++ {
		// get TSV tag
		tsvtag := t.Field(i).Tag.Get("tsv")
		if tsvtag != "" {
			// find tsv position by header
			for j := 0; j < len(headers); j++ {
				if headers[j] == tsvtag {
					// indices are 1 start
					p.indices[j] = i + 1
					p.structMode = true
				}
			}
		}
	}

	if !p.structMode {
		for i := 0; i < len(headers); i++ {
			p.indices[i] = i + 1
		}
	}

	return p, nil
}

// NewParserWithoutHeader creates new TSV parser with given io.Reader
func NewParserWithoutHeader(reader io.Reader, data interface{}) *Parser {
	r := csv.NewReader(reader)
	r.Comma = '\t'

	p := &Parser{
		Reader:    r,
		Data:      data,
		ref:       reflect.ValueOf(data).Elem(),
		normalize: -1,
	}

	return p
}

// Next puts reader forward by a line
func (p *Parser) Next() (eof bool, err error) {

	// Get next record
	var records []string

	for {
		// read until valid record
		records, err = p.Reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				return true, nil
			}
			return false, err
		}
		if len(records) > 0 {
			break
		}
	}

	if len(p.indices) == 0 {
		p.indices = make([]int, len(records))
		// mapping simple index
		for i := 0; i < len(records); i++ {
			p.indices[i] = i + 1
		}
	}

	// record should be a pointer
	for i, record := range records {
		idx := p.indices[i]
		if idx == 0 {
			// skip empty index
			continue
		}
		// get target field
		field := p.ref.Field(idx - 1)
		switch field.Kind() {
		case reflect.String:
			// Normalize text
			if p.normalize >= 0 {
				record = p.normalize.String(record)
			}
			field.SetString(record)
		case reflect.Bool:
			if record == "" {
				field.SetBool(false)
			} else {
				col, err := strconv.ParseBool(record)
				if err != nil {
					return false, err
				}
				field.SetBool(col)
			}
		case reflect.Int:
			if record == "" {
				field.SetInt(0)
			} else {
				col, err := strconv.ParseInt(record, 10, 0)
				if err != nil {
					return false, err
				}
				field.SetInt(col)
			}
		default:
			return false, errors.New("Unsupported field type")
		}
	}

	return false, nil
}
