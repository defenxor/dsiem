package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/olivere/elastic"
)

// Plugin defines field mapping
type Plugin struct {
	Name             string       `json:"name"`
	Type             string       `json:"type"` // PluginRule || TaxonomyRule
	Output           string       `json:"output_file"`
	Index            string       `json:"index_pattern"`
	ES               string       `json:"elasticsearch_address"`
	IdentifierField  string       `json:"identifier_field"`
	IdentifierValue  string       `json:"identifier_value"`
	IdentifierFilter string       `json:"identifier_filter"`
	Fields           FieldMapping `json:"field_mapping"`
}

// FieldMapping defines field mapping
type FieldMapping struct {
	Title           string `json:"title,omitempty"` //<mapped-field, get uniq entries>
	Timestamp       string `json:"timestamp"`       //<mapped-field>
	TimestampFormat string `json:"timestamp_format"`
	Sensor          string `json:"sensor"` //<mapped-field>
	PluginID        string `json:"plugin_id,omitempty"`
	PluginSID       string `json:"plugin_sid,omitempty"`    // <auto-gen if empty>
	Product         string `json:"product,omitempty"`       // <static-per-conf>
	Category        string `json:"category,omitempty"`      //<mapped-field, used only on taxonomyRule>
	SubCategory     string `json:"subcategory,omitempty"`   //<mapped-field, used only on taxonomyRule>
	SrcIP           string `json:"src_ip"`                  //<mapped-field>
	SrcPort         string `json:"src_port"`                //<mapped-field>
	DstIP           string `json:"dst_ip"`                  //<mapped-field>
	DstPort         string `json:"dst_port"`                //<mapped-field>
	Protocol        string `json:"protocol,omitempty"`      // <mapped-field, or static>
	CustomData1     string `json:"custom_data1,omitempty"`  //<static>
	CustomLabel1    string `json:"custom_label1,omitempty"` // <mapped-field>
	CustomData2     string `json:"custom_data2,omitempty"`  // <static>
	CustomLabel2    string `json:"custom_label2,omitempty"` // <mapped-field>
	CustomData3     string `json:"custom_data3,omitempty"`
	CustomLabel3    string `json:"custom_label3,omitempty"`
}

const (
	ftCollect = iota
	ftStatic
	ftES
)

var client *elastic.Client

func initClient(esURL string) (err error) {
	client, err = elastic.NewSimpleClient(elastic.SetURL(esURL))
	return
}

func validateIndex(index string) (err error) {
	exists, err := client.IndexExists(index).Do(context.Background())
	if err == nil && !exists {
		err = errors.New("Index " + index + " doesnt exist")
	}
	return
}

func createPlugin(plugin Plugin, confFile string) (err error) {
	fmt.Print("Creating plugin (logstash config) for ", plugin.Name,
		", using ES: ", plugin.ES, " and index pattern: ", plugin.Index, "\n")

	if err = initClient(plugin.ES); err != nil {
		return
	}
	/* temporarily disable checks
	if err = validateIndex(plugin.Index); err != nil {
		return
	}
	if err = validateESField(plugin); err != nil {
		return
	}
	*/
	if getType(plugin.Fields.Title) == ftCollect {
		return createPluginCollect(plugin, confFile)
	}
	return createPluginNonCollect(plugin, confFile)
}

func createPluginNonCollect(plugin Plugin, confFile string) (err error) {

	// Prepare the struct to be used with the template
	pt := pluginTemplate{}
	pt.P = plugin
	pt.Creator = progName
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

func createPluginCollect(plugin Plugin, confFile string) (err error) {

	// taxonomyRule type plugin doesnt need to collect title since it is relying on
	// category field (which doesnt have to be unique per title) instead of Plugin_SID
	// that requires a unique SID for each title
	if plugin.Type != "PluginRule" {
		return errors.New("Only PluginRule plugin support collect: keyword in title field")
	}

	// first get the refs
	ref, err := collectTitles(plugin, confFile)
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
	pt.Creator = progName
	pt.CreateDate = time.Now().Format(time.RFC3339)
	transformToLogstashField(&pt.P.Fields)

	// Parse and execute the template, saving result to buff
	t, err := template.New(plugin.Name).Parse(templPluginCollect)
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

func collectTitles(plugin Plugin, confFile string) (c tsvRef, err error) {

	c.init(plugin.Name, confFile)

	title := strings.Replace(plugin.Fields.Title, "collect:", "", 1) + ".keyword"

	/*
		// use this to collect a lot of titles
		plugin.Index = "siem_events-*"
		title = "title.keyword"
	*/

	fmt.Print("Checking the existence of field ", title, "... ")
	exist, err := isESFieldExist(plugin.Index, title)
	if err != nil {
		return
	}
	if !exist {
		err = errors.New("Title collection requires field " + title + " to exist on index " + plugin.Index)
		return
	}
	fmt.Println("OK")

	terms := elastic.NewTermsAggregation().Field(title).Size(1000)
	ctx := context.Background()
	searchResult, err := client.Search().
		Index(plugin.Index).
		Aggregation("uniqterm", terms).
		Pretty(true).
		Do(ctx)
	if err != nil {
		return
	}

	agg, found := searchResult.Aggregations.Terms("uniqterm")
	if !found {
		err = errors.New("cannot find aggregation uniqterm in ES query result")
		return
	}
	count := len(agg.Buckets)
	if count == 0 {
		err = errors.New("cannot find matching title in field " + title + " on index " + plugin.Index)
		return
	}
	fmt.Println("Found", count, "uniq titles.")

	newSID := 1
	plugID := strings.Replace(plugin.Fields.PluginID, "static:", "", 1)
	nID, err := strconv.Atoi(plugID)
	if err != nil {
		return
	}
	for _, titleBucket := range agg.Buckets {
		t := titleBucket.Key.(string)
		// fmt.Println("found title:", t)
		// increase SID counter only if the last entry
		if shouldIncrease := c.upsert(plugin.Name, nID, &newSID, t); shouldIncrease {
			newSID++
		}
	}
	return
}

func validateESField(plugin Plugin) (err error) {
	s := reflect.ValueOf(&plugin.Fields).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		// skip empty fields
		str := f.Interface().(string)
		if str == "" {
			continue
		}
		// skip static fields
		if t := getType(str); t == ftStatic {
			continue
		}
		fmt.Print("Checking existence of field ", str, "... ")
		exist, err := isESFieldExist(plugin.Index, str)
		if err != nil {
			return err
		}
		if !exist {
			return errors.New("Cannot find any document in " + plugin.Index +
				" that has a field named " + typeOfT.Field(i).Name)
		}
		fmt.Println("OK")
		// fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
	return
}

func isESFieldExist(index string, field string) (exist bool, err error) {
	existQuery := elastic.NewExistsQuery(field)
	countService := elastic.NewCountService(client)
	countResult, err := countService.Index(index).
		Query(existQuery).
		Pretty(true).
		Do(context.Background())
	if countResult > 0 {
		exist = true
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

func parse(confFile string) (plugin Plugin, err error) {
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

func createConfig(confFile, address, index, name, typ string) error {
	if typ != "PluginRule" && typ != "TaxonomyRule" {
		return errors.New("type can only be PluginRule or TaxonomyRule")
	}

	defMappingText := "es:INSERT_ES_FIELDNAME_HERE"
	plugin := Plugin{}
	plugin.ES = address
	plugin.Index = index
	plugin.Name = name
	plugin.Output = "70_siem-plugin-" + name + ".conf"
	plugin.Type = typ
	plugin.IdentifierField = getStaticText("LOGSTASH_IDENTIFYING_FIELD") + " (example: [application] or [log_type] etc)"
	plugin.IdentifierValue = getStaticText("IDENTIFYING_FIELD_VALUE") + " (example: suricata)"
	plugin.IdentifierFilter = getStaticText("ADDITIONAL_FILTER_HERE") + " (example: and [alert])"
	plugin.Fields.Timestamp = defMappingText
	plugin.Fields.TimestampFormat = getStaticText("TIMESTAMP_FORMAT") + " (example: ISO8601)"
	plugin.Fields.Title = defMappingText + " or collect:INSERT_ES_FIELDNAME_HERE"
	plugin.Fields.SrcIP = defMappingText
	plugin.Fields.DstIP = defMappingText
	plugin.Fields.SrcPort = defMappingText
	plugin.Fields.DstPort = defMappingText
	plugin.Fields.Protocol = defMappingText + " or " + getStaticText("PROTOCOL_NAME")
	plugin.Fields.Sensor = defMappingText
	plugin.Fields.Product = getStaticText("PRODUCT_NAME")
	switch {
	case plugin.Type == "PluginRule":
		plugin.Fields.PluginID = getStaticText("PLUGIN_NUMBER")
		plugin.Fields.PluginSID = defMappingText + " (this is ignored when title above is set to collect:)"
	case plugin.Type == "TaxonomyRule":
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

func getStaticText(s string) string {
	defStaticText := "INSERT_STATIC_VALUE_HERE"
	return strings.Replace(defStaticText, "STATIC_VALUE", s, -1)
}
