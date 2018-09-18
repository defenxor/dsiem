package main

import (
	"fmt"
	"os"
	"path"

	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/dsiem/pkg/server"
	"dsiem/internal/dsiem/pkg/siem"
	xc "dsiem/internal/dsiem/pkg/xcorrelator"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"

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
var eventChannel chan event.NormalizedEvent

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringP("address", "a", "0.0.0.0", "IP address to listen on")
	serverCmd.Flags().IntP("port", "p", 8080, "TCP port to listen on")
	serverCmd.Flags().Bool("dev", false, "Enable development environment specific setting")
	serverCmd.Flags().Bool("debug", false, "Enable debug messages for tracing and troubleshooting")
	viper.BindPFlag("address", serverCmd.Flags().Lookup("address"))
	viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))
	viper.BindPFlag("dev", serverCmd.Flags().Lookup("dev"))
	viper.BindPFlag("debug", serverCmd.Flags().Lookup("debug"))
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
	if viper.GetBool("debug") {
		fmt.Println(msg)
		panic(err)
	} else {
		fmt.Println("Exiting: " + msg + ": " + err.Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dsiem",
	Short: "SIEM for ELK stack",
	Long: `
DSiem is a security event correlation engine for ELK stack.
Provides OSSIM-style event correlation, and relies on 
Filebeat, Logstash, and Elasticsearch to do the rest.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long:  `Print the version and build date information`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version, buildTime)
	},
}

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",
	Long: `
Start server listening on /events for event sent 
from logstash, and on /config for configuration read/write from UI`,
	Run: func(cmd *cobra.Command, args []string) {

		d, err := fs.GetDir(viper.GetBool("dev"))
		if err != nil {
			exit("Cannot get current directory??", err)
		}
		confDir := path.Join(d, "configs")
		logDir := path.Join(d, "logs")
		addr := viper.GetString("address")
		port := viper.GetInt("port")

		eventChannel = make(chan event.NormalizedEvent)

		log.Setup(viper.GetBool("debug"))
		log.Info("Starting "+progName+" "+versionCmd.Version, 0)

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
		err = siem.InitDirectives(confDir, eventChannel)
		if err != nil {
			exit("Cannot initialize directives", err)
		}
		err = siem.InitBackLog(path.Join(logDir, aEventsLogs))
		if err != nil {
			exit("Cannot initialize backlog", err)
		}
		err = siem.InitAlarm(path.Join(logDir, alarmLogs))
		if err != nil {
			exit("Cannot initialize alarm", err)
		}
		err = server.Start(eventChannel, confDir, addr, port)
		if err != nil {
			exit("Cannot start server", err)
		}

	},
}
