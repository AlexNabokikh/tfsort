package main

import (
	"log"

	"github.com/AlexNabokikh/tfsort/tsort"
	"github.com/spf13/cobra"
)

func main() {
	var filePath string
	var outputPath string
	var dryRun bool

	var rootCmd = &cobra.Command{
		Use:   "tfsort [file]",
		Short: "Sorts the variables and outputs in a Terraform file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				filePath = args[0]
			}

			if filePath == "" {
				return cmd.Usage()
			}

			if err := tsort.ValidateFilePath(filePath); err != nil {
				return err
			}

			i := tsort.NewIngestor()

			return i.Parse(filePath, outputPath, dryRun)
		},
	}

	rootCmd.PersistentFlags().StringVarP(
		&outputPath,
		"out",
		"o",
		"",
		"Path to the output file")
	rootCmd.PersistentFlags().BoolVarP(
		&dryRun,
		"dry-run",
		"d", false,
		"Preview the changes without altering the original file.")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
