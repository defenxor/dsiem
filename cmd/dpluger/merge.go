package main

import (
	"log"

	"github.com/defenxor/dsiem/internal/pkg/cmd"
	"github.com/defenxor/dsiem/internal/pkg/dpluger"
	"github.com/spf13/cobra"
)

var mergeCmd = &cobra.Command{
	Use:   "merge-directive",
	Short: "safely merge two directive files",
	Long:  `Safely merge two directive files, and upload it to dsiem`,
	Run: func(c *cobra.Command, args []string) {
		host, err := c.Flags().GetString("frontend-host")
		if err != nil {
			log.Fatal(err.Error())
		}

		source, err := c.Flags().GetString("source-directive")
		if err != nil {
			log.Fatal(err.Error())
		}

		target, err := c.Flags().GetString("file")
		if err != nil {
			log.Fatal(err.Error())
		}

		commander := cmd.NewCommand()
		if err := dpluger.Merge(commander, dpluger.MergeConfig{
			Host:       host,
			SourceJSON: source,
			TargetJSON: target,
		}); err != nil {
			log.Fatal(err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().String("frontend-host", "http://frontend:8080", "dsiem frontend host address, default to http://frontend:8080")
	mergeCmd.Flags().String("source-directive", "", "existing directive name to be merged with the new directive file, required.")
	mergeCmd.Flags().String("file", "", "path new directive file to be merged with the existing directive, required.")

	if err := mergeCmd.MarkFlagRequired("source-directive"); err != nil {
		log.Fatal(err.Error())
	}

	if err := mergeCmd.MarkFlagRequired("file"); err != nil {
		log.Fatal(err.Error())
	}
}
