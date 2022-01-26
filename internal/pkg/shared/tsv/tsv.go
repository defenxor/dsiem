package tsv

import (
	"io"

	"github.com/valyala/tsvreader"
)

type Castable interface {
	// String cast the interface value into string
	String() string

	// Int cast the interface value into int
	Int() int
}

type Parsable interface {
	// Defaults set all Parsable empty value to value according to the given interface.
	Defaults(interface{})

	// Next set next Parsable field value to the Castable.
	Next(Castable) bool
}

type Parser struct {
	first  bool
	reader *tsvreader.Reader

	// NoHeader mark the data as no-header TSV.
	NoHeader bool
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: tsvreader.New(r),
		first:  true,
	}
}

func (p *Parser) Read(v Parsable, def interface{}) bool {
	if p.first && !p.NoHeader {
		if !p.reader.Next() {
			return false
		}

		for p.reader.HasCols() {
			p.reader.SkipCol()
		}

		p.first = false
	}

	hasNext := p.reader.Next()
	for p.reader.HasCols() {
		if !v.Next(p.reader) {
			p.reader.SkipCol()
		}
	}

	v.Defaults(def)
	return hasNext
}
