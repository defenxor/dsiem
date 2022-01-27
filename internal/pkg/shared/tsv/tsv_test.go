package tsv

import (
	"bytes"
	"fmt"
	"testing"
)

type subject struct {
	Name      string `json:"name" csv:"name"`
	ID        int    `json:"id" csv:"id"`
	SID       int    `json:"sid" csv:"sid"`
	SIDTitle  string `json:"title" csv:"title"`
	Category  string `json:"category" csv:"category"`
	Kingdom   string `json:"kingdom,omitempty" csv:"kingdom,omitempty"`
	lastIndex int
}

func (p *subject) Defaults(in interface{}) {
	v, ok := in.(subject)
	if !ok {
		return
	}

	if p.Name == "" {
		p.Name = v.Name
	}

	if p.ID == 0 {
		p.ID = v.ID
	}

	if p.SID == 0 {
		p.SID = v.SID
	}

	if p.SIDTitle == "" {
		p.SIDTitle = v.SIDTitle
	}

	if p.Category == "" {
		p.Category = v.Category
	}

	if p.Kingdom == "" {
		p.Kingdom = v.Kingdom
	}
}

func (p *subject) Next(b Castable) bool {
	switch p.lastIndex {
	case 0:
		p.Name = b.String()
	case 1:
		p.ID = b.Int()
	case 2:
		p.SID = b.Int()
	case 3:
		p.SIDTitle = b.String()
	case 4:
		p.Category = b.String()
	case 5:
		p.Kingdom = b.String()
	default:
		return false
	}

	p.lastIndex++
	return true
}

func (p subject) compare(v subject) error {
	if p.Name != v.Name {
		return fmt.Errorf("different name found, expected '%s' but got '%s'", v.Name, p.Name)
	}

	if p.ID != v.ID {
		return fmt.Errorf("different ID found, expected %d but got %d", v.ID, p.ID)
	}

	if p.SID != v.SID {
		return fmt.Errorf("different SID found, expected %d but got %d", v.SID, p.SID)
	}

	if p.SIDTitle != v.SIDTitle {
		return fmt.Errorf("different SIDTitle found, expected '%s' but got '%s'", v.SIDTitle, p.SIDTitle)
	}

	if p.Category != v.Category {
		return fmt.Errorf("different Category found, expected '%s' but got '%s'", v.Category, p.Category)
	}

	if p.Kingdom != v.Kingdom {
		return fmt.Errorf("different Kingdom found, expected '%s' but got '%s'", v.Kingdom, p.Kingdom)
	}

	return nil
}

var dataWithHeader = []byte(`name	id	sid	title	category	kingdom
test	1337	13370001	Some random plugin Name	Test Category
test	1337	13370002	Another random plugin name	Test Category	Test Kingdom
test	1337	13370003	Foo Bar Qux	Test Category	Test Kingdom
test	1337	13370004	Foo Bar Baz	Test Category	Test Kingdom	Test Overflow 1 BOO
`)

var dataWithoutHeader = []byte(`test	1337	13370001	Some random plugin Name	Test Category
test	1337	13370002	Another random plugin name	Test Category	Test Kingdom
test	1337	13370003	Foo Bar Qux	Test Category	Test Kingdom
test	1337	13370004	Foo Bar Baz	Test Category	Test Kingdom	Test Overflow 1 BOO
`)

func TestParserWithHeader(t *testing.T) {
	b := bytes.NewReader(dataWithHeader)

	defaultSubject := subject{
		Kingdom: "DEFAULT",
	}

	subjects := []subject{}
	parser := NewParser(b)
	for {
		var s subject
		ok := parser.Read(&s, defaultSubject)
		if !ok {
			break
		}

		subjects = append(subjects, s)
	}

	expected := []subject{
		{
			Name:     "test",
			ID:       1337,
			SID:      13370001,
			SIDTitle: "Some random plugin Name",
			Category: "Test Category",
			Kingdom:  "DEFAULT",
		},
		{
			Name:     "test",
			ID:       1337,
			SID:      13370002,
			SIDTitle: "Another random plugin name",
			Category: "Test Category",
			Kingdom:  "Test Kingdom",
		},
		{
			Name:     "test",
			ID:       1337,
			SID:      13370003,
			SIDTitle: "Foo Bar Qux",
			Category: "Test Category",
			Kingdom:  "Test Kingdom",
		},
		{
			Name:     "test",
			ID:       1337,
			SID:      13370004,
			SIDTitle: "Foo Bar Baz",
			Category: "Test Category",
			Kingdom:  "Test Kingdom",
		},
	}

	if len(subjects) != len(expected) {
		t.Fatalf("expected %d results, but got %d", len(expected), len(subjects))
	}

	for idx, plugin := range subjects {
		if err := plugin.compare(expected[idx]); err != nil {
			t.Error(err.Error())
		}
	}
}

func TestParserWithoutHeader(t *testing.T) {
	b := bytes.NewReader(dataWithoutHeader)

	defaultSubject := subject{
		Kingdom: "DEFAULT",
	}

	subjects := []subject{}
	parser := NewParser(b)
	parser.NoHeader = true

	for {
		var s subject
		ok := parser.Read(&s, defaultSubject)
		if !ok {
			break
		}

		subjects = append(subjects, s)
	}

	expected := []subject{
		{
			Name:     "test",
			ID:       1337,
			SID:      13370001,
			SIDTitle: "Some random plugin Name",
			Category: "Test Category",
			Kingdom:  "DEFAULT",
		},
		{
			Name:     "test",
			ID:       1337,
			SID:      13370002,
			SIDTitle: "Another random plugin name",
			Category: "Test Category",
			Kingdom:  "Test Kingdom",
		},
		{
			Name:     "test",
			ID:       1337,
			SID:      13370003,
			SIDTitle: "Foo Bar Qux",
			Category: "Test Category",
			Kingdom:  "Test Kingdom",
		},
		{
			Name:     "test",
			ID:       1337,
			SID:      13370004,
			SIDTitle: "Foo Bar Baz",
			Category: "Test Category",
			Kingdom:  "Test Kingdom",
		},
	}

	if len(subjects) != len(expected) {
		t.Fatalf("expected %d results, but got %d", len(expected), len(subjects))
	}

	for idx, plugin := range subjects {
		if err := plugin.compare(expected[idx]); err != nil {
			t.Error(err.Error())
		}
	}
}
