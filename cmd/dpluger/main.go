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
	"errors"
	"fmt"
	"os"
	"strings"

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

	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(directiveCmd)
	rootCmd.AddCommand(splitterCmd)
	rootCmd.PersistentFlags().StringP("config", "c", "dpluger_config.json", "config file to use")
	createCmd.Flags().StringP("address", "a", "http://elasticsearch:9200", "Elasticsearch endpoint to use")
	createCmd.Flags().StringP("indexPattern", "i", "suricata-*", "index pattern to read fields from")
	createCmd.Flags().StringP("name", "n", "suricata", "the name of the generated plugin")
	createCmd.Flags().StringP("type", "t", "SID", "the type of the generated plugin, can be SID or Taxonomy")

	directiveCmd.Flags().StringP("tsvFile", "f", "", "dpluger TSV file to use")
	directiveCmd.Flags().StringP("outFile", "o", "directives_dsiem.json", "directive file to create")
	directiveCmd.Flags().StringP("priority", "p", "3", "default priority to use (1 - 5)")
	directiveCmd.Flags().StringP("reliability", "r", "1", "reliability to use (0 - 10) for stage 1")
	directiveCmd.Flags().StringP("kingdom", "k", "Environmental Awareness", "default kingdom to use")
	directiveCmd.Flags().StringP("titleTemplate", "m", "EVENT_TITLE (SRC_IP to DST_IP)", "directive title template. EVENT_TITLE will be replaced with entries from TSV file, and SRC_IP and DST_IP will be replaced at runtime with the actual addresses.")
	directiveCmd.Flags().IntP("dirNumber", "i", 100000, "Starting directive number")
	splitterCmd.Flags().StringP("targetFile", "i", "", "Directive file to split")
	splitterCmd.Flags().StringP("suffix", "s", "_", "Suffix for generated file, example: test.json will be splitted into test_1.json and test_2.json if suffix set to _")
	splitterCmd.Flags().IntP("min", "n", 1024, "Directive count for splitted files")
	splitterCmd.Flags().BoolP("delete-input", "d", false, "Delete input file after splitting")

	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("address", createCmd.Flags().Lookup("address"))
	viper.BindPFlag("index", createCmd.Flags().Lookup("indexPattern"))
	viper.BindPFlag("name", createCmd.Flags().Lookup("name"))
	viper.BindPFlag("type", createCmd.Flags().Lookup("type"))
	viper.BindPFlag("tsvFile", directiveCmd.Flags().Lookup("tsvFile"))
	viper.BindPFlag("outFile", directiveCmd.Flags().Lookup("outFile"))
	viper.BindPFlag("priority", directiveCmd.Flags().Lookup("priority"))
	viper.BindPFlag("reliability", directiveCmd.Flags().Lookup("reliability"))
	viper.BindPFlag("kingdom", directiveCmd.Flags().Lookup("kingdom"))
	viper.BindPFlag("dirNumber", directiveCmd.Flags().Lookup("dirNumber"))
	viper.BindPFlag("titleTemplate", directiveCmd.Flags().Lookup("titleTemplate"))
	viper.BindPFlag("targetFile", splitterCmd.Flags().Lookup("targetFile"))
	viper.BindPFlag("suffix", splitterCmd.Flags().Lookup("suffix"))
	viper.BindPFlag("min", splitterCmd.Flags().Lookup("min"))
	viper.BindPFlag("delete", splitterCmd.Flags().Lookup("delete-input"))
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

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an empty config template for dpluger",
	Long:  `Create an empty config template for dpluger`,
	Run: func(cmd *cobra.Command, args []string) {
		config := viper.GetString("config")
		address := viper.GetString("address")
		index := viper.GetString("index")
		name := viper.GetString("name")
		typ := viper.GetString("type")
		if err := dpluger.CreateConfig(config, address, index, name, typ); err != nil {
			exit("Cannot parse config file", err)
		}
		fmt.Println("Template created in " + config + "\n" +
			"Now you should edit the generated template and insert the appropriate parameters and ES field names.")
	},
}

var directiveCmd = &cobra.Command{
	Use:   "directive",
	Short: "Create a Dsiem directive file from dpluger TSV",
	Long:  `Create a Dsiem directive file from dpluger TSV`,
	Run: func(cmd *cobra.Command, args []string) {
		tsvFile := viper.GetString("tsvFile")
		outFile := viper.GetString("outFile")
		priority := viper.GetInt("priority")
		reliability := viper.GetInt("reliability")
		kingdom := viper.GetString("kingdom")
		dirNumber := viper.GetInt("dirNumber")
		titleTemplate := viper.GetString("titleTemplate")

		if !strings.Contains(titleTemplate, "EVENT_TITLE") {
			exit("titleTemplate must contain EVENT_TITLE", errors.New("wrong titleTemplate"))
		}

		if priority < 1 || priority > 5 {
			exit("Priority must be between 1 and 5", errors.New("wrong priority"))
		}
		if reliability < 0 || reliability > 10 {
			exit("Reliability must be between 0 - 10", errors.New("wrong reliability"))
		}
		if dirNumber < 1 {
			exit("dirNumber must be greater than 0", errors.New("wrong dirNumber"))
		}

		if !fs.FileExist(tsvFile) {
			exit(tsvFile+" doesn't exist", errors.New("wrong TSVFile parameter"))
		}

		if err := dpluger.CreateDirective(tsvFile, outFile, kingdom, titleTemplate,
			priority, reliability, dirNumber); err != nil {
			exit("Cannot create directive file", err)
		}
		fmt.Println("Directives file written in " + outFile + "\n" +
			"Now you should edit the generated file and deploy it to dsiem frontend configs directory")
	},
}

var splitterCmd = &cobra.Command{
	Use:   "split-directive",
	Short: "Split directive file",
	Long:  `Split large directive file into multiple files with certain suffix`,
	Run: func(cmd *cobra.Command, args []string) {
		target := viper.GetString("targetFile")
		suffix := viper.GetString("suffix")
		count := viper.GetInt("min")
		delete := viper.GetBool("delete")

		if !fs.FileExist(target) {
			exit(target+" doesn't exist", errors.New("wrong target parameter"))
		}

		if count < 1 {
			exit("count must be greater than 0", errors.New("wrong count"))
		}

		if err := dpluger.SplitDirective(target, suffix, count, delete); err != nil {
			exit("Failed to split "+target+" file", err)
		}

		fmt.Println("File ", target, " has been splitted")
	},
}
