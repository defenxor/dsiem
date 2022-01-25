package siem

import (
	"strings"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

var sampleDirective = &Directive{
	ID:       1337,
	Name:     "test",
	Kingdom:  "test",
	Category: "test",
}

func TestValidate(t *testing.T) {
	if !log.TestMode {
		t.Logf("Enabling log test mode")
		log.EnableTestingMode()
	}

	t.Run("reference on first rule", func(t *testing.T) {
		sample := *sampleDirective
		sample.Rules = []rule.DirectiveRule{
			{
				Name:      "test-rule-1",
				Stage:     1,
				From:      ":1",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
			{
				Name:      "test-rule-2",
				Stage:     2,
				From:      ":1",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
		}

		err := ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrReferenceOnFirstRule.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrReferenceOnFirstRule.Error(), err.Error())
		}

		sample = *sampleDirective
		sample.Rules = []rule.DirectiveRule{
			{
				Name:      "test-rule-1",
				Stage:     1,
				From:      "ANY",
				PortFrom:  ":1",
				To:        "ANY",
				PortTo:    "ANY",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
			{
				Name:      "test-rule-2",
				Stage:     2,
				From:      ":1",
				PortFrom:  ":1",
				To:        ":1",
				PortTo:    ":1",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
		}

		err = ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrReferenceOnFirstRule.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrReferenceOnFirstRule.Error(), err.Error())
		}
	})

	t.Run("negative reference number", func(t *testing.T) {
		sample := *sampleDirective
		sample.Rules = []rule.DirectiveRule{
			{
				Name:      "test-rule-1",
				Stage:     1,
				From:      "ANY",
				To:        "ANY",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
			{
				Name:      "test-rule-2",
				Stage:     2,
				From:      ":-1",
				To:        ":-1",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
		}

		err := ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrInvalidReference.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrInvalidReference.Error(), err.Error())
		}
	})

	t.Run("non-number reference", func(t *testing.T) {
		sample := *sampleDirective
		sample.Rules = []rule.DirectiveRule{
			{
				Name:      "test-rule-1",
				Stage:     1,
				From:      "ANY",
				To:        "ANY",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
			{
				Name:      "test-rule-2",
				Stage:     2,
				From:      ":foo",
				To:        ":bar",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
		}

		err := ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrInvalidReference.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrInvalidReference.Error(), err.Error())
		}
	})

	t.Run("reference to non-exist rule", func(t *testing.T) {
		sample := *sampleDirective
		sample.Rules = []rule.DirectiveRule{
			{
				Name:      "test-rule-1",
				Stage:     1,
				From:      "ANY",
				To:        "ANY",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
			{
				Name:      "test-rule-2",
				Stage:     2,
				From:      ":3",
				To:        ":4",
				Type:      "PluginRule",
				PluginID:  1,
				PluginSID: []int{1},
			},
		}

		err := ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrInvalidReference.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrInvalidReference.Error(), err.Error())
		}
	})

	t.Run("directive with no name", func(t *testing.T) {
		sample := *sampleDirective
		sample.Name = ""

		err := ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrNoDirectiveName.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrNoDirectiveName.Error(), err.Error())
		}
	})

	t.Run("directive with no kingdom", func(t *testing.T) {
		sample := *sampleDirective
		sample.Kingdom = ""

		err := ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrNoDirectiveKingdom.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrNoDirectiveKingdom.Error(), err.Error())
		}
	})

	t.Run("directive with no category", func(t *testing.T) {
		sample := *sampleDirective
		sample.Category = ""

		err := ValidateDirective(&sample, &Directives{Dirs: []Directive{}})
		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), ErrNoDirectiveCategory.Error()) {
			t.Errorf("expected error to contain '%s' but got '%s'", ErrNoDirectiveCategory.Error(), err.Error())
		}
	})
}

func TestIsReference(t *testing.T) {
	tc := []struct {
		str      string
		expected bool
	}{
		{":1", true},
		{":2", true},
		{":3", true},
		{":10", true},
		{":99", true},
		{":01", false},
		{":a", false},
		{":x", false},
		{":-1", false},
		{":999", false},
	}

	for _, c := range tc {
		result := validateReference(c.str, 999) == nil
		if result != c.expected {
			t.Errorf("expected reference check of '%s' to be %t but got %t", c.str, c.expected, result)
		}
	}
}
