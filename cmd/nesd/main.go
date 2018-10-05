package main

import (
	"dsiem/internal/nesd/pkg/server"
	log "dsiem/internal/shared/pkg/logger"

	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	progName = "nesd"
)

var (
	version   string
	buildTime string
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringP("address", "a", "127.0.0.1", "IP address to listen on")
	serverCmd.Flags().IntP("port", "p", 8081, "TCP port to listen on")
	serverCmd.Flags().StringP("csvdir", "d", "", "directory of Nessus CSV scan results")
	serverCmd.Flags().Bool("debug", false, "Enable debug messages for tracing and troubleshooting")
	viper.BindPFlag("address", serverCmd.Flags().Lookup("address"))
	viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))
	viper.BindPFlag("debug", serverCmd.Flags().Lookup("debug"))
	viper.BindPFlag("csvdir", serverCmd.Flags().Lookup("csvdir"))
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
	Use:   "nesd",
	Short: "Serve nessus CSV result over HTTP",
	Long: `
Serve nessus CSV scan results over HTTP. 
To be used by dsiem as source for vulnerability scan lookup`,
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
Start server listening on for vulnerability lookup request`,
	Run: func(cmd *cobra.Command, args []string) {

		csvDir := viper.GetString("csvdir")
		addr := viper.GetString("address")
		port := viper.GetInt("port")

		log.Setup(viper.GetBool("debug"))

		log.Info(log.M{Msg: "Starting " + progName + " " + version})

		err := server.InitCSV(csvDir)
		if err != nil {
			exit("Cannot read Nessus CSV from "+csvDir, err)
		}

		err = server.Start(addr, port)
		if err != nil {
			exit("Cannot start server", err)
		}
	},
}
