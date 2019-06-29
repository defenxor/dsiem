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
	"fmt"
	"os"

	"github.com/defenxor/dsiem/internal/pkg/ossimcnv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	srcFile     string
	dstFile     string
	ossimRefDir string
	nSplit      int
)

const (
	progName = "ossimcnv"
)

var version string
var buildTime string

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringP("in", "i", "./user.xml", "source OSSIM directive XML file to convert, e.g. point to user.xml path")
	rootCmd.PersistentFlags().StringP("out", "o", "./directives_ossim.json", "source OSSIM directive XML file to convert, e.g. point to user.xml path")
	rootCmd.PersistentFlags().StringP("refdir", "r", "./ossimref", "location of TSV files produced by running dumptable.sh in OSSIM server")
	rootCmd.PersistentFlags().IntP("split", "n", 1, "split the directive .json content to this number of files")

	viper.BindPFlag("in", rootCmd.PersistentFlags().Lookup("in"))
	viper.BindPFlag("out", rootCmd.PersistentFlags().Lookup("out"))
	viper.BindPFlag("refdir", rootCmd.PersistentFlags().Lookup("refdir"))
	viper.BindPFlag("split", rootCmd.PersistentFlags().Lookup("split"))
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
	Use:   "ossimcnv",
	Short: "OSSIM directive converter",
	Long:  `Ossimcnv converts OSSIM directives to Dsiem directives format`,
	Run: func(cmd *cobra.Command, args []string) {
		in := viper.GetString("in")
		out := viper.GetString("out")
		refdir := viper.GetString("refdir")
		split := viper.GetInt("split")

		filename, err := ossimcnv.CreateTempOSSIMFile(in)
		if err != nil {
			exit("Cannot create temporary XML file", err)
			return
		}
		err = ossimcnv.ParseOSSIMTSVs(refdir)
		if err != nil {
			exit("Cannot parse ossim reference TSV from "+refdir, err)
			return
		}
		err = ossimcnv.CreateSIEMDirective(filename, out, split)
		if err != nil {
			exit("Cannot create Dsiem json directive", err)
			return
		}
		fmt.Println("Done.")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long:  `Print the version and build date information`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version, buildTime)
	},
}
