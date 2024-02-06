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
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime/trace"
	"sync"
	"time"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/expcounter"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/server"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/worker"
	xc "github.com/defenxor/dsiem/internal/pkg/dsiem/xcorrelator"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	progName    = "dsiem"
	aEventsLogs = "siem_alarm_events.json"
	alarmLogs   = "siem_alarms.json"
)

var version string
var buildTime string
var eventChan chan event.NormalizedEvent
var bpChan chan bool

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.PersistentFlags().Bool("dev", false, "Enable development environment specific setting")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug messages for tracing and troubleshooting")
	serverCmd.Flags().StringP("address", "a", "0.0.0.0", "IP address for the HTTP server to listen on")
	serverCmd.Flags().IntP("port", "p", 8080, "TCP port for the HTTP server to listen on")
	serverCmd.Flags().IntP("maxDelay", "d", 180, "Max. processing delay in seconds before throttling incoming events (under-pressure condition), 0 means disabled")
	serverCmd.Flags().IntP("maxQueue", "q", 25000, "Length of queue for directive evaluation process, 0 means unlimited/unbounded")
	serverCmd.Flags().IntP("maxEPS", "e", 1000, "Max. number of incoming events/second")
	serverCmd.Flags().IntP("minEPS", "i", 100, "Min. events/second rate allowed when throttling incoming events")
	serverCmd.Flags().IntP("minAlarmLifetime", "l", 0,
		"Min. alarm lifetime in minutes. Backlog won't expire sooner than this regardless rule timeouts. This is to support processing of delayed events")
	serverCmd.Flags().IntP("holdDuration", "n", 10, "Duration in seconds before resetting overload condition state")
	serverCmd.Flags().Bool("apm", false, "Enable elastic APM instrumentation")
	serverCmd.Flags().Bool("writeableConfig", false, "Whether to allow configuration file update through HTTP")
	serverCmd.Flags().Bool("pprof", false, "Enable go pprof on the web interface")
	serverCmd.Flags().Bool("websocket", false, "Enable websocket endpoint that streams events/second measurement data")
	serverCmd.Flags().Bool("trace", false, "Generate 10 seconds trace file for debugging.")
	serverCmd.Flags().Bool("intelPrivateIP", false, "Whether to check private IP addresses against threat intel")
	serverCmd.Flags().StringP("mode", "m", "standalone", "Deployment mode, can be set to standalone, cluster-frontend, or cluster-backend")
	serverCmd.Flags().IntP("cacheDuration", "c", 10, "Cache expiration time in minutes for intel and vuln query results")
	serverCmd.Flags().String("msq", "nats://dsiem-nats:4222", "Nats address to use for frontend - backend communication")
	serverCmd.Flags().String("frontend", "", "Frontend URL to pull configuration from, e.g. http://frontend:8080 (used only by backends).")
	serverCmd.Flags().String("node", "", "Unique node name to use when deployed in cluster mode.")
	serverCmd.Flags().StringSliceP("tags", "t", []string{"Identified Threat", "False Positive", "Valid Threat", "Security Incident"},
		"Alarm tags to use, the first one will be assigned to new alarms")
	serverCmd.Flags().Int("medRiskMin", 3,
		"Minimum alarm risk value to be classified as Medium risk. Lower value than this will be classified as Low risk")
	serverCmd.Flags().Int("medRiskMax", 6,
		"Maximum alarm risk value to be classified as Medium risk. Higher value than this will be classified as High risk")
	serverCmd.Flags().StringSliceP("status", "s", []string{"Open", "In-Progress", "Closed"},
		"Alarm status to use, the first one will be assigned to new alarms")
	validateCmd.Flags().StringP("filePattern", "f", "directives_*.json", "Directive file pattern glob to validate")
	viper.BindPFlag("dev", rootCmd.PersistentFlags().Lookup("dev"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("address", serverCmd.Flags().Lookup("address"))
	viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))
	viper.BindPFlag("maxDelay", serverCmd.Flags().Lookup("maxDelay"))
	viper.BindPFlag("maxQueue", serverCmd.Flags().Lookup("maxQueue"))
	viper.BindPFlag("maxEPS", serverCmd.Flags().Lookup("maxEPS"))
	viper.BindPFlag("minEPS", serverCmd.Flags().Lookup("minEPS"))
	viper.BindPFlag("minAlarmLifetime", serverCmd.Flags().Lookup("minAlarmLifetime"))
	viper.BindPFlag("holdDuration", serverCmd.Flags().Lookup("holdDuration"))
	viper.BindPFlag("cacheDuration", serverCmd.Flags().Lookup("cacheDuration"))
	viper.BindPFlag("apm", serverCmd.Flags().Lookup("apm"))
	viper.BindPFlag("pprof", serverCmd.Flags().Lookup("pprof"))
	viper.BindPFlag("trace", serverCmd.Flags().Lookup("trace"))
	viper.BindPFlag("intelPrivateIP", serverCmd.Flags().Lookup("intelPrivateIP"))
	viper.BindPFlag("mode", serverCmd.Flags().Lookup("mode"))
	viper.BindPFlag("msq", serverCmd.Flags().Lookup("msq"))
	viper.BindPFlag("node", serverCmd.Flags().Lookup("node"))
	viper.BindPFlag("frontend", serverCmd.Flags().Lookup("frontend"))
	viper.BindPFlag("tags", serverCmd.Flags().Lookup("tags"))
	viper.BindPFlag("status", serverCmd.Flags().Lookup("status"))
	viper.BindPFlag("medRiskMin", serverCmd.Flags().Lookup("medRiskMin"))
	viper.BindPFlag("medRiskMax", serverCmd.Flags().Lookup("medRiskMax"))
	viper.BindPFlag("filePattern", validateCmd.Flags().Lookup("filePattern"))
	viper.BindPFlag("writeableConfig", serverCmd.Flags().Lookup("writeableConfig"))
	viper.BindPFlag("websocket", serverCmd.Flags().Lookup("websocket"))
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
	fmt.Println(msg+":", err)
	os.Exit(1)
}

var rootCmd = &cobra.Command{
	Use:   "dsiem",
	Short: "SIEM for ELK stack",
	Long: `
DSiem is an event correlation engine for ELK stack.

DSiem provides OSSIM-style correlation for normalized logs/events, and relies on 
Filebeat, Logstash, and Elasticsearch to do the rest.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long:  `Print the version and build information`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version, buildTime)
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate directive files",
	Long:  `Test loading and parsing directives from configs directory`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Setup(viper.GetBool("debug"))
		pattern := viper.GetString("filePattern")
		d, err := fs.GetDir(viper.GetBool("dev"))
		if err != nil {
			exit("Cannot get current directory??", err)
		}
		confDir := path.Join(d, "configs")
		res, count, err := siem.LoadDirectivesFromFile(confDir, pattern, false)
		if err != nil {
			exit("Error occur", err)
		} else {
			fmt.Printf("found %d valid entries out of %d directive(s) in configs/%s\n", len(res.Dirs), count, pattern)
		}
	},
}

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",
	Long: `
Start dsiem server in a standalone or clustered deployment mode (either as frontend or backend).

Frontends listen for normalized events from logstash and distribute them to backends through NATS message queue.
Frontends also serve incoming request for configuration management from web UI.

Backends receive events on the message queue channel, perform correlation based on configured directive rules, 
and then send results/alarms to elasticsearch through local filebeat.

Standalone mode perform both frontend and backend functions in a single dsiem instance directly, without the need for
external message queue.`,

	Run: func(cmd *cobra.Command, args []string) {

		d, err := fs.GetDir(viper.GetBool("dev"))
		if err != nil {
			exit("Cannot get current directory??", err)
		}

		confDir := path.Join(d, "configs")
		logDir := path.Join(d, "logs")
		webDir := path.Join(d, "web", "dist")
		addr := viper.GetString("address")
		port := viper.GetInt("port")
		pprof := viper.GetBool("pprof")
		mode := viper.GetString("mode")
		msq := viper.GetString("msq")
		node := viper.GetString("node")
		traceFlag := viper.GetBool("trace")
		intelPrivIPFlag := viper.GetBool("intelPrivateIP")
		frontend := viper.GetString("frontend")
		maxQueue := viper.GetInt("maxQueue")
		maxEPS := viper.GetInt("maxEPS")
		minEPS := viper.GetInt("minEPS")
		holdDuration := viper.GetInt("holdDuration")
		cacheDuration := viper.GetInt("cacheDuration")
		esapm := viper.GetBool("apm")
		writeableConfig := viper.GetBool("writeableConfig")
		websocket := viper.GetBool("websocket")
		minAlarmLifetime := viper.GetInt("minAlarmLifetime")

		if err := checkMode(mode, msq, node, frontend); err != nil {
			exit("Incorrect mode configuration", err)
		}

		if minEPS > maxEPS {
			exit("Incorrect EPS setting", errors.New("minEPS must be <= than maxEPS"))
		}

		if traceFlag {
			fo, err := os.CreateTemp(os.TempDir(), progName+"*.trace")
			if err != nil {
				exit("Cannot create temp file for tracer", err)
			}
			defer fo.Close()
			wrt := bufio.NewWriter(fo)
			trace.Start(wrt)
			t := time.NewTimer(10 * time.Second)
			go func() {
				<-t.C
				trace.Stop()
				t.Stop()
				fmt.Println("Done writing trace file.")
			}()
		}

		apm.Enable(esapm)

		// make sure status and tags from env var is recognized as slices
		viper.Set("status", viper.GetStringSlice("status"))
		viper.Set("tags", viper.GetStringSlice("tags"))
		// saving the config for UI to read. /config dir maybe read-only though, so
		// just put out warning on failure
		err = viper.WriteConfigAs(path.Join(confDir, progName+"_config.json"))
		if err != nil {
			log.Warn(log.M{Msg: "Cannot write startup info file to " + confDir +
				". Web UI will not be able to use that info: " + err.Error()})
		}

		eventChan = make(chan event.NormalizedEvent)
		bpChan = make(chan bool)
		var sendBpChan chan<- bool

		log.Setup(viper.GetBool("debug"))
		log.Info(log.M{Msg: "Starting " + progName + " " + version +
			" in " + mode + " mode."})

		if mode == "cluster-backend" {
			if err := worker.Start(eventChan, msq,
				progName, node, confDir, frontend); err != nil {
				exit("Cannot start backend worker process", err)
			}
			sendBpChan = worker.GetBackPressureChannel()
		} else {
			sendBpChan = bpChan
		}

		if mode != "cluster-frontend" {
			err = asset.Init(confDir)
			if err != nil {
				exit("Cannot initialize assets from "+confDir, err)
			}
			err = xc.InitIntel(confDir, cacheDuration)
			if err != nil {
				exit("Cannot initialize threat intel", err)
			}
			err = xc.InitVuln(confDir, cacheDuration)
			if err != nil {
				exit("Cannot initialize Vulnerability scan result", err)
			}
			err = siem.InitDirectives(confDir, eventChan, minAlarmLifetime, maxEPS, maxQueue)
			if err != nil {
				exit("Cannot initialize directives", err)
			}
			err = alarm.Init(path.Join(logDir, alarmLogs), intelPrivIPFlag)
			if err != nil {
				exit("Cannot initialize alarm", err)
			}
			err = siem.InitBackLogManager(path.Join(logDir, aEventsLogs),
				sendBpChan, holdDuration)
			if err != nil {
				exit("Cannot initialize backlog manager", err)
			}
		}

		expcounter.Init(mode)

		cf := server.Config{}
		cf.EvtChan = eventChan
		cf.BpChan = bpChan
		cf.Confd = confDir
		cf.Webd = webDir
		cf.WriteableConfig = writeableConfig
		cf.Pprof = pprof
		cf.Mode = mode
		cf.MaxEPS = maxEPS
		cf.MinEPS = minEPS
		cf.MsqCluster = msq
		cf.MsqPrefix = progName
		cf.NodeName = node
		cf.Addr = addr
		cf.Port = port
		cf.WebSocket = websocket

		err = server.Start(cf)
		if err != nil {
			exit("Cannot start server", err)
		}

		waitInterruptSignal()
	},
}

func checkMode(mode, node, msq, frontend string) error {
	if mode != "standalone" &&
		mode != "cluster-frontend" &&
		mode != "cluster-backend" {
		return errors.New("mode must be standalone || cluster-frontend || cluster-backend")
	}
	if mode == "cluster-frontend" {
		if msq == "" || node == "" {
			return errors.New("mode cluster-frontend requires msq and node options")
		}
	}
	if mode == "cluster-backend" {
		if frontend == "" || msq == "" || node == "" {
			return errors.New("mode cluster-backend requires msq, node, and frontend options")
		}
	}
	return nil
}

func waitInterruptSignal() {
	var wg sync.WaitGroup
	wg.Add(1)
	var ch chan os.Signal
	ch = make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		wg.Done()
	}()
	wg.Wait()
}
