package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime/trace"
	"time"

	"dsiem/internal/dsiem/pkg/alarm"
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/dsiem/pkg/server"
	"dsiem/internal/dsiem/pkg/siem"
	"dsiem/internal/dsiem/pkg/worker"
	xc "dsiem/internal/dsiem/pkg/xcorrelator"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"dsiem/internal/shared/pkg/pprof"

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
	serverCmd.Flags().IntP("maxDelay", "d", 180, "Max. processing delay in seconds before throttling iconming events")
	serverCmd.Flags().IntP("maxEPS", "e", 1000, "Max. number of incoming events/second")
	serverCmd.Flags().IntP("minEPS", "i", 100, "Min. events/second rate allowed when throttling incoming events")
	serverCmd.Flags().IntP("holdDuration", "n", 10, "Duration in seconds before resetting overload condition state")
	serverCmd.Flags().Bool("apm", true, "Enable elastic APM instrumentation")
	serverCmd.Flags().String("pprof", "", "Generate performance profiling information for either cpu, mutex, memory, or block.")
	serverCmd.Flags().Bool("trace", false, "Generate trace file for debugging.")
	serverCmd.Flags().StringP("mode", "m", "standalone", "Deployment mode, can be set to standalone, cluster-frontend, or cluster-backend")
	serverCmd.Flags().String("msqUrl", "nats://dsiem-nats:4222", "Nats-streaming URL to use for frontend - backend communication.")
	serverCmd.Flags().String("msq", "test-cluster", "Nats-streaming cluster name to use for frontend - backend communication.")
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
	viper.BindPFlag("maxEPS", serverCmd.Flags().Lookup("maxEPS"))
	viper.BindPFlag("minEPS", serverCmd.Flags().Lookup("minEPS"))
	viper.BindPFlag("holdDuration", serverCmd.Flags().Lookup("holdDuration"))
	viper.BindPFlag("apm", serverCmd.Flags().Lookup("apm"))
	viper.BindPFlag("pprof", serverCmd.Flags().Lookup("pprof"))
	viper.BindPFlag("trace", serverCmd.Flags().Lookup("trace"))
	viper.BindPFlag("mode", serverCmd.Flags().Lookup("mode"))
	viper.BindPFlag("msqUrl", serverCmd.Flags().Lookup("msqUrl"))
	viper.BindPFlag("msq", serverCmd.Flags().Lookup("msq"))
	viper.BindPFlag("node", serverCmd.Flags().Lookup("node"))
	viper.BindPFlag("frontend", serverCmd.Flags().Lookup("frontend"))
	viper.BindPFlag("tags", serverCmd.Flags().Lookup("tags"))
	viper.BindPFlag("status", serverCmd.Flags().Lookup("status"))
	viper.BindPFlag("medRiskMin", serverCmd.Flags().Lookup("medRiskMin"))
	viper.BindPFlag("medRiskMax", serverCmd.Flags().Lookup("medRiskMax"))
	viper.BindPFlag("filePattern", validateCmd.Flags().Lookup("filePattern"))
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
	Long:  `Test loading and parsing directives from specified configuration files`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Setup(viper.GetBool("debug"))
		pattern := viper.GetString("filePattern")
		d, err := fs.GetDir(viper.GetBool("dev"))
		if err != nil {
			exit("Cannot get current directory??", err)
		}
		confDir := path.Join(d, "configs")
		_, _, err = siem.LoadDirectivesFromFile(confDir, pattern)
		if err != nil {
			exit("Error occur", err)
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
		pp := viper.GetString("pprof")
		mode := viper.GetString("mode")
		msqURL := viper.GetString("msqUrl")
		msq := viper.GetString("msq")
		node := viper.GetString("node")
		traceFlag := viper.GetBool("trace")
		frontend := viper.GetString("frontend")
		maxEPS := viper.GetInt("maxEPS")
		minEPS := viper.GetInt("minEPS")
		holdDuration := viper.GetInt("holdDuration")

		if err := checkMode(mode, msq, node, frontend); err != nil {
			exit("Incorrect mode configuration", err)
		}

		if minEPS > maxEPS {
			exit("Incorrect EPS setting", errors.New("minEPS must be <= than maxEPS"))
		}

		if pp != "" {
			f, err := pprof.GetProfiler(pp)
			if err != nil {
				exit("Cannot start profiler", err)
			}
			defer f.Stop()
		}

		if traceFlag {
			fo, err := ioutil.TempFile(os.TempDir(), progName+"*.trace")
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

		// saving the config for UI to read
		err = viper.WriteConfigAs(path.Join(confDir, progName+"_config.json"))
		if err != nil {
			exit("Error writing config file", err)
		}

		eventChan = make(chan event.NormalizedEvent)
		bpChan = make(chan bool)
		var sendBpChan chan<- bool

		log.Setup(viper.GetBool("debug"))
		log.Info(log.M{Msg: "Starting " + progName + " " + versionCmd.Version +
			" in " + mode + " mode."})

		if mode == "cluster-backend" {
			if err := worker.Start(eventChan, msqURL,
				msq, progName, node, confDir, frontend); err != nil {
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
			err = xc.InitIntel(confDir)
			if err != nil {
				exit("Cannot initialize threat intel", err)
			}
			err = xc.InitVuln(confDir)
			if err != nil {
				exit("Cannot initialize Vulnerability scan result", err)
			}
			err = siem.InitDirectives(confDir, eventChan)
			if err != nil {
				exit("Cannot initialize directives", err)
			}
			err = siem.InitBackLog(path.Join(logDir, aEventsLogs),
				sendBpChan, holdDuration)
			if err != nil {
				exit("Cannot initialize backlog", err)
			}
			err = alarm.Init(path.Join(logDir, alarmLogs))
			if err != nil {
				exit("Cannot initialize alarm", err)
			}
		}

		err = server.Start(
			eventChan, bpChan, confDir, webDir,
			mode, maxEPS, minEPS, msqURL, msq, progName, node, addr, port)
		if err != nil {
			exit("Cannot start server", err)
		}

	},
}

func checkMode(mode, msq, node, frontend string) error {
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
