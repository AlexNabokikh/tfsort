package cmd

import (
	"errors"
	"fmt"
	"io/fs"

	"os"
	"path/filepath"
	"strings"

	"github.com/AlexNabokikh/tfsort/internal/hclsort"
	"github.com/spf13/cobra"
)

// Execute is the entry point for the CLI.
func Execute(version, commit, date string) {
	var (
		outputPath string
		dryRun     bool
	)

	rootCmd := &cobra.Command{
		Use:   "tfsort [flags] [files...]",
		Short: "A utility to sort Terraform variables and outputs.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			ingestor := hclsort.NewIngestor()
			paths, err := argsToPaths(args)
			if err != nil {
				return err
			}

			return processPaths(ingestor, paths, dryRun, outputPath)
		},
	}

	// SilenceUsage ensures that the usage message is not printed on every error.
	rootCmd.SilenceUsage = true

	// Set the version string for Cobra to use
	if version != "" && version != "dev" {
		rootCmd.Version = fmt.Sprintf(
			"%s (commit: %s, built: %s)",
			version,
			commit,
			date,
		)
	} else {
		rootCmd.Version = "dev (build details not available)"
		if version == "dev" && commit != "none" && date != "unknown" {
			rootCmd.Version = fmt.Sprintf(
				"dev (commit: %s, built: %s)",
				commit,
				date,
			)
		}
	}

	rootCmd.PersistentFlags().StringVarP(
		&outputPath,
		"out",
		"o",
		"",
		"path to the output file (cannot be used with --recursive)",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&dryRun,
		"dry-run",
		"d", false,
		"preview the changes without altering the original file(s).",
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func argsToPaths(args []string) ([]string, error) {
	if len(args) == 0 {
		return []string{"."}, nil // default to current directory
	}

	if len(args) == 1 && args[0] == "-" {
		isStdin, err := useStdin()
		if err != nil {
			return args, err
		}

		if isStdin {
			return []string{hclsort.StdInPathIdentifier}, nil
		}
	}

	return args, nil
}

// processPaths processes the provided paths, handling both files and directories.
// It will walk through directories recursively.
func processPaths(
	ingestor *hclsort.Ingestor,
	paths []string,
	dryRun bool,
	outputPath string,
) error {
	if len(paths) == 1 && paths[0] == hclsort.StdInPathIdentifier {
		return ingestor.Parse(paths[0], outputPath, dryRun, true)
	}

	pathErrors := []error{}
	for _, path := range paths {
		stat, statErr := os.Stat(path)
		if statErr != nil {
			pathErrors = append(pathErrors, fmt.Errorf("failed to stat path: %w", statErr))
			continue
		}

		if stat.IsDir() {
			// Recursive
			err := filepath.WalkDir(path, newWalkDirCallback(ingestor, dryRun))
			if err != nil {
				pathErrors = append(pathErrors, fmt.Errorf("error walking directory '%s': %w", path, err))
			}
		} else {
			// Single file
			if err := hclsort.ValidateFilePath(path); err != nil {
				pathErrors = append(pathErrors, fmt.Errorf("error validating file '%s': %w", path, err))
				continue
			}

			err := ingestor.Parse(path, outputPath, dryRun, false)
			if err != nil {
				pathErrors = append(pathErrors, fmt.Errorf("error processing file '%s': %w", path, err))
			}
		}
	}

	if len(pathErrors) > 0 {
		errStrings := make([]string, len(pathErrors))
		for i, e := range pathErrors {
			errStrings[i] = e.Error()
		}
		return fmt.Errorf("could not process all paths:\n%s", strings.Join(errStrings, "\n"))
	}

	return nil
}

// newWalkDirCallback creates a callback function for filepath.WalkDir.
func newWalkDirCallback(
	ingestor *hclsort.Ingestor,
	isDryRun bool,
) fs.WalkDirFunc {
	return func(currentPath string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Warning: error accessing path %s: %v\n",
				currentPath,
				err,
			)
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}

			return err
		}

		if d.IsDir() {
			dirName := d.Name()
			if dirName == ".git" ||
				dirName == ".terraform" ||
				dirName == ".terragrunt-cache" {
				if !isDryRun {
					fmt.Printf("Skipping directory: %s\n", currentPath)
				}
				return filepath.SkipDir
			}

			return nil
		}

		fileExtension := strings.TrimPrefix(filepath.Ext(currentPath), ".")
		if !ingestor.AllowedTypes[fileExtension] {
			return nil
		}

		if !isDryRun {
			fmt.Printf("Processing %s...\n", currentPath)
		}
		err = ingestor.Parse(currentPath, "", isDryRun, false)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Error sorting file %s: %v\n",
				currentPath,
				err,
			)
		}
		return nil
	}
}

// useStdin determines whether to read stdin.
func useStdin() (bool, error) {
	stat, statErr := os.Stdin.Stat()
	if statErr != nil {
		return false, fmt.Errorf("failed to stat stdin: %w", statErr)
	}

	return (stat.Mode() & os.ModeCharDevice) == 0, nil
}
