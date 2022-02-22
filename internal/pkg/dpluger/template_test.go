package dpluger

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/template"
)

func TestGroupByCustomDataTemplate(t *testing.T) {
	expectedOutput := `
  if [plugin_sid] in ["1", "2", "3"] {
    mutate { 
      "custom_label1" => "image"
      "custom_data1" => "%{[data][win][eventdata][image]}"
      "custom_label2" => "channel"
      "custom_data2" => "%{[data][win][system][channel]}"
    }
  }
  if [plugin_sid] in ["4", "5"] {
    mutate { 
      "custom_label1" => "image"
      "custom_data1" => "%{[data][win][eventdata][image]}"
    }
  }
  if [plugin_sid] in ["6"] {
    mutate { 
      "custom_label1" => "image"
      "custom_data1" => "%{[data][bar]}"
    }
  }
  if [plugin_sid] in ["7", "8"] {
    mutate { 
      "custom_label1" => "foo"
      "custom_data1" => "%{[data][bar]}"
    }
  }
`
	ref := []PluginSIDWithCustomData{
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      1,
				SIDTitle: "Multiple viruses detected",
				Category: "wazuh alert level 12",
				Kingdom:  "virus",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "image",
				CustomData1:  "data.win.eventdata.image",
				CustomLabel2: "channel",
				CustomData2:  "data.win.system.channel",
			},
		},
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      2,
				SIDTitle: "Clamd warning",
				Category: "wazuh alert level 7",
				Kingdom:  "virus",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "image",
				CustomData1:  "data.win.eventdata.image",
				CustomLabel2: "channel",
				CustomData2:  "data.win.system.channel",
			},
		},
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      3,
				SIDTitle: "Clamd foo",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "image",
				CustomData1:  "data.win.eventdata.image",
				CustomLabel2: "channel",
				CustomData2:  "data.win.system.channel",
			},
		},
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      4,
				SIDTitle: "Clamd bar",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "image",
				CustomData1:  "data.win.eventdata.image",
				CustomLabel2: "channel",
			},
		},
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      5,
				SIDTitle: "Clamd none",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "image",
				CustomData1:  "data.win.eventdata.image",
				CustomLabel2: "channel",
			},
		},
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      6,
				SIDTitle: "Clamd a",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "image",
				CustomData1:  "data.bar",
				CustomLabel2: "channel",
			},
		},
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      7,
				SIDTitle: "Clamd b",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "foo",
				CustomData1:  "data.bar",
				CustomLabel2: "channel",
			},
		},
		{
			PluginSID: PluginSID{
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      8,
				SIDTitle: "Clamd c",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
			},
			CustomDataSet: CustomDataSet{
				CustomLabel1: "foo",
				CustomData1:  "data.bar",
				CustomLabel2: "channel",
			},
		},
	}

	m := GroupByCustomData(ref)

	tpl, err := template.New("test").Funcs(templateFunctions).Parse(templGroupByCustomData)
	if err != nil {
		t.Fatal(err.Error())
	}

	buf := bytes.NewBuffer([]byte{})
	if err := tpl.Execute(buf, m); err != nil {
		t.Fatal(err.Error())
	}

	if errors := compareStringSet(buf.String(), expectedOutput, "\n"); len(errors) != 0 {
		for _, err := range errors {
			t.Error(err.Error())
		}
	}
}

func compareStringSet(s1, s2, sep string) []error {
	set1, set2 := strings.Split(s1, sep), strings.Split(s2, sep)
	if len(set1) != len(set2) {
		return []error{fmt.Errorf("expected %d line(s), got %d", len(set2), len(set1))}
	}

	errors := make([]error, 0)
	for idx, set := range set1 {
		if set != set2[idx] {
			errors = append(errors, fmt.Errorf("line %d not match, \nexpected\t '%s',\ngot\t\t\t '%s'", idx, set2[idx], set))
		}
	}

	return errors
}
