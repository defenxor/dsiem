// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package dpluger

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Plugin defines field mapping
type Plugin struct {
	Name                         string       `json:"name"`
	Type                         string       `json:"type"` // SID || Taxonomy
	Output                       string       `json:"output_file"`
	Index                        string       `json:"index_pattern"`
	ES                           string       `json:"elasticsearch_address"`
	IdentifierField              string       `json:"identifier_field"`
	IdentifierValue              string       `json:"identifier_value"`
	IdentifierFilter             string       `json:"identifier_filter"`
	IdentifierBlockSource        string       `json:"identifier_block_source"`
	IdentifierBlockSourceContent string       `json:"-"`
	ESCollectionFilter           string       `json:"es_collect_filter"`
	Fields                       FieldMapping `json:"field_mapping"`
}

// FieldMapping defines field mapping
type FieldMapping struct {
	Title           string `json:"title,omitempty"`
	Timestamp       string `json:"timestamp"`
	TimestampFormat string `json:"timestamp_format"`
	Sensor          string `json:"sensor"`
	PluginID        string `json:"plugin_id,omitempty"`
	PluginSID       string `json:"plugin_sid,omitempty"`
	Product         string `json:"product,omitempty"`
	Category        string `json:"category,omitempty"`
	SubCategory     string `json:"subcategory,omitempty"`
	SrcIP           string `json:"src_ip"`
	SrcPort         string `json:"src_port"`
	DstIP           string `json:"dst_ip"`
	DstPort         string `json:"dst_port"`
	Protocol        string `json:"protocol,omitempty"`
	CustomData1     string `json:"custom_data1,omitempty"`
	CustomLabel1    string `json:"custom_label1,omitempty"`
	CustomData2     string `json:"custom_data2,omitempty"`
	CustomLabel2    string `json:"custom_label2,omitempty"`
	CustomData3     string `json:"custom_data3,omitempty"`
	CustomLabel3    string `json:"custom_label3,omitempty"`
}

const (
	ftCollect = iota
	ftStatic
	ftES
)

var collector esCollector

// Parse read dpluger config from confFile and returns a Plugin
func Parse(confFile string) (plugin Plugin, err error) {
	file, err := os.Open(confFile)
	if err != nil {
		return
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return
	}
	err = json.Unmarshal(byteValue, &plugin)
	return
}

// CreateConfig generates dpluger config file
func CreateConfig(confFile, address, index, name, typ string) error {
	if typ != "SID" && typ != "Taxonomy" {
		return errors.New("type can only be SID or Taxonomy")
	}

	defMappingText := "es:INSERT_ES_FIELDNAME_HERE"
	plugin := Plugin{}
	plugin.ES = address
	plugin.Index = index
	plugin.Name = name
	plugin.Output = "70_dsiem-plugin_" + name + ".conf"
	plugin.Type = typ
	plugin.IdentifierField = getStaticText("LOGSTASH_IDENTIFYING_FIELD") + " (example: [application] or [fields][log_type] etc)"
	plugin.IdentifierValue = getStaticText("IDENTIFYING_FIELD_VALUE") + " (example: suricata)"
	plugin.IdentifierFilter = getStaticText("ADDITIONAL_FILTER") + " (example: and [alert])"
	plugin.ESCollectionFilter = getStaticText("ES_TERM_FILTER") + " (example: type=http will only collect SIDs from documents whose type field is http). Separate multiple term with ; character"
	plugin.Fields.Timestamp = defMappingText
	plugin.Fields.TimestampFormat = getStaticText("TIMESTAMP_FORMAT") + " (example: ISO8601)"
	plugin.Fields.Title = defMappingText
	plugin.Fields.SrcIP = defMappingText
	plugin.Fields.DstIP = defMappingText
	plugin.Fields.SrcPort = defMappingText
	plugin.Fields.DstPort = defMappingText
	plugin.Fields.Protocol = defMappingText + " or " + getStaticText("PROTOCOL_NAME")
	plugin.Fields.Sensor = defMappingText
	plugin.Fields.Product = getStaticText("PRODUCT_NAME")
	plugin.Fields.CustomLabel1 = "INSERT CUSTOM FIELD NAME FOR CUSTOMDATA1 HERE. Remove this and CustomData1 if not used."
	plugin.Fields.CustomData1 = defMappingText
	plugin.Fields.Category = defMappingText + " or " + getStaticText("CATEGORY_NAME")
	switch {
	case plugin.Type == "SID":
		plugin.Fields.PluginID = getStaticText("PLUGIN_NUMBER")
		plugin.Fields.PluginSID = defMappingText + " or collect:INSERT_ES_FIELDNAME_HERE"
	case plugin.Type == "Taxonomy":
		plugin.Fields.SubCategory = defMappingText
	}
	bConfig, err := json.MarshalIndent(plugin, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(confFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	f.SetDeadline(time.Now().Add(60 * time.Second))
	_, err = f.WriteString(string(bConfig) + "\n")
	return err
}

type CreatePluginConfig struct {
	Plugin      Plugin
	ConfigFile  string
	Creator     string
	Validate    bool
	UsePipeline bool
	SIDListFile string
}

func (cfg CreatePluginConfig) isCollect() bool {
	return getType(cfg.Plugin.Fields.PluginSID) == ftCollect
}

// CreatePlugin starts plugin creation
func CreatePlugin(cfg CreatePluginConfig) error {
	fmt.Printf("Creating plugin (logstash config) for %s using ES: %s and index pattern: %s\n", cfg.Plugin.Name, cfg.Plugin.ES, cfg.Plugin.Index)

	var err error
	if collector, err = newESCollector(cfg.Plugin.ES); err != nil {
		return err
	}

	if cfg.Validate {
		if err := collector.ValidateIndex(cfg.Plugin.Index); err != nil {
			return err
		}

		if err := validateESField(cfg.Plugin); err != nil {
			return err
		}
	}

	if cfg.isCollect() {
		return createPluginCollect(cfg)
	}

	return createPluginNonCollect(cfg)
}

func createPluginNonCollect(cfg CreatePluginConfig) error {

	// Prepare the struct to be used with the template
	pt := pluginTemplate{
		Plugin:     cfg.Plugin,
		Creator:    cfg.Creator,
		CreateDate: time.Now().Format(time.RFC3339),
	}

	FieldMappingToLogstashField(&pt.Plugin.Fields)

	var identifierBlock string
	if cfg.Plugin.IdentifierBlockSource != "" {
		b, err := os.ReadFile(cfg.Plugin.IdentifierBlockSource)
		if err == nil {
			pt.Plugin.IdentifierBlockSourceContent = string(b)
			identifierBlock = templWithIdentifierBlockContent
		} else {
			fmt.Printf("error reading block source file '%s', skipping add block source from file, %s\n", cfg.Plugin.IdentifierBlockSource, err.Error())
		}
	}

	if identifierBlock == "" {
		if cfg.UsePipeline {
			identifierBlock = templPipeline
		} else {
			identifierBlock = templNonPipeline
		}
	}

	if cfg.SIDListFile != "" {
		var ref tsvRef
		ref.initWithConfig(cfg.SIDListFile)

		pt.Ref = ref
		pt.SIDListGroup = ref.GroupByCustomData()
	}

	// Parse and execute the template
	templateText := templHeader + identifierBlock + templPluginNonCollect + templFooter

	t, err := template.New(cfg.Plugin.Name).Funcs(templateFunctions).Parse(templateText)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	err = t.Execute(w, pt)
	w.Flush()
	if err != nil {
		return err
	}

	// prepare plugin output file
	dir := path.Dir(cfg.ConfigFile)
	fname := path.Join(dir, cfg.Plugin.Output)
	f, err := os.OpenFile(fname, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	err = removeEmptyLines(&buf, f)
	if err != nil {
		return err
	}

	if cfg.Plugin.Type != "SID" {
		return nil
	}

	if cfg.SIDListFile != "" {
		fmt.Println("Done creating plugin, no TSV file created, since dpluger already supplied with a TSV file")
		return nil
	}

	fmt.Println("Done creating plugin, now creating TSV for directive auto generation ..")
	// first get the refs
	ref, err := collectPair(cfg.Plugin, cfg.ConfigFile, cfg.Plugin.ESCollectionFilter, cfg.Validate)
	if err != nil {
		return err
	}
	if err := ref.save(); err != nil {
		return err
	}

	return nil
}

var ErrNonSIDCollect = errors.New("only SID-type plugin support collect: keyword")

func createPluginCollect(cfg CreatePluginConfig) error {

	// Prepare the struct to be used with template
	SIDField := LogstashFieldNotation(strings.Replace(cfg.Plugin.Fields.Title, "collect:", "", 1))

	pt := pluginTemplate{
		Plugin:        cfg.Plugin,
		Creator:       cfg.Creator,
		SIDField:      "%{" + SIDField + "}",
		SIDFieldPlain: SIDField,
		CreateDate:    time.Now().Format(time.RFC3339),
	}

	if cfg.SIDListFile == "" {
		// Taxonomy type plugin doesnt need to collect title since it is relying on
		// category field (which doesnt have to be unique per title) instead of Plugin_SID
		// that requires a unique SID for each title
		if cfg.Plugin.Type != "SID" {
			return ErrNonSIDCollect
		}

		// first get the refs
		ref, err := collectSID(cfg.Plugin, cfg.ConfigFile, cfg.Plugin.ESCollectionFilter, cfg.Validate)
		if err != nil {
			return err
		}

		if err := ref.save(); err != nil {
			return err
		}

		pt.Ref = ref
	} else {
		var ref tsvRef
		ref.initWithConfig(cfg.SIDListFile)

		pt.Ref = ref
		pt.SIDListGroup = ref.GroupByCustomData()
	}

	FieldMappingToLogstashField(&pt.Plugin.Fields)
	var identifierBlock string
	if cfg.Plugin.IdentifierBlockSource != "" {
		b, err := os.ReadFile(cfg.Plugin.IdentifierBlockSource)
		if err == nil {
			pt.Plugin.IdentifierBlockSourceContent = string(b)
			identifierBlock = templWithIdentifierBlockContent
		} else {
			fmt.Printf("error reading block source file '%s', skipping add block source from file, %s\n", cfg.Plugin.IdentifierBlockSource, err.Error())
		}
	}

	if identifierBlock == "" {
		if cfg.UsePipeline {
			identifierBlock = templPipeline
		} else {
			identifierBlock = templNonPipeline
		}
	}

	// Parse and execute the template
	templateText := templHeader + identifierBlock + templPluginCollect + templFooter

	// Parse and execute the template, saving result to buff
	t, err := template.New(cfg.Plugin.Name).Funcs(templateFunctions).Parse(templateText)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	err = t.Execute(w, pt)
	w.Flush()
	if err != nil {
		return err
	}

	// prepare plugin output file
	dir := path.Dir(cfg.ConfigFile)
	fname := path.Join(dir, cfg.Plugin.Output)
	f, err := os.OpenFile(fname, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	err = removeEmptyLines(&buf, f)
	if err != nil {
		return err
	}

	return nil
}

func counter() func() int {
	i := -1
	return func() int {
		i++
		return i
	}
}

func removeEmptyLines(input io.Reader, output io.Writer) (err error) {
	scanner := bufio.NewScanner(input)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	for scanner.Scan() {
		s := scanner.Text()
		s = strings.TrimRight(s, " ")
		fmt.Fprintf(w, "%s\n", s)
	}
	w.Flush()
	regex, err := regexp.Compile(`\n\n+`)
	if err != nil {
		return
	}
	s := regex.ReplaceAllString(buf.String(), "\n\n")

	fmt.Fprintf(output, "%s", s)
	err = scanner.Err()
	return
}

func setField(f *FieldMapping, field string, value string) {
	v := reflect.ValueOf(f).Elem().FieldByName(field)
	if v.IsValid() {
		v.SetString(value)
	}
}

func collectPair(plugin Plugin, confFile, esFilter string, validate bool) (tsvRef, error) {
	var (
		ctx = context.Background()
		c   = tsvRef{}
		err error
	)

	sidSource := strings.Replace(plugin.Fields.PluginSID, "es:", "", 1)
	sidSource, err = checkKeyword(ctx, plugin.Index, sidSource)
	if err != nil {
		return c, err
	}

	titleSource := strings.Replace(plugin.Fields.Title, "es:", "", 1)
	titleSource, err = checkKeyword(ctx, plugin.Index, titleSource)
	if err != nil {
		return c, err
	}

	shouldCollectCategory := false
	categorySource := plugin.Fields.Category
	if strings.Contains(plugin.Fields.Category, "es:") {
		shouldCollectCategory = true
		categorySource, err = checkKeyword(ctx, plugin.Index, strings.Replace(plugin.Fields.Category, "es:", "", 1))
		if err != nil {
			return c, err
		}
	}

	if validate {
		fmt.Printf("Checking the existence of field '%s' ... ", sidSource)
		var exist bool
		exist, err = collector.IsESFieldExist(plugin.Index, sidSource)
		if err != nil {
			return c, err
		}

		if !exist {
			return c, fmt.Errorf("Plugin SID collection requires field '%s' to exist on index '%s'", sidSource, plugin.Index)
		}

		fmt.Println("OK")

		fmt.Printf("Checking the existence of field '%s' ... ", titleSource)
		exist, err = collector.IsESFieldExist(plugin.Index, titleSource)
		if err != nil {
			return c, err
		}

		if !exist {
			return c, fmt.Errorf("Plugin SID collection requires field '%s' to exist on index '%s'", titleSource, plugin.Index)
		}

		fmt.Println("OK")

		if shouldCollectCategory {
			fmt.Printf("Checking the existence of field '%s' ... ", categorySource)
			exist, err = collector.IsESFieldExist(plugin.Index, categorySource)
			if err != nil {
				return c, err
			}

			if !exist {
				return c, fmt.Errorf("Plugin SID collection requires field '%s' to exist on index '%s'", categorySource, plugin.Index)
			}

			fmt.Println("OK")
		}
	}

	fmt.Printf("Collecting unique entries for field '%s' and '%s' on index '%s' ... ", titleSource, sidSource, plugin.Index)
	if esFilter != "" {
		fmt.Printf("Limiting collection with term '%s' ", esFilter)
	}

	fmt.Println("OK")

	return collector.CollectPair(plugin, confFile, sidSource, esFilter, titleSource, categorySource, shouldCollectCategory)
}

func collectSID(plugin Plugin, confFile, esFilter string, validate bool) (tsvRef, error) {
	var (
		ctx = context.Background()
		c   tsvRef
		err error
	)

	sidSource := strings.Replace(plugin.Fields.PluginSID, "collect:", "", 1)
	sidSource, err = checkKeyword(ctx, plugin.Index, sidSource)
	if err != nil {
		return c, err
	}

	shouldCollectCategory := false
	categorySource := plugin.Fields.Category
	if strings.Contains(plugin.Fields.Category, "es:") {
		shouldCollectCategory = true
		categorySource, err = checkKeyword(ctx, plugin.Index, strings.Replace(plugin.Fields.Category, "es:", "", 1))
		if err != nil {
			return c, err
		}
	}

	if validate {
		fmt.Printf("Checking the existence of field '%s' ... ", sidSource)

		exist, err := collector.IsESFieldExist(plugin.Index, sidSource)
		if err != nil {
			return c, err
		}

		if !exist {
			return c, fmt.Errorf("Plugin SID collection requires field '%s' to exist on index '%s'", sidSource, plugin.Index)
		}

		if shouldCollectCategory {
			fmt.Printf("Checking the existence of field '%s' .... ", categorySource)
			exist, err := collector.IsESFieldExist(plugin.Index, categorySource)
			if err != nil {
				return c, err
			}

			if !exist {
				return c, fmt.Errorf("Plugin SID collection requires field '%s' to exist on index '%s'", categorySource, plugin.Index)
			}
		}

		fmt.Println("OK")
	}

	fmt.Printf("Collecting unique entries from '%s' on index '%s' to create Plugin SIDs ... \n", sidSource, plugin.Index)
	if esFilter != "" {
		fmt.Printf("Limitting collection with term '%s'\n", esFilter)
	}

	return collector.Collect(plugin, confFile, sidSource, esFilter, categorySource, shouldCollectCategory)
}

func validateESField(plugin Plugin) error {
	s := reflect.ValueOf(&plugin.Fields).Elem()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		// skip empty fields
		str := f.Interface().(string)
		if str == "" {
			continue
		}

		// skip non-es field
		if t := getType(str); t != ftES {
			continue
		}

		str = strings.Replace(str, "es:", "", 1)

		fmt.Printf("Checking existence of field '%s' ... ", str)
		exist, err := collector.IsESFieldExist(plugin.Index, str)
		if err != nil {
			return err
		}

		if !exist {
			return fmt.Errorf("can not find any document in '%s' that has a field named '%s'", plugin.Index, str)
		}

		fmt.Println("OK")
	}

	return nil
}

func getType(s string) int {
	switch {
	case strings.HasPrefix(s, "collect:"):
		return ftCollect
	case strings.HasPrefix(s, "es:"):
		return ftES
	default:
		return ftStatic
	}
}

func getStaticText(s string) string {
	defStaticText := "INSERT_STATIC_VALUE_HERE"
	return strings.Replace(defStaticText, "STATIC_VALUE", s, -1)
}

func checkKeyword(ctx context.Context, index, field string) (string, error) {
	fmt.Printf("Checking field '%s' mapping type ... ", field)
	fieldType, haskeyword, err := collector.FieldType(context.Background(), index, field)
	if err == ErrFieldMappingNotExist {
		return "", fmt.Errorf("no mapping found for field '%s', %s", field, err.Error())
	}

	if err != nil {
		return "", fmt.Errorf("error while checking field mapping for '%s', %s", field, err.Error())
	}

	if fieldType != FieldTypeKeyword && haskeyword {
		fmt.Printf("found '%s' with .keyword field available, adding .keyword\n", fieldType)
		field = fmt.Sprintf("%s.keyword", field)
	} else if fieldType != FieldTypeKeyword && !haskeyword {
		fmt.Printf("found '%s' but .keyword field not available, using field-name as it is\n", fieldType)
	} else {
		fmt.Printf("found '%s', skipping .keyword\n", fieldType)
	}

	return field, nil
}
