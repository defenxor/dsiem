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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	Name               string       `json:"name"`
	Type               string       `json:"type"` // SID || Taxonomy
	Output             string       `json:"output_file"`
	Index              string       `json:"index_pattern"`
	ES                 string       `json:"elasticsearch_address"`
	IdentifierField    string       `json:"identifier_field"`
	IdentifierValue    string       `json:"identifier_value"`
	IdentifierFilter   string       `json:"identifier_filter"`
	ESCollectionFilter string       `json:"es_collect_filter"`
	Fields             FieldMapping `json:"field_mapping"`
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

var esVersion int
var collector esCollector

// Parse read dpluger config from confFile and returns a Plugin
func Parse(confFile string) (plugin Plugin, err error) {
	file, err := os.Open(confFile)
	if err != nil {
		return
	}
	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
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
	plugin.ESCollectionFilter = getStaticText("ES_TERM_FILTER") + " (example: type=http will only collect SIDs from documents whose type field is http)"
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
	switch {
	case plugin.Type == "SID":
		plugin.Fields.PluginID = getStaticText("PLUGIN_NUMBER")
		plugin.Fields.PluginSID = defMappingText + " or collect:INSERT_ES_FIELDNAME_HERE"
	case plugin.Type == "Taxonomy":
		plugin.Fields.Category = defMappingText
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

// CreatePlugin starts plugin creation
func CreatePlugin(plugin Plugin, confFile, creator string, validate bool) (err error) {
	fmt.Print("Creating plugin (logstash config) for ", plugin.Name,
		", using ES: ", plugin.ES, " and index pattern: ", plugin.Index, "\n")

	if collector, err = newESCollector(plugin.ES); err != nil {
		return
	}

	if validate {
		if err = collector.ValidateIndex(plugin.Index); err != nil {
			return
		}
		if err = validateESField(plugin); err != nil {
			return
		}
	}
	if getType(plugin.Fields.PluginSID) == ftCollect {
		return createPluginCollect(plugin, confFile, creator, plugin.ESCollectionFilter, validate)
	}
	return createPluginNonCollect(plugin, confFile, creator)
}

func createPluginNonCollect(plugin Plugin, confFile, creator string) (err error) {

	// Prepare the struct to be used with the template
	pt := pluginTemplate{}
	pt.P = plugin
	pt.Creator = creator
	pt.CreateDate = time.Now().Format(time.RFC3339)

	transformToLogstashField(&pt.P.Fields)

	// Parse and execute the template
	t, err := template.New(plugin.Name).Parse(templPluginNonCollect)
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

	// Prepare plugin output file
	dir := path.Dir(confFile)
	fname := path.Join(dir, plugin.Output)
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

func createPluginCollect(plugin Plugin, confFile, creator, esFilter string, validate bool) (err error) {

	// Taxnomy type plugin doesnt need to collect title since it is relying on
	// category field (which doesnt have to be unique per title) instead of Plugin_SID
	// that requires a unique SID for each title
	if plugin.Type != "SID" {
		return errors.New("Only SID-type plugin support collect: keyword")
	}

	// first get the refs
	ref, err := collectSID(plugin, confFile, esFilter, validate)
	if err != nil {
		return err
	}
	if err := ref.save(); err != nil {
		return err
	}

	// Prepare the struct to be used with template
	pt := pluginTemplate{}
	pt.P = plugin
	pt.R = ref
	pt.Creator = creator
	pt.SIDField = getLogstashFieldNotation(
		strings.Replace(plugin.Fields.Title, "collect:", "", 1))
	pt.SIDFieldPlain = pt.SIDField
	pt.SIDField = "%{" + pt.SIDField + "}"
	pt.CreateDate = time.Now().Format(time.RFC3339)
	transformToLogstashField(&pt.P.Fields)

	// Parse and execute the template, saving result to buff
	t, err := template.New(plugin.Name).Funcs(template.FuncMap{"counter": counter}).Parse(templPluginCollect)
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
	dir := path.Dir(confFile)
	fname := path.Join(dir, plugin.Output)
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

func transformToLogstashField(fields *FieldMapping) {
	// iterate over fields to change them to logstash notation
	s := reflect.ValueOf(fields).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		// skip empty fields
		str := f.Interface().(string)
		if str == "" {
			continue
		}
		var v string
		if t := getType(str); t == ftES {
			// convert to logstash [field][subfield] notation
			v = getLogstashFieldNotation(str)
			// do this except for timestamp, as it is only used in date filter
			if typeOfT.Field(i).Name != "Timestamp" {
				v = "%{" + v + "}"
			}
		} else {
			v = str
		}
		// set it
		setField(fields, typeOfT.Field(i).Name, v)
		// fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
}

func getLogstashFieldNotation(src string) (res string) {
	s := strings.Replace(src, "es:", "", 1)
	s = strings.Replace(s, "collect:", "", 1)
	s = strings.Replace(s, ".", "][", -1)
	s = strings.Replace(s, s, "["+s, 1)
	s = strings.Replace(s, s, s+"]", 1)
	res = s
	return
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

func collectSID(plugin Plugin, confFile, esFilter string, validate bool) (c tsvRef, err error) {
	sidSource := strings.Replace(plugin.Fields.PluginSID, "collect:", "", 1) + ".keyword"

	if validate {
		fmt.Print("Checking the existence of field ", sidSource, "... ")
		var exist bool
		exist, err = collector.IsESFieldExist(plugin.Index, sidSource)
		if err != nil {
			return
		}
		if !exist {
			err = errors.New("Plugin SID collection requires field " + sidSource + " to exist on index " + plugin.Index)
			return
		}
		fmt.Println("OK")
	}
	fmt.Println("Collecting unique entries from " + sidSource + " on index " + plugin.Index + " to create Plugin SIDs ...")
	if esFilter != "" {
		fmt.Println("Limiting collection with term " + esFilter)
	}
	return collector.Collect(plugin, confFile, sidSource, esFilter)
}

func validateESField(plugin Plugin) (err error) {
	s := reflect.ValueOf(&plugin.Fields).Elem()
	// typeOfT := s.Type()
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
		fmt.Print("Checking existence of field ", str, "... ")
		exist, err := collector.IsESFieldExist(plugin.Index, str)
		if err != nil {
			return err
		}
		if !exist {
			return errors.New("Cannot find any document in " + plugin.Index +
				" that has a field named " + str)
		}
		fmt.Println("OK")
		// fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
	return
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
