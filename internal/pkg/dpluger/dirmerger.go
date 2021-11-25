package dpluger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
)

type Commander interface {
	PromptBool(string) bool
	Log(string)
}

type FileReader interface {
	Read(string) ([]byte, error)
}

type MergeConfig struct {
	Host       string
	SourceJSON string
	TargetJSON string
}

type mergeOption struct {
	transport  http.RoundTripper
	fileReader FileReader
}

type MergeOptionFunc func(*mergeOption)

func WithCustomTransport(tr http.RoundTripper) MergeOptionFunc {
	return func(o *mergeOption) {
		o.transport = tr
	}
}

func WithCustomFileReader(fr FileReader) MergeOptionFunc {
	return func(o *mergeOption) {
		o.fileReader = fr
	}
}

func Merge(cmd Commander, cfg MergeConfig, options ...MergeOptionFunc) error {
	opt := &mergeOption{}

	for _, option := range options {
		option(opt)
	}

	if opt.fileReader == nil {
		opt.fileReader = &defaultFileReader{}
	}

	httpClient := http.Client{}
	if opt.transport != nil {
		httpClient.Transport = opt.transport
	}

	jsonURL := fmt.Sprintf("%s/%s", cfg.Host, cfg.SourceJSON)
	res, err := httpClient.Get(jsonURL)
	if err != nil {
		return fmt.Errorf("can not get existing file '%s', %s", cfg.SourceJSON, err.Error())
	}

	if res.StatusCode == http.StatusNotFound {
		res.Body.Close()
		return fmt.Errorf("can not find source JSON '%s'", cfg.SourceJSON)
	}

	if res.StatusCode == http.StatusForbidden {
		res.Body.Close()
		return fmt.Errorf("can not access souce JSON '%s', access denied", cfg.SourceJSON)
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return fmt.Errorf("can not get source JSON '%s', %d", cfg.SourceJSON, res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		res.Body.Close()
		return fmt.Errorf("can not read source JSON file '%s', %s", cfg.SourceJSON, err.Error())
	}

	res.Body.Close()

	var dir siem.Directives
	if err := json.Unmarshal(b, &dir); err != nil {
		return fmt.Errorf("can not parse source JSON '%s', %s", cfg.SourceJSON, err.Error())
	}

	b, err = opt.fileReader.Read(cfg.TargetJSON)
	if err != nil {
		return fmt.Errorf("can not read target JSON '%s', %s", cfg.TargetJSON, err.Error())
	}

	var targetDir siem.Directives
	if err := json.Unmarshal(b, &targetDir); err != nil {
		return fmt.Errorf("can not parse target JSON '%s', %s", cfg.TargetJSON, err.Error())
	}

	resultDir := mergeDirectives(cmd, dir, targetDir)

	b, err = json.Marshal(resultDir)
	if err != nil {
		return fmt.Errorf("can not parse result JSON for '%s', %s", cfg.SourceJSON, err.Error())
	}

	// push the directive back to the frontend
	res, err = httpClient.Post(jsonURL, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("can not apply merged directive, %s", err.Error())
	}

	defer res.Body.Close()

	// ensure connection reuse
	io.Copy(io.Discard, res.Body)

	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("can not apply merged directive, original directive not found")
	}

	if res.StatusCode == http.StatusForbidden {
		return fmt.Errorf("can not apply merged directive, access denied")
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("can not apply merged directive (%d)", res.StatusCode)
	}

	cmd.Log(fmt.Sprintf("file '%s' merged", cfg.SourceJSON))

	return nil
}

func mergeDirectives(cmd Commander, dir1, dir2 siem.Directives) siem.Directives {
	// map of directive.ID to its index in dir1.Dirs
	indexes := make(map[int]int)

	for index, directive := range dir1.Dirs {
		indexes[directive.ID] = index
	}

	newDirectives := make([]siem.Directive, 0)

	for _, directive := range dir2.Dirs {
		origIndex, ok := indexes[directive.ID]
		if !ok {
			newDirectives = append(newDirectives, directive)
			continue
		}

		same := compareDirective(dir1.Dirs[origIndex], directive)
		if same {
			continue
		}

		ok = cmd.PromptBool(fmt.Sprintf("directive #%d: '%s' is not equal to the original directive, replace?", origIndex+1, dir1.Dirs[origIndex].Name))
		if ok {
			dir1.Dirs[origIndex] = directive
		} else {
			cmd.Log(fmt.Sprintf("change in directive #%d: '%s' omitted", origIndex+1, dir1.Dirs[origIndex].Name))
		}
	}

	return siem.Directives{
		Dirs: append(dir1.Dirs, newDirectives...),
	}
}

// compareDirective perform deep comparison between the two siem.Directive(s) and return true if they are equal.
func compareDirective(dir1, dir2 siem.Directive) bool {
	if err := directiveEqual(dir1, dir2); err != nil {
		return false
	}

	return true
}

func directivesEqual(dir1, dir2 siem.Directives) error {
	for _, directive := range dir2.Dirs {
		var found bool
		for _, existing := range dir1.Dirs {
			if existing.ID == directive.ID && directive.Name == existing.Name {
				found = true

				if err := directiveEqual(existing, directive); err != nil {
					return err
				}

				break
			}
		}

		if !found {
			return fmt.Errorf("directive '%s' is not found in existing directives", directive.Name)
		}
	}

	return nil
}

func directiveEqual(dir1, dir2 siem.Directive) error {
	if dir1.ID != dir2.ID {
		return fmt.Errorf("directive ID is not equal, %d != %d", dir1.ID, dir2.ID)
	}

	if dir1.Name != dir2.Name {
		return fmt.Errorf("directive name is not equal, %s != %s", dir1.Name, dir2.Name)
	}

	if dir1.Priority != dir2.Priority {
		return fmt.Errorf("directive priority is not equal, %d != %d", dir1.Priority, dir2.Priority)
	}

	if dir1.Disabled != dir2.Disabled {
		return fmt.Errorf("directive disabled flag is not equal, %t != %t", dir1.Disabled, dir2.Disabled)
	}

	if dir1.AllRulesAlwaysActive != dir2.AllRulesAlwaysActive {
		return fmt.Errorf("directive always active flag is not equal, %t != %t", dir1.AllRulesAlwaysActive, dir2.AllRulesAlwaysActive)
	}

	if dir1.Kingdom != dir2.Kingdom {
		return fmt.Errorf("directive kingdom is not equal, %s != %s", dir1.Kingdom, dir2.Kingdom)
	}

	if dir1.Category != dir2.Category {
		return fmt.Errorf("directive category is not equal, %s != %s", dir1.Category, dir2.Category)
	}

	if len(dir1.Rules) != len(dir2.Rules) {
		return fmt.Errorf("rule count is not the same, %d != %d", len(dir1.Rules), len(dir2.Rules))
	}

	for idx, rule1 := range dir1.Rules {
		var found bool

		for _, rule2 := range dir2.Rules {
			if err := ruleEqual(rule1, rule2); err != nil {
				return fmt.Errorf("rule #%d is not equal, %s", idx, err.Error())
			} else {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("rule '%s' is not found", rule1.Name)
		}
	}

	return nil
}

func ruleEqual(rule1, rule2 rule.DirectiveRule) error {
	if rule1.Name != rule2.Name {
		return fmt.Errorf("rule name is not equal, %s != %s", rule1.Name, rule2.Name)
	}

	if rule1.Stage != rule2.Stage {
		return fmt.Errorf("rule stage is not the same, %d != %d", rule1.Stage, rule2.Stage)
	}

	if rule1.PluginID != rule2.PluginID {
		return fmt.Errorf("rule PluginID is not the same, %d != %d", rule1.PluginID, rule2.PluginID)
	}

	if !reflect.DeepEqual(rule1.PluginSID, rule2.PluginSID) {
		return fmt.Errorf("rule pluginSID is not the same,%#v != %#v", rule1.PluginSID, rule2.PluginSID)
	}

	if !reflect.DeepEqual(rule1.Product, rule2.Product) {
		return fmt.Errorf("rule Product is not the same,%#v != %#v", rule1.Product, rule2.Product)
	}

	if rule1.Category != rule2.Category {
		return fmt.Errorf("rule Category is not the same, %s != %s", rule1.Category, rule2.Category)
	}

	if !reflect.DeepEqual(rule1.SubCategory, rule2.SubCategory) {
		return fmt.Errorf("rule SubCategory is not the same,%#v != %#v", rule1.SubCategory, rule2.SubCategory)
	}

	if rule1.Occurrence != rule2.Occurrence {
		return fmt.Errorf("rule Occurrence is not the same, %d != %d", rule1.Occurrence, rule2.Occurrence)
	}

	if rule1.From != rule2.From {
		return fmt.Errorf("rule From is not the same, %s != %s", rule1.From, rule2.From)
	}

	if rule1.To != rule2.To {
		return fmt.Errorf("rule To is not the same, %s != %s", rule1.To, rule2.To)
	}

	if rule1.Type != rule2.Type {
		return fmt.Errorf("rule Type is not the same, %s != %s", rule1.Type, rule2.Type)
	}

	if rule1.PortFrom != rule2.PortFrom {
		return fmt.Errorf("rule PortFrom is not the same, %s != %s", rule1.PortFrom, rule2.PortFrom)
	}

	if rule1.PortTo != rule2.PortTo {
		return fmt.Errorf("rule PortTo is not the same, %s != %s", rule1.PortTo, rule2.PortTo)
	}

	if rule1.Protocol != rule2.Protocol {
		return fmt.Errorf("rule Protocol is not the same, %s != %s", rule1.Protocol, rule2.Protocol)
	}

	if rule1.Reliability != rule2.Reliability {
		return fmt.Errorf("rule Reliability is not the same, %d != %d", rule1.Reliability, rule2.Reliability)
	}

	if rule1.Timeout != rule2.Timeout {
		return fmt.Errorf("rule Timeout is not the same, %d != %d", rule1.Timeout, rule2.Timeout)
	}

	if rule1.Occurrence != rule2.Occurrence {
		return fmt.Errorf("rule Occurrence is not the same, %d != %d", rule1.Occurrence, rule2.Occurrence)
	}

	if rule1.StartTime != rule2.StartTime {
		return fmt.Errorf("rule StartTime is not the same, %d != %d", rule1.StartTime, rule2.StartTime)
	}

	if rule1.EndTime != rule2.EndTime {
		return fmt.Errorf("rule EndTime is not the same, %d != %d", rule1.EndTime, rule2.EndTime)
	}

	if rule1.RcvdTime != rule2.RcvdTime {
		return fmt.Errorf("rule RcvdTime is not the same, %d != %d", rule1.RcvdTime, rule2.RcvdTime)
	}

	if rule1.Status != rule2.Status {
		return fmt.Errorf("rule Status is not the same, %s != %s", rule1.Status, rule2.Status)
	}

	if !reflect.DeepEqual(rule1.Events, rule2.Events) {
		return fmt.Errorf("rule Events is not the same,%#v != %#v", rule1.Events, rule2.Events)
	}

	if rule1.StickyDiff != rule2.StickyDiff {
		return fmt.Errorf("rule StickyDiff is not the same, %s != %s", rule1.StickyDiff, rule2.StickyDiff)
	}

	if rule1.CustomData1 != rule2.CustomData1 {
		return fmt.Errorf("rule CustomData1 is not the same, %s != %s", rule1.CustomData1, rule2.CustomData1)
	}

	if rule1.CustomLabel1 != rule2.CustomLabel1 {
		return fmt.Errorf("rule CustomLabel1 is not the same, %s != %s", rule1.CustomLabel1, rule2.CustomLabel1)
	}

	if rule1.CustomData2 != rule2.CustomData2 {
		return fmt.Errorf("rule CustomData2 is not the same, %s != %s", rule1.CustomData2, rule2.CustomData2)
	}

	if rule1.CustomLabel2 != rule2.CustomLabel2 {
		return fmt.Errorf("rule CustomLabel2 is not the same, %s != %s", rule1.CustomLabel2, rule2.CustomLabel2)
	}

	if rule1.CustomData3 != rule2.CustomData3 {
		return fmt.Errorf("rule CustomData3 is not the same, %s != %s", rule1.CustomData3, rule2.CustomData3)
	}

	if rule1.CustomLabel3 != rule2.CustomLabel3 {
		return fmt.Errorf("rule CustomLabel3 is not the same, %s != %s", rule1.CustomLabel3, rule2.CustomLabel3)
	}

	return nil
}

type defaultFileReader struct {
}

func (f *defaultFileReader) Read(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}
