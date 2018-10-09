package ossimcnv

import (
	"dsiem/internal/pkg/shared/fs"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

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
	Stage          int      `json:"stage"`
	Name           string   `xml:"name,attr" json:"name"`
	Type           string   `json:"type"`
	PluginID       int64    `xml:"plugin_id,attr" json:"plugin_id,omitempty"`
	PluginSIDstr   string   `xml:"plugin_sid,attr" json:"plugin_sid_str,omitempty"`
	PluginSID      []int64  `json:"plugin_sid,omitempty"`
	Productstr     string   `xml:"product,attr" json:"product_str,omitempty"`
	Product        []string `json:"product,omitempty"`
	Category       string   `xml:"category,attr" json:"category,omitempty"`
	SubCategorystr string   `xml:"subcategory,attr" json:"subcategory_str,omitempty"`
	SubCategory    []string `json:"subcategory,omitempty"`
	Occurrence     int64    `xml:"occurrence,attr" json:"occurrence"`
	From           string   `xml:"from,attr" json:"from"`
	To             string   `xml:"to,attr" json:"to"`
	PortFrom       string   `xml:"port_from,attr" json:"port_from"`
	PortTo         string   `xml:"port_to,attr" json:"port_to"`
	Reliability    int      `xml:"reliability,attr" json:"reliability"`
	Timeout        int64    `xml:"time_out,attr" json:"timeout"`
	Protocol       string   `xml:"protocol,attr" json:"protocol"`
	Rules          []rules  `xml:"rules" json:"rules,omitempty"`
}

func insertDirectivesXML(filename string) error {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, `<?xml version="1.0" encoding="UTF-8"?>`) {
			lines[i] = `<?xml version="1.0" encoding="UTF-8"?>` + "\n<directives>"
			break
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(filename, []byte(output), 0644)
	return err
}

// CreateSIEMDirective create directive.json in resFile, based on patched OSSIM xml in tempXMLFile
func CreateSIEMDirective(tempXMLFile string, resFile string) (err error) {
	xmlFile, err := os.Open(tempXMLFile)
	if err != nil {
		return err
	}
	defer xmlFile.Close()
	defer os.Remove(tempXMLFile)

	byteValue, _ := ioutil.ReadAll(xmlFile)
	sValue := string(byteValue)
	if sValue == "" {
		return errors.New("Cannot read content from " + tempXMLFile)
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

		// renumber rule's stage and convert plugin_sid, product, and subcategory from string to array of ints
		for j := range d.Directives[i].Rules {
			d.Directives[i].Rules[j].Stage = j + 1
			thisRule := &d.Directives[i].Rules[j]
			// define type, PluginRule or TaxonomyRule
			if thisRule.PluginSIDstr != "" {
				thisRule.Type = "PluginRule"
			} else {
				thisRule.Type = "TaxonomyRule"
			}

			if thisRule.Type == "PluginRule" {
				// for plugin_sid

				// first handle 1:Plugin_SID in Plugin_SID by copying from the referenced rule
				if strings.Contains(thisRule.PluginSIDstr, ":") {
					v := strings.Split(thisRule.PluginSIDstr, ":")
					n, err := strconv.Atoi(v[0])
					if err != nil {
						return err
					}
					n--
					thisRule.PluginSID = d.Directives[i].Rules[n].PluginSID

				} else {
					// the rest, convert sid,sid,sid to []int
					strSids := strings.Split(thisRule.PluginSIDstr, ",")
					nArr := []int64{}
					for k := range strSids {
						n, _ := strconv.Atoi(strSids[k])
						nArr = append(nArr, int64(n))
					}
					thisRule.PluginSID = nArr
				}
				thisRule.PluginSIDstr = ""
			}

			if thisRule.Type == "TaxonomyRule" {
				// for product
				strSids := strings.Split(thisRule.Productstr, ",")
				sArr := []string{}
				for _, v := range strSids {
					n, _ := strconv.Atoi(v)
					for i := range oProd.Products {
						if oProd.Products[i].ID == n {
							sArr = append(sArr, oProd.Products[i].Name)
							break
						}
					}
				}
				thisRule.Productstr = ""
				thisRule.Product = sArr

				// for product category and subcategory, these are optional and may not be present, e.g. in directive 501742
				pCatID, err := strconv.Atoi(thisRule.Category)
				if err == nil {
					for i := range oPcat.Categories {
						if oPcat.Categories[i].ID == pCatID {
							// replace the number with name/string representation
							thisRule.Category = oPcat.Categories[i].Name
						}
					}

					if thisRule.SubCategorystr != "" {
						strSids := strings.Split(thisRule.SubCategorystr, ",")
						sArr := []string{}
						// first need to find the product category
						for _, v := range strSids {
							n, _ := strconv.Atoi(v)
							for i := range oPsub.SubCategories {
								if oPsub.SubCategories[i].ID == n && oPsub.SubCategories[i].CatID == pCatID {
									sArr = append(sArr, oPsub.SubCategories[i].Name)
									break
								}
							}
						}
						thisRule.SubCategorystr = ""
						thisRule.SubCategory = sArr
					}
				}
			}

			// fix defaults and formatting
			if thisRule.Protocol == "" {
				thisRule.Protocol = "ANY"
			}
			if strings.Contains(thisRule.From, ":") {
				v := strings.Split(thisRule.From, ":")
				thisRule.From = ":" + v[0]
			}
			if strings.Contains(thisRule.To, ":") {
				v := strings.Split(thisRule.To, ":")
				thisRule.To = ":" + v[0]
			}
			if strings.Contains(thisRule.PortFrom, ":") {
				v := strings.Split(thisRule.PortFrom, ":")
				thisRule.PortFrom = ":" + v[0]
			}
			if strings.Contains(thisRule.PortTo, ":") {
				v := strings.Split(thisRule.PortTo, ":")
				thisRule.PortTo = ":" + v[0]
			}
		}
	}
	b, err := json.MarshalIndent(d, "", "  ")
	// fmt.Println(string(b))

	err = fs.OverwriteFile(string(b), resFile)
	return nil
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

// CreateTempOSSIMFile creates a patched temporary XML file based on original OSSIM XML in src
func CreateTempOSSIMFile(src string) (filename string, err error) {
	if !fs.FileExist(src) {
		return "", errors.New(src + " doesn't exist.")
	}
	from, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer from.Close()

	dst := src + ".tmp"

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
	err = fs.AppendToFile("</directives>", dst)
	return dst, err
}
