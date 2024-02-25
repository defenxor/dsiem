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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
	"github.com/defenxor/dsiem/internal/pkg/shared/ip"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"github.com/remeh/sizedwaitgroup"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	progName = "dtester"
)

var version string
var buildTime string

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(dsiemCmd)
	rootCmd.AddCommand(fbeatCmd)
	rootCmd.PersistentFlags().StringP("file", "f", "directives_*.json", "file glob pattern to load directives from")
	rootCmd.PersistentFlags().IntP("max", "n", 1000, "Maximum number of events to send per rule")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "print sent events to console")

	dsiemCmd.Flags().StringP("address", "a", "127.0.0.1", "Dsiem IP address to send events to")
	dsiemCmd.Flags().IntP("port", "p", 8080, "Dsiem TCP port")
	dsiemCmd.Flags().StringP("homenet", "i", "192.168.0.1", "IP address to use to represent HOME_NET. This IP must already be defined in dsiem assets configuration")
	dsiemCmd.Flags().StringP("srcip", "s", "", "Fixed source IP address to use for all events, put a high value asset here to quickly raise alarms")
	dsiemCmd.Flags().StringP("destip", "d", "", "Fixed destination IP address to use for all events, put a high value asset here to quickly raise alarms")
	dsiemCmd.Flags().IntP("rps", "r", 500, "number of HTTP post request per second")
	dsiemCmd.Flags().IntP("concurrency", "c", 50, "number of concurrent HTTP post to submit")

	fbeatCmd.Flags().StringP("logfile", "l", "/var/log/external/dtester.json", "log file location for filebeat mode. Filebeat must be configured to harvest this file.")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("file", rootCmd.PersistentFlags().Lookup("file"))
	viper.BindPFlag("max", rootCmd.PersistentFlags().Lookup("max"))

	viper.BindPFlag("concurrency", dsiemCmd.Flags().Lookup("concurrency"))
	viper.BindPFlag("address", dsiemCmd.Flags().Lookup("address"))
	viper.BindPFlag("port", dsiemCmd.Flags().Lookup("port"))
	viper.BindPFlag("homenet", dsiemCmd.Flags().Lookup("homenet"))
	viper.BindPFlag("srcip", dsiemCmd.Flags().Lookup("srcip"))
	viper.BindPFlag("destip", dsiemCmd.Flags().Lookup("destip"))
	viper.BindPFlag("rps", dsiemCmd.Flags().Lookup("rps"))

	viper.BindPFlag("logfile", fbeatCmd.Flags().Lookup("logfile"))
}

func initConfig() {
	viper.SetEnvPrefix(progName)
	viper.AutomaticEnv()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		exit("Error returned from command", err)
	}
}

func exit(msg string, err error) {
	fmt.Println("Exiting: " + msg + ": " + err.Error())
	os.Exit(1)
}

var rootCmd = &cobra.Command{
	Use:   "dtester",
	Short: "Directive rules tester for Dsiem",
	Long: `
Dtester test directive rules by sending a simulated matching event to dsiem,
either directly, or through filebeat and logstash`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long:  `Print the version and build date information`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version, buildTime)
	},
}

var dsiemCmd = &cobra.Command{
	Use:   "dsiem",
	Short: "Start sending events to dsiem",
	Long:  `Send events to dsiem at defined address and port`,
	Run: func(cmd *cobra.Command, args []string) {

		log.Setup(true)

		addr := viper.GetString("address")
		port := viper.GetInt("port")
		file := viper.GetString("file")

		dirs, _, err := siem.LoadDirectivesFromFile("", file, false)
		if err != nil {
			exit("Cannot initialize directives", err)
		}
		sender(&dirs, addr, port)
	},
}

var fbeatCmd = &cobra.Command{
	Use:   "fbeat",
	Short: "Start sending events through filebeat",
	Long:  `Send events to dsiem through filebeat and logstash`,
	Run: func(cmd *cobra.Command, args []string) {

		log.Setup(true)

		logfile := viper.GetString("logfile")
		file := viper.GetString("file")

		dirs, _, err := siem.LoadDirectivesFromFile("", file, false)
		if err != nil {
			exit("Cannot initialize directives", err)
		}
		toFilebeat(&dirs, logfile)
	},
}

func toFilebeat(d *siem.Directives, logfile string) {
	max := viper.GetInt("max")
	verbose := viper.GetBool("verbose")
	swg := sizedwaitgroup.New(10)
	srcip := viper.GetString("srcip")
	destip := viper.GetString("destip")

	for _, v := range d.Dirs {
		var prevPortTo int
		var prevPortFrom int
		var prevFrom string
		var prevTo string
		for _, j := range v.Rules {
			amt := j.Occurrence
			if amt > max {
				amt = max
			}
			e := event.NormalizedEvent{}
			e.Sensor = progName
			e.Title = j.Name
			e.SrcPort = genPort(j.PortFrom, prevPortFrom, false)
			e.DstPort = genPort(j.PortTo, prevPortTo, true)
			e.Protocol = genProto(j.Protocol)
			e.PluginID = j.PluginID
			e.PluginSID = pickOneFromIntSlice(j.PluginSID)
			e.Product = pickOneFromStrSlice(j.Product)
			e.Category = j.Category
			e.SubCategory = pickOneFromStrSlice(j.SubCategory)

			if destip == "" {
				e.DstIP = genIP(j.To, prevTo, e.SrcIP)
			} else {
				e.DstIP = destip
			}

			if srcip == "" {
				e.SrcIP = genIP(j.From, prevFrom, "")
			} else {
				e.SrcIP = srcip
			}

			prevPortTo = e.DstPort
			prevPortFrom = e.SrcPort
			prevFrom = e.SrcIP
			prevTo = e.DstIP

			for i := 0; i < amt; i++ {
				for {
					//	err := fn(&e, c, st, iter, verbose)
					err := savetoLog(e, logfile, verbose)
					if err != nil {
						// exit if error
						exit("Cannot save to "+logfile, err)
					}
					break
				}
			}
		}
	}
	swg.Wait()
}

func sender(d *siem.Directives, addr string, port int) {
	max := viper.GetInt("max")
	// conc := viper.GetInt("concurrency")
	rps := viper.GetInt("rps")
	conc := viper.GetInt("concurrency")
	verbose := viper.GetBool("verbose")
	srcip := viper.GetString("srcip")
	destip := viper.GetString("destip")

	keepAliveTimeout := 600 * time.Second
	timeout := 5 * time.Second

	defaultTransport := &http.Transport{
		Dial:                (&net.Dialer{KeepAlive: keepAliveTimeout}).Dial,
		MaxIdleConns:        conc + 100,
		MaxIdleConnsPerHost: conc + 100,
	}
	c := &http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
	}
	swg := sizedwaitgroup.New(conc)
	fn := rateLimit(rps, rps, timeout, sendHTTPSingleConn)

	for _, v := range d.Dirs {
		var prevPortTo int
		var prevPortFrom int
		var prevFrom string
		var prevTo string
		for _, j := range v.Rules {
			amt := j.Occurrence
			if amt > max {
				amt = max
			}
			e := event.NormalizedEvent{}
			e.Title = "Dtester event"
			e.Sensor = progName
			e.SrcPort = genPort(j.PortFrom, prevPortFrom, false)
			e.DstPort = genPort(j.PortTo, prevPortTo, true)
			e.Protocol = genProto(j.Protocol)
			e.PluginID = j.PluginID
			e.PluginSID = pickOneFromIntSlice(j.PluginSID)
			e.Product = pickOneFromStrSlice(j.Product)
			e.Category = j.Category
			e.SubCategory = pickOneFromStrSlice(j.SubCategory)

			if destip == "" {
				e.DstIP = genIP(j.To, prevTo, e.SrcIP)
			} else {
				e.DstIP = destip
			}

			if srcip == "" {
				e.SrcIP = genIP(j.From, prevFrom, "")
			} else {
				e.SrcIP = srcip
			}

			prevPortTo = e.DstPort
			prevPortFrom = e.SrcPort
			prevFrom = e.SrcIP
			prevTo = e.DstIP

			for i := 0; i < amt; i++ {
				swg.Add()
				go func(st int, iter int) {
					defer swg.Done()
					for {
						//	err := fn(&e, c, st, iter, verbose)
						err := fn(&e, c, j.Stage, iter, verbose)
						if err != nil {
							log.Info(log.M{Msg: "Error: " + err.Error() + ". Retrying in 3 second."})
							time.Sleep(3 * time.Second)
							continue
						}
						break
					}
				}(j.Stage, i)
			}
		}
	}
	swg.Wait()
}

type eventPoster func(e *event.NormalizedEvent, c *http.Client, stage int, iter int, verbose bool) error

func rateLimit(rps, burst int, wait time.Duration, h eventPoster) eventPoster {
	l := rate.NewLimiter(rate.Limit(rps), burst)

	return func(e *event.NormalizedEvent, c *http.Client, stage int, iter int, verbose bool) error {
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel()
		if err := l.Wait(ctx); err != nil {
			return err
		}
		return h(e, c, stage, iter, verbose)
	}
}

func sendHTTPSingleConn(e *event.NormalizedEvent, c *http.Client, stage int, iter int, verbose bool) error {
	e.EventID = genUUID()
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	evt := string(b)
	url := "http://" + viper.GetString("address") + ":" + strconv.Itoa(viper.GetInt("port")) + "/events/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body) // read the body to avoid mem leak? internet says we has to do this

	if resp.StatusCode != http.StatusOK {
		return errors.New("Received HTTP " + strconv.Itoa(resp.StatusCode) + " status")
	}

	if verbose {
		fmt.Println("Sent event for stage:", stage, "order #:", iter)
		fmt.Println(evt)
	}
	return nil
}

func randInt(min int, max int) int {
	m := max - min
	if max < min || m == 0 {
		return 0
	}
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(m) + min
}

func genPort(portList string, prev int, useLowPort bool) int {

	if _, ok := str.RefToDigit(portList); ok {
		return prev
	}

	if portList == "ANY" {
		if useLowPort {
			return randInt(20, 1024)
		}
		return randInt(4000, 65535)
	}
	s := pickOneFromCsv(portList)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func genIP(ruleAddr string, prev string, counterpart string) string {
	if ruleAddr == "HOME_NET" {
		return viper.GetString("homenet")
	}

	priv, err := ip.IsPrivateIP(counterpart)
	if counterpart != "" && ruleAddr == "ANY" && err != nil && !priv {
		return viper.GetString("homenet")
	}

	if _, ok := str.RefToDigit(ruleAddr); ok {
		return prev
	}

	// testing, always return prev
	if prev != "" {
		return prev
	}

	var octet [4]int
	for octet[0] == 0 || octet[0] == 10 || octet[0] == 127 || octet[0] == 192 || octet[0] == 172 {
		octet[0] = randInt(1, 254)
	}
	for i := 1; i <= 3; i++ {
		octet[i] = randInt(1, 254)
	}
	sIP := strconv.Itoa(octet[0]) + "." + strconv.Itoa(octet[1]) +
		"." + strconv.Itoa(octet[2]) + "." + strconv.Itoa(octet[3])
	return sIP
}

func genProto(proto string) string {
	if proto == "ANY" {
		return pickOneFromCsv("TCP,UDP")
	}
	return proto
}

func pickOneFromCsv(s string) string {
	sSlice := str.CsvToSlice(s)
	l := len(sSlice)
	return sSlice[randInt(1, l)]
}

func pickOneFromStrSlice(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[randInt(1, l)]
}

func pickOneFromIntSlice(n []int) int {
	l := len(n)
	if l == 0 {
		return 0
	}
	return n[randInt(1, l)]
}

func genUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return "static-id-doesnt-really-matter"
	}
	return u.String()
}

func savetoLog(e event.NormalizedEvent, logfile string, verbose bool) error {
	e.EventID = genUUID()
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	vJSON, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if verbose {
		fmt.Println(vJSON)
	}
	f.SetDeadline(time.Now().Add(60 * time.Second))
	_, err = f.WriteString(string(vJSON) + "\n")
	return err
}
