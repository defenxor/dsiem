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
