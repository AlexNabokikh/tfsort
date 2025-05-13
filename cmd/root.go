package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlexNabokikh/tfsort/internal/hclsort"
	"github.com/spf13/cobra"
)

// errShowHelp is a sentinel error used to indicate that help should be shown.
var errShowHelp = errors.New("show help requested")

// Execute is the entry point for the CLI.
func Execute(version, commit, date string) {
	var (
		outputPath string
		dryRun     bool
		recursive  bool
	)

	rootCmd := &cobra.Command{
		Use: "tfsort [file_or_directory|-]",
		Short: "A utility to sort Terraform variables and outputs. " +
			"If no file is specified or '-' is used as the filename, input is read from stdin. " +
			"If a directory is provided with -r, files are processed recursively.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ingestor := hclsort.NewIngestor()

			if recursive {
				return runRecursiveMode(ingestor, args, dryRun, outputPath)
			}
			return runSingleMode(cmd, ingestor, args, outputPath, dryRun)
		},
	}

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
	rootCmd.PersistentFlags().BoolVarP(
		&recursive,
		"recursive",
		"r",
		false,
		"recursively sort files in a directory (in-place unless --dry-run is specified)",
	)

	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, errShowHelp) {
			log.Fatalf("error: %s", err)
		}
	}
}

// runRecursiveMode handles the logic when the --recursive flag is used.
func runRecursiveMode(
	ingestor *hclsort.Ingestor,
	args []string,
	isDryRun bool,
	outPath string,
) error {
	if outPath != "" {
		return errors.New(
			"the -o/--out flag cannot be used with -r/--recursive",
		)
	}
	if len(args) == 0 {
		return errors.New(
			"a directory path must be specified when using --recursive",
		)
	}
	inputPath := args[0]
	if inputPath == "-" {
		return errors.New("stdin input ('-') cannot be used with --recursive")
	}

	info, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf(
			"failed to stat input path '%s': %w",
			inputPath,
			err,
		)
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"inputPath '%s' is not a directory; --recursive requires a directory",
			inputPath,
		)
	}

	if !isDryRun {
		fmt.Printf("Recursively processing directory: %s\n", inputPath)
	}

	walkErr := filepath.WalkDir(
		inputPath,
		newWalkDirCallback(ingestor, isDryRun),
	)
	if walkErr != nil {
		return fmt.Errorf(
			"error walking directory '%s': %w",
			inputPath,
			walkErr,
		)
	}
	return nil
}

// newWalkDirCallback creates a callback function for filepath.WalkDir.
func newWalkDirCallback(
	ingestor *hclsort.Ingestor,
	isDryRun bool,
) fs.WalkDirFunc {
	return func(currentPath string, d fs.DirEntry, errInWalk error) error {
		if errInWalk != nil {
			fmt.Fprintf(
				os.Stderr,
				"Warning: error accessing path %s: %v\n",
				currentPath,
				errInWalk,
			)
			if errors.Is(errInWalk, fs.ErrPermission) {
				return nil
			}
			return errInWalk
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
		errParse := ingestor.Parse(currentPath, "", isDryRun, false)
		if errParse != nil {
			fmt.Fprintf(
				os.Stderr,
				"Error sorting file %s: %v\n",
				currentPath,
				errParse,
			)
		}
		return nil
	}
}

// runSingleMode handles the logic for a single file input or stdin.
func runSingleMode(
	cmd *cobra.Command,
	ingestor *hclsort.Ingestor,
	args []string,
	outPath string,
	isDryRun bool,
) error {
	currentInputPath, isStdin, err := determineInputSource(args)
	if err != nil {
		if errors.Is(err, errShowHelp) {
			_ = cmd.Help()
			return nil
		}
		return err
	}

	if !isStdin {
		if validationErr := hclsort.ValidateFilePath(currentInputPath); validationErr != nil {
			return validationErr
		}
	}

	return ingestor.Parse(currentInputPath, outPath, isDryRun, isStdin)
}

// determineInputSource determines the input path and whether it's from stdin.
func determineInputSource(
	args []string,
) (string, bool, error) {
	if len(args) == 0 {
		stat, statErr := os.Stdin.Stat()
		if statErr != nil {
			return "", false, fmt.Errorf("failed to stat stdin: %w", statErr)
		}
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			return hclsort.StdInPathIdentifier, true, nil
		}
		return "", false, errShowHelp
	}

	pathArg := args[0]
	if pathArg == "-" {
		return hclsort.StdInPathIdentifier, true, nil
	}
	return pathArg, false, nil
}
