package cmd

import (
	"log"

	"github.com/AlexNabokikh/tfsort/internal/hclsort"
	"github.com/spf13/cobra"
)

// Execute is the entry point for the CLI.
func Execute() {
	var (
		filePath   string
		outputPath string
		dryRun     bool
	)

	rootCmd := &cobra.Command{
		Use:   "tfsort [file]",
		Short: "A utility to sort Terraform variables and outputs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				filePath = args[0]
			}

			if filePath == "" {
				return cmd.Usage()
			}

			if err := hclsort.ValidateFilePath(filePath); err != nil {
				return err
			}

			i := hclsort.NewIngestor()

			return i.Parse(filePath, outputPath, dryRun)
		},
	}

	rootCmd.PersistentFlags().StringVarP(
		&outputPath,
		"out",
		"o",
		"",
		"path to the output file")
	rootCmd.PersistentFlags().BoolVarP(
		&dryRun,
		"dry-run",
		"d", false,
		"preview the changes without altering the original file.")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
