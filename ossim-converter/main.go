package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	ossimDirectiveFile = "user.xml"
	resultFile         = "directives_ossim.json"
)

var progDir string
var devEnv bool

type directives struct {
	Directives []directive `xml:"directive" json:"directives"`
}

type directive struct {
	ID       int    `xml:"id,attr" json:"id"`
	Name     string `xml:"name,attr" json:"name"`
	Priority int    `xml:"priority,attr" json:"priority"`
	Kingdom  string `json:"kingdom"`
	Category string `json:"category"`
	Rules    []rule `xml:"rule" json:"rules"`
}

type rules struct {
	Rules []rule `xml:"rule" json:"rule"`
}

type rule struct {
	Stage        int     `json:"stage"`
	Name         string  `xml:"name,attr" json:"name"`
	PluginID     int64   `xml:"plugin_id,attr" json:"plugin_id"`
	PluginSIDstr string  `xml:"plugin_sid,attr" json:"plugin_sid_str,omitempty"`
	PluginSID    []int64 `json:"plugin_sid,omitempty"`
	Occurrence   int64   `xml:"occurrence,attr" json:"occurrence"`
	From         string  `xml:"from,attr" json:"from"`
	To           string  `xml:"to,attr" json:"to"`
	PortFrom     string  `xml:"port_from,attr" json:"port_from"`
	PortTo       string  `xml:"port_to,attr" json:"port_to"`
	Reliability  int     `xml:"reliability,attr" json:"reliability"`
	Timeout      int64   `xml:"time_out,attr" json:"timeout"`
	Protocol     string  `xml:"protocol,attr" json:"protocol"`
	Rules        []rules `xml:"rules" json:"rules,omitempty"`
}

func init() {
	b := flag.Bool("dev", false, "enable/disable dev env specific settings.")
	flag.Parse()
	devEnv = *b
	d, _ := getDir()
	progDir = d
}

func main() {
	setupLogger()
	filename, err := createTempOSSIMFile()
	if err != nil {
		logger.Info(err)
		return
	}
	err = parseOSSIMTSVs()
	if err != nil {
		logger.Info(err)
		return
	}
	res, err := createSIEMDirective(filename)
	if err != nil {
		logger.Info(err)
		return
	}
	logger.Info("Done. Results in " + res)
}

func createSIEMDirective(tempXMLFile string) (resFile string, err error) {
	xmlFile, err := os.Open(tempXMLFile)
	if err != nil {
		return "", err
	}
	defer xmlFile.Close()
	defer os.Remove(tempXMLFile)

	byteValue, _ := ioutil.ReadAll(xmlFile)
	sValue := string(byteValue)
	if sValue == "" {
		return "", errors.New("Cannot read content from " + tempXMLFile)
	}

	var d directives
	xml.Unmarshal(byteValue, &d)

	for i := range d.Directives {
		// let it be empty if we cant find it
		var kingdom, category string
		kingdom, category = findKingdomCategory(d.Directives[i].ID)
		d.Directives[i].Kingdom = kingdom
		d.Directives[i].Category = category

		// flatten rules
		res := flattenRule(d.Directives[i].Rules, []rule{})
		d.Directives[i].Rules = res
		// renumber rule's stage and convert plugin_sid from string to array of ints
		for j := range d.Directives[i].Rules {
			d.Directives[i].Rules[j].Stage = j + 1
			strSids := strings.Split(d.Directives[i].Rules[j].PluginSIDstr, ",")
			nArr := []int64{}
			for k := range strSids {
				n, _ := strconv.Atoi(strSids[k])
				nArr = append(nArr, int64(n))
			}
			d.Directives[i].Rules[j].PluginSIDstr = ""
			d.Directives[i].Rules[j].PluginSID = nArr
			// fix defaults and formatting
			if d.Directives[i].Rules[j].Protocol == "" {
				d.Directives[i].Rules[j].Protocol = "ANY"
			}
			if strings.Contains(d.Directives[i].Rules[j].From, ":") {
				v := strings.Split(d.Directives[i].Rules[j].From, ":")
				d.Directives[i].Rules[j].From = ":" + v[0]
			}
			if strings.Contains(d.Directives[i].Rules[j].To, ":") {
				v := strings.Split(d.Directives[i].Rules[j].To, ":")
				d.Directives[i].Rules[j].To = ":" + v[0]
			}
			if strings.Contains(d.Directives[i].Rules[j].PortFrom, ":") {
				v := strings.Split(d.Directives[i].Rules[j].PortFrom, ":")
				d.Directives[i].Rules[j].PortFrom = ":" + v[0]
			}
			if strings.Contains(d.Directives[i].Rules[j].PortTo, ":") {
				v := strings.Split(d.Directives[i].Rules[j].PortTo, ":")
				d.Directives[i].Rules[j].PortTo = ":" + v[0]
			}
		}
	}

	b, err := json.MarshalIndent(d, "", "  ")
	// fmt.Println(string(b))

	resFile = path.Join(progDir, resultFile)
	err = writeToFile(string(b), resFile)
	return resFile, nil
}

func flattenRule(node []rule, target []rule) (merged []rule) {
	for i := range node {
		r := node[i]
		if r.Rules != nil {
			r.Rules = []rules{}
		}
		target = append(target, r)
		if node[i].Rules != nil {
			for j := range node[i].Rules {
				return flattenRule(node[i].Rules[j].Rules, target)
			}
		}
	}
	return target
}

func createTempOSSIMFile() (filename string, err error) {
	src := path.Join(progDir, ossimDirectiveFile)
	if !fileExist(src) {
		return "", errors.New(src + " doesn't exist.")
	}
	from, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer from.Close()

	dst := path.Join(progDir, ossimDirectiveFile+".tmp")

	_ = os.Remove(dst)

	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(to, from)
	if err != nil {
		return "", err
	}
	to.Close()
	if err = insertDirectivesXML(dst); err != nil {
		return "", err
	}
	err = appendToFile("</directives>", dst)
	return dst, err
}
