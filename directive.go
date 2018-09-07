package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
)

const (
	directiveFile = "conf/directives.json"
)

var uCases directives
var eventChannel chan normalizedEvent

func startDirective(d directive, c chan normalizedEvent) {
	logger.Info("Started directive ", d.ID)
	for {
		// handle incoming event
		evt := <-c
		if !doesEventMatchRule(evt, d.Rules[0]) {
			continue
		}

		logInfo("directive "+strconv.Itoa(d.ID)+" found matched event.", evt.ConnID)
		backlogManager(evt, d)
	}
}

func doesEventMatchRule(e normalizedEvent, r directiveRule) bool {
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
	// covers r.From == CIDR network address, or single IP
	if r.From != "HOME_NET" && r.From != "!HOME_NET" && r.From != "ANY" && r.From != e.SrcIP && !isIPinCIDR(e.SrcIP, r.From) {
		return false
	}

	eDstInHomeNet := e.dstIPInHomeNet()
	if r.To == "HOME_NET" && eDstInHomeNet == false {
		return false
	}
	if r.To == "!HOME_NET" && eDstInHomeNet == true {
		return false
	}
	if r.To != "HOME_NET" && r.To != "!HOME_NET" && r.To != "ANY" && r.To != e.DstIP && !isIPinCIDR(e.DstIP, r.To) {
		return false
	}

	if r.PortFrom != "ANY" && r.PortFrom != strconv.Itoa(e.SrcPort) {
		return false
	}
	if r.PortTo != "ANY" && r.PortTo != strconv.Itoa(e.DstPort) {
		return false
	}

	// PluginID, PluginSID, SrcIP, DstIP, SrcPort, DstPort all match
	return true
}

func initDirectives() error {
	dir, err := getDir()
	if err != nil {
		return err
	}
	filename := dir + "/" + directiveFile
	if !fileExist(filename) {
		return errors.New("Cannot find " + filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)
	err = json.Unmarshal(byteValue, &uCases)
	if err != nil {
		return err
	}

	total := len(uCases.Directives)
	logger.Info("Loaded ", strconv.Itoa(total), " directives.")

	for i := range uCases.Directives {
		printDirective(uCases.Directives[i])
	}

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
	logger.Info("Directive ID: " + strconv.Itoa(d.ID) + " Name: " + d.Name)
	for j := 0; j < len(d.Rules); j++ {
		logger.Info("- Rule " + strconv.Itoa(d.Rules[j].Stage) + " Name: " + d.Rules[j].Name +
			" From: " + d.Rules[j].From + " To: " + d.Rules[j].To)
	}
}
