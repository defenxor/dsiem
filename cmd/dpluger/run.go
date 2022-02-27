package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"github.com/defenxor/dsiem/internal/pkg/dpluger"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runCmd.Flags().BoolP("skipTLSVerify", "s", false, "whether to skip ES server certificate verification (when using HTTPS)")
	runCmd.Flags().BoolP("usePipeline", "p", false, "whether to generate plugin that is suitable for logstash pipeline to pipeline configuration")
	runCmd.Flags().BoolP("validate", "v", true, "Check whether each referred ES field exists on the target index")
	runCmd.Flags().StringP("sid-list", "f", "", "optional, Plugin SID list file to use for generating logstash plugin (tsv-formatted). If set, dpluger will not generate any .tsv file, and assumes that you already have a .tsv file containing list of plugin SID, either by previous dpluger run or created manually")
	viper.BindPFlag("validate", runCmd.Flags().Lookup("validate"))
	viper.BindPFlag("skipTLSVerify", runCmd.Flags().Lookup("skipTLSVerify"))
	viper.BindPFlag("usePipeline", runCmd.Flags().Lookup("usePipeline"))
	viper.BindPFlag("sid-list", runCmd.Flags().Lookup("sid-list"))

	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Create Logstash plugin for Dsiem",
	Long:  `Create Logstash plugin for Dsiem`,
	Run: func(cmd *cobra.Command, args []string) {
		config := viper.GetString("config")
		validate := viper.GetBool("validate")
		skipTLSVerify := viper.GetBool("skipTLSVerify")
		usePipeline := viper.GetBool("usePipeline")
		SIDListFile := viper.GetString("sid-list")

		if skipTLSVerify {
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}

		if !fs.FileExist(config) {
			exit("Cannot read from config file", errors.New(config+" doesn't exist"))
		}
		if err := log.Setup(true); err != nil {
			exit("Cannot setup logger", err)
		}

		plugin, err := dpluger.Parse(config)
		if err != nil {
			exit("Cannot parse config file", err)
		}

		if err := dpluger.CreatePlugin(dpluger.CreatePluginConfig{
			Plugin:      plugin,
			ConfigFile:  config,
			Creator:     progName,
			Validate:    validate,
			UsePipeline: usePipeline,
			SIDListFile: SIDListFile,
		}); err != nil {
			exit("Error encountered while running config file", err)
		}

		fmt.Println("Logstash conf file created.")
	},
}
