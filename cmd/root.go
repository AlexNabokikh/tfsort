package cmd

import (
	"log"

	"github.com/AlexNabokikh/tfsort/tsort"
	"github.com/spf13/cobra"
)

// Execute is the entry point for the CLI.
func Execute() {
	var (
		filePath   string
		outputPath string
		dryRun     bool
		recursive  bool
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

			if err := tsort.ValidateFilePath(filePath); err != nil {
				return err
			}

			i := tsort.NewIngestor()

			if recursive {
				// Ignore the outputPath when in recursive mode
				return i.ParseAll(filePath, dryRun)
			}

			return i.Parse(filePath, outputPath, dryRun)

		},
	}

	rootCmd.PersistentFlags().StringVarP(
		&outputPath,
		"out",
		"o",
		"",
		"path to the output file. Ignored if --recursive is used.")
	rootCmd.PersistentFlags().BoolVarP(
		&dryRun,
		"dry-run",
		"d", false,
		"preview the changes without altering the original file.")
	rootCmd.PersistentFlags().BoolVarP(
		&recursive,
		"recursive",
		"r", false,
		"parse all Terraform files within the provided directory and its subdirectories recursively")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
