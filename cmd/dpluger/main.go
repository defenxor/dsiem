package main

import (
	"errors"
	"fmt"
	"os"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/defenxor/dsiem/internal/pkg/dpluger"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	progName = "dpluger"
)

var version string
var buildTime string

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.PersistentFlags().StringP("config", "c", "dpluger_config.json", "config file to use")
	createCmd.Flags().StringP("address", "a", "http://elasticsearch:9200", "Elasticsearch endpoint to use")
	createCmd.Flags().StringP("indexPattern", "i", "suricata-*", "index pattern to read fields from")
	createCmd.Flags().StringP("name", "n", "suricata", "the name of the generated plugin")
	createCmd.Flags().StringP("type", "t", "SID", "the type of the generated plugin, can be SID or Taxonomy")
	runCmd.Flags().BoolP("validate", "v", true, "Check whether each referred ES field exists on the target index")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("address", createCmd.Flags().Lookup("address"))
	viper.BindPFlag("index", createCmd.Flags().Lookup("indexPattern"))
	viper.BindPFlag("name", createCmd.Flags().Lookup("name"))
	viper.BindPFlag("type", createCmd.Flags().Lookup("type"))
	viper.BindPFlag("validate", runCmd.Flags().Lookup("validate"))
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
	Use:   "dpluger",
	Short: "Logstash config creator for Dsiem",
	Long: `
Dpluger reads existing elasticsearch index pattern and creates a Dsiem logstash
config file (i.e. a plugin) from it.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long:  `Print the version and build date information`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version, buildTime)
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Creates logstash plugin for dsiem",
	Long:  `Creates logstash plugin for dsiem`,
	Run: func(cmd *cobra.Command, args []string) {
		config := viper.GetString("config")
		validate := viper.GetBool("validate")

		if !fs.FileExist(config) {
			exit("Cannot read from config file", errors.New(config+" doesnt exist"))
		}
		if err := log.Setup(true); err != nil {
			exit("Cannot setup logger", err)
		}
		plugin, err := dpluger.Parse(config)
		if err != nil {
			exit("Cannot parse config file", err)
		}
		if err := dpluger.CreatePlugin(plugin, config, progName, validate); err != nil {
			exit("Error encountered while running config file", err)
		}
		fmt.Println("Logstash conf file created.")
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates an empty config template for dpluger",
	Long:  `Creates an empty config template for dpluger`,
	Run: func(cmd *cobra.Command, args []string) {
		config := viper.GetString("config")
		address := viper.GetString("address")
		index := viper.GetString("index")
		name := viper.GetString("name")
		typ := viper.GetString("type")
		if err := dpluger.CreateConfig(config, address, index, name, typ); err != nil {
			exit("Cannot parse config file", err)
		}
		fmt.Println("Template created. in " + config + "\n" +
			"Now you should edit the generated template and insert the appropriate parameters and ES field names.")
	},
}
