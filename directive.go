package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const (
	confDir           = "conf"
	directiveFileGlob = "directives_*.json"
)

var uCases directives
var eventChannel chan normalizedEvent

func startDirective(d directive, c chan normalizedEvent) {
	// logger.Info("Started directive ", d.ID)
	for {
		// handle incoming event
		evt := <-c
		if !doesEventMatchRule(&evt, &d.Rules[0], 0) {
			continue
		}

		logInfo("directive "+strconv.Itoa(d.ID)+" found matched event.", evt.ConnID)
		go backlogManager(&evt, &d)
	}
}

func doesEventMatchRule(e *normalizedEvent, r *directiveRule, connID uint64) bool {
	if e.PluginID != r.PluginID {
		return false
	}

	sidMatch := false
	for i := range r.PluginSID {
		if r.PluginSID[i] == e.PluginSID {
			sidMatch = true
			break
		}
	}
	if sidMatch == false {
		return false
	}

	eSrcInHomeNet := e.srcIPInHomeNet()
	if r.From == "HOME_NET" && eSrcInHomeNet == false {
		return false
	}
	if r.From == "!HOME_NET" && eSrcInHomeNet == true {
		return false
	}
	// covers  r.From == "IP", r.From == "IP1, IP2", r.From == CIDR-netaddr, r.From == "CIDR1, CIDR2"
	if r.From != "HOME_NET" && r.From != "!HOME_NET" && r.From != "ANY" &&
		!caseInsensitiveContains(r.From, e.SrcIP) && !isIPinCIDR(e.SrcIP, r.From, connID) {
		return false
	}

	eDstInHomeNet := e.dstIPInHomeNet()
	if r.To == "HOME_NET" && eDstInHomeNet == false {
		return false
	}
	if r.To == "!HOME_NET" && eDstInHomeNet == true {
		return false
	}
	// covers  r.To == "IP", r.To == "IP1, IP2", r.To == CIDR-netaddr, r.To == "CIDR1, CIDR2"
	if r.To != "HOME_NET" && r.To != "!HOME_NET" && r.To != "ANY" &&
		!caseInsensitiveContains(r.To, e.DstIP) && !isIPinCIDR(e.DstIP, r.To, connID) {
		return false
	}

	if r.PortFrom != "ANY" && !caseInsensitiveContains(r.PortFrom, strconv.Itoa(e.SrcPort)) {
		return false
	}
	if r.PortTo != "ANY" && !caseInsensitiveContains(r.PortTo, strconv.Itoa(e.DstPort)) {
		return false
	}

	// PluginID, PluginSID, SrcIP, DstIP, SrcPort, DstPort all match
	return true
}

func initDirectives() error {
	p := path.Join(progDir, confDir, directiveFileGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}

	for i := range files {
		var d directives
		if !fileExist(files[i]) {
			return errors.New("Cannot find " + files[i])
		}
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := ioutil.ReadAll(file)
		err = json.Unmarshal(byteValue, &d)
		if err != nil {
			return err
		}
		for j := range d.Directives {
			uCases.Directives = append(uCases.Directives, d.Directives[j])
		}
	}

	total := len(uCases.Directives)
	if total == 0 {
		return errors.New("cannot find any directive to load from conf dir")
	}
	logInfo("Loaded "+strconv.Itoa(total)+" directives.", 0)

	/*
		for i := range uCases.Directives {
			printDirective(uCases.Directives[i])
		}
	*/

	var dirchan []chan normalizedEvent
	eventChannel = make(chan normalizedEvent)

	for i := 0; i < total; i++ {
		dirchan = append(dirchan, make(chan normalizedEvent))
		go startDirective(uCases.Directives[i], dirchan[i])

		// copy incoming events to all directive channels
		go func() {
			for {
				evt := <-eventChannel
				for i := range dirchan {
					dirchan[i] <- evt
				}
			}
		}()
	}
	return nil
}

func printDirective(d directive) {
	logInfo("Directive ID: "+strconv.Itoa(d.ID)+" Name: "+d.Name, 0)
	for j := 0; j < len(d.Rules); j++ {
		logInfo("- Rule "+strconv.Itoa(d.Rules[j].Stage)+" Name: "+d.Rules[j].Name+
			" From: "+d.Rules[j].From+" To: "+d.Rules[j].To, 0)
	}
}
