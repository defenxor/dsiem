package dpluger

import (
	"strings"
	"testing"
)

const (
	base = "testdata"
)

func TestTSVUpsert(t *testing.T) {

	for _, c := range []struct {
		testName       string
		sidList        string
		name           string
		sid            int
		id             int
		title          string
		category       string
		shouldIncrease bool
		count          int
	}{
		{
			testName: "adding new plugin",
			name:     "test",
			sidList: `plugin	id	sid	title	category	kingdom
test	1337	1	test-plugin-1	test-plugin
test	1337	2	test-plugin-2	test-plugin
			`,
			sid:            3,
			id:             1337,
			title:          "test-plugin-3",
			category:       "test-plugin",
			shouldIncrease: true,
			count:          3,
		},
		{
			testName: "readding existing plugin",
			name:     "test",
			sidList: `plugin	id	sid	title	category	kingdom
test	1337	1	test-plugin-1	test-plugin
test	1337	2	test-plugin-2	test-plugin
			`,
			sid:            1,
			id:             1337,
			title:          "test-plugin-1",
			category:       "test-plugin",
			shouldIncrease: true,
			count:          2,
		},
		{
			testName: "readding existing plugin but with different title",
			name:     "test",
			sidList: `plugin	id	sid	title	category	kingdom
test	1337	1	test-plugin-1	test-plugin
test	1337	2	test-plugin-2	test-plugin
			`,
			sid:            1,
			id:             1337,
			title:          "test-plugin-1-x",
			category:       "test-plugin",
			shouldIncrease: true,
			count:          3,
		},
		{
			testName: "readding existing plugin but with different but existing SID number",
			name:     "test",
			sidList: `plugin	id	sid	title	category	kingdom
test	1337	1	test-plugin-1	test-plugin
test	1337	2	test-plugin-2	test-plugin
			`,
			sid:            2,
			id:             1337,
			title:          "test-plugin-1",
			category:       "test-plugin",
			shouldIncrease: false,
			count:          2,
		},
		{
			testName: "readding existing plugin but with different but not-exist SID number",
			name:     "test",
			sidList: `plugin	id	sid	title	category	kingdom
test	1337	1	test-plugin-1	test-plugin
test	1337	2	test-plugin-2	test-plugin
			`,
			sid:            3,
			id:             1337,
			title:          "test-plugin-1",
			category:       "test-plugin",
			shouldIncrease: false,
			count:          2,
		},
	} {
		t.Run(c.testName, func(t *testing.T) {
			r := strings.NewReader(c.sidList)

			var ref tsvRef
			ref.initWithReader(c.name, base, r)

			shouldIncrease := ref.upsert(c.name, c.id, &c.sid, c.category, c.title)
			if shouldIncrease != c.shouldIncrease {
				t.Errorf("expected should-increase to be %t, got %t", c.shouldIncrease, shouldIncrease)
			}

			if c.count != ref.count() {
				t.Errorf("expected %d plugin(s), got %d", c.count, ref.count())
			}
		})
	}

}

func TestGroupByCustomData(t *testing.T) {
	ref := tsvRef{
		SIDs: map[int]PluginSID{
			1: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      1,
				SIDTitle: "Multiple viruses detected",
				Category: "wazuh alert level 12",
				Kingdom:  "virus",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "image",
					CustomData1:  "data.win.eventdata.image",
					CustomLabel2: "channel",
					CustomData2:  "data.win.system.channel",
				},
			},
			2: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      2,
				SIDTitle: "Clamd warning",
				Category: "wazuh alert level 7",
				Kingdom:  "virus",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "image",
					CustomData1:  "data.win.eventdata.image",
					CustomLabel2: "channel",
					CustomData2:  "data.win.system.channel",
				},
			},
			3: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      3,
				SIDTitle: "Clamd foo",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "image",
					CustomData1:  "data.win.eventdata.image",
					CustomLabel2: "channel",
					CustomData2:  "data.win.system.channel",
				},
			},
			4: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      4,
				SIDTitle: "Clamd bar",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "image",
					CustomData1:  "data.win.eventdata.image",
					CustomLabel2: "channel",
				},
			},
			5: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      5,
				SIDTitle: "Clamd none",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "image",
					CustomData1:  "data.win.eventdata.image",
					CustomLabel2: "channel",
				},
			},
			6: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      6,
				SIDTitle: "Clamd a",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "image",
					CustomData1:  "data.bar",
					CustomLabel2: "channel",
				},
			},
			7: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      7,
				SIDTitle: "Clamd b",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "foo",
					CustomData1:  "data.bar",
					CustomLabel2: "channel",
				},
			},
			8: {
				Name:     "wazuh-virus",
				ID:       1002,
				SID:      8,
				SIDTitle: "Clamd c",
				Category: "Disabling Security Toolss",
				Kingdom:  "Defense Evasion",
				CustomDataSet: CustomDataSet{
					CustomLabel1: "foo",
					CustomData1:  "data.bar",
					CustomLabel2: "channel",
				},
			},
		},
	}

	m := ref.GroupByCustomData()

	if len(m) != 4 {
		t.Errorf("expected %d group(s), got %d", 3, len(m))
	}
}
