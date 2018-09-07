package main

import (
	"strconv"
	"time"

	"github.com/teris-io/shortid"
)

type backLog struct {
	ID           string
	Risk         int
	CurrentStage int
	HighestStage int
	Directive    directive
	BEvents      bLogEvents
}

var bLogs backLogs
var sid *shortid.Shortid
var ticker *time.Ticker

type backLogs struct {
	BackLogs []backLog
}

type bLogEvent struct {
	RuleNo  int
	EventID string
}

type bLogEvents struct {
	BLogEvents []bLogEvent
}

func initShortID() {
	sid, _ = shortid.New(1, shortid.DefaultABC, 2342)
}

// this checks for timed-out backlog and discard it
func startBackLogTicker() {
	ticker = time.NewTicker(time.Second * 10)
	go func() {
		for {
			<-ticker.C
			now := time.Now().Unix()
			for i := range bLogs.BackLogs {
				cs := bLogs.BackLogs[i].CurrentStage
				idx := cs - 1
				start := bLogs.BackLogs[i].Directive.Rules[idx].StartTime
				timeout := bLogs.BackLogs[i].Directive.Rules[idx].Timeout
				maxTime := start + timeout
				if maxTime > now {
					continue
				}
				logger.Info("directive " + strconv.Itoa(bLogs.BackLogs[i].Directive.ID) + " backlog " + bLogs.BackLogs[i].ID + " expired. Deleting it.")
			}
		}
	}()
}

func backlogManager(e normalizedEvent, d directive) {

	found := false
	for i := range bLogs.BackLogs {
		cs := bLogs.BackLogs[i].CurrentStage
		// only applicable for non-stage 1, where there's more specific identifier like IP address to match
		if bLogs.BackLogs[i].Directive.ID == d.ID && cs > 1 {
			// should check for currentStage rule match with event
			// heuristic, we know stage starts at 1 but rules start at 0
			idx := cs - 1
			currRule := bLogs.BackLogs[i].Directive.Rules[idx]
			if doesEventMatchRule(e, currRule) {
				logInfo("Directive "+strconv.Itoa(d.ID)+" backlog "+bLogs.BackLogs[i].ID+" matched. Not creating new backlog.", e.ConnID)
				bLogs.BackLogs[i].processMatchedEvent(e, idx)
				found = true
			}
		}
	}

	if found {
		return
	}

	// create new backlog here, passing the event as the 1st event for the backlog
	bid, _ := sid.Generate()
	logInfo("Directive "+strconv.Itoa(d.ID)+" created new backlog "+bid, e.ConnID)
	b := backLog{}
	b.ID = bid
	b.Directive = directive{}

	copyDirective(&b.Directive, &d)
	initBackLogRules(b.Directive, e)
	b.Directive.Rules[0].StartTime = time.Now().Unix()

	b.CurrentStage = 1
	b.HighestStage = len(d.Rules)
	b.processMatchedEvent(e, 0)
	bLogs.BackLogs = append(bLogs.BackLogs, b)
}

func copyDirective(dst *directive, src *directive) {
	dst.ID = src.ID
	dst.Priority = src.Priority
	dst.Name = src.Name
	for i := range src.Rules {
		r := src.Rules[i]
		dst.Rules = append(dst.Rules, r)
	}
}

func initBackLogRules(d directive, e normalizedEvent) {
	// need to copy the directiveRules here, changing 1:TO etc to actual IP address.
	for i := range d.Rules {

		// the first rule cannot use reference to other
		if i == 0 {
			continue
		}

		// for the rest, refer to the referenced stage if its not ANY or HOME_NET or !HOME_NET
		// if the reference is ANY || HOME_NET || !HOME_NET then refer to event
		r := d.Rules[i].From
		v, err := reftoDigit(r)
		if err == nil {
			vmin1 := v - 1
			ref := d.Rules[vmin1].From
			if ref != "ANY" && ref != "HOME_NET" && ref != "!HOME_NET" {
				d.Rules[i].From = ref
			} else {
				d.Rules[i].From = e.SrcIP
			}
		}
		r = d.Rules[i].To
		v, err = reftoDigit(r)
		if err == nil {
			vmin1 := v - 1
			ref := d.Rules[vmin1].To
			if ref != "ANY" && ref != "HOME_NET" && ref != "!HOME_NET" {
				d.Rules[i].To = ref
			} else {
				d.Rules[i].To = e.DstIP
			}
		}
		r = d.Rules[i].PortFrom
		v, err = reftoDigit(r)
		if err == nil {
			vmin1 := v - 1
			ref := d.Rules[vmin1].PortFrom
			if ref != "ANY" {
				d.Rules[i].PortFrom = ref
			} else {
				d.Rules[i].PortFrom = strconv.Itoa(e.SrcPort)
			}
		}
		r = d.Rules[i].PortTo
		v, err = reftoDigit(r)
		if err == nil {
			vmin1 := v - 1
			ref := d.Rules[vmin1].PortTo
			if ref != "ANY" {
				d.Rules[i].PortTo = ref
			} else {
				d.Rules[i].PortTo = strconv.Itoa(e.DstPort)
			}
		}
	}
}

func (b *backLog) calcStage() {
	s := b.CurrentStage
	// heuristic, we know stage starts at 1 but rules start at 0
	rIdx := s - 1
	currRule := b.Directive.Rules[rIdx]
	currEvents := b.BEvents.BLogEvents
	if currEvents != nil {

	}
	nEvents := len(b.BEvents.BLogEvents)
	if nEvents >= currRule.Occurrence && b.CurrentStage < b.HighestStage {
		b.CurrentStage++
		logger.Info("directive " + strconv.Itoa(b.Directive.ID) + " backlog " + b.ID + " increased stage to " + strconv.Itoa(b.CurrentStage))
		idx := b.CurrentStage - 1
		b.Directive.Rules[idx].StartTime = time.Now().Unix()
		b.calcRisk()
	}
}

func (b *backLog) calcRisk() {

	s := b.CurrentStage
	idx := s - 1
	from := b.Directive.Rules[idx].From
	to := b.Directive.Rules[idx].To
	value := getAssetValue(from)
	tval := getAssetValue(to)
	if tval > value {
		value = tval
	}

	reliability := b.Directive.Rules[idx].Reliability
	priority := b.Directive.Priority

	risk := priority * reliability * value / 25
	cRisk := b.Risk
	if risk != cRisk {
		logger.Info("directive " + strconv.Itoa(b.Directive.ID) + " backlog " + b.ID + " risk increased from " + strconv.Itoa(cRisk) + " to " + strconv.Itoa(risk))
	}
	b.Risk = risk
}

func (b *backLog) processMatchedEvent(e normalizedEvent, idx int) {
	be := bLogEvent{}
	be.RuleNo = idx
	be.EventID = e.EventID
	b.BEvents.BLogEvents = append(b.BEvents.BLogEvents, be)
	b.calcStage()
}
