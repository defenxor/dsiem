package dpluger

import "testing"

func TestGroupByCustomData(t *testing.T) {
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

	if len(m) != 4 {
		t.Errorf("expected %d group(s), got %d", 3, len(m))
	}
}
