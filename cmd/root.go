package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/AlexNabokikh/tfsort/internal/hclsort"
	"github.com/spf13/cobra"
)

// Execute is the entry point for the CLI.
func Execute() {
	var (
		outputPath string
		dryRun     bool
	)

	rootCmd := &cobra.Command{
		Use:   "tfsort [file|-]",
		Short: "A utility to sort Terraform variables and outputs. If no file is specified or '-' is used as the filename, input is read from stdin.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				stat, err := os.Stdin.Stat()
				if err != nil {
					return fmt.Errorf("failed to stat stdin: %w", err)
				}
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return cmd.Usage()
				}
				ingestor := hclsort.NewIngestor()
				return ingestor.Parse(hclsort.StdInPathIdentifier, outputPath, dryRun, true)
			}

			var isStdin bool
			currentInputPath := args[0]

			if currentInputPath == "-" {
				isStdin = true
				currentInputPath = hclsort.StdInPathIdentifier
			} else {
				isStdin = false
				if err := hclsort.ValidateFilePath(currentInputPath); err != nil {
					return err
				}
			}

			ingestor := hclsort.NewIngestor()
			return ingestor.Parse(currentInputPath, outputPath, dryRun, isStdin)
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
