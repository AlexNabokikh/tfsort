package hclsort

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ValidateFilePath checks if the path is valid for processing.
func ValidateFilePath(path string) error {
	if path == "" {
		return errors.New("file path is required")
	}

	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return errors.New("file does not exist")
	case err != nil:
		return fmt.Errorf("error accessing file '%s': %w", path, err)
	case info.IsDir():
		return errors.New("path is a directory, not a file")
	default:
		return nil
	}
}

// CheckFileExtension verifies the file extension against a list of allowed types.
func CheckFileExtension(path string, allowedTypes map[string]bool) error {
	fileExtension := ""
	ext := filepath.Ext(path)
	if len(ext) > 0 {
		fileExtension = ext[1:]
	}

	if !allowedTypes[fileExtension] {
		if fileExtension != "" {
			return fmt.Errorf(
				"file extension '%s' is not a supported Terraform/HCL type",
				fileExtension,
			)
		}
	}
	return nil
}

// ReadFileBytes reads the content of the file at the given path.
func ReadFileBytes(path string) ([]byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file '%s': %w", path, err)
	}
	return src, nil
}

// WriteSortedContent handles writing the outputBytes to the specified destination.
func WriteSortedContent(
	originalPathOrMarker string,
	outputPath string,
	dryRun bool,
	outputBytes []byte,
	isInputFromStdin bool,
) error {
	finalBytes := append(bytes.TrimSpace(outputBytes), '\n')

	switch {
	case outputPath != "":
		err := os.WriteFile(outputPath, finalBytes, 0644)
		if err != nil {
			return fmt.Errorf(
				"error writing output to file '%s': %w",
				outputPath,
				err,
			)
		}
	case dryRun:
		fmt.Print(string(finalBytes))
	case isInputFromStdin:
		_, err := fmt.Print(string(finalBytes))
		if err != nil {
			return fmt.Errorf("error writing to stdout: %w", err)
		}
	default:
		err := os.WriteFile(originalPathOrMarker, finalBytes, 0644)
		if err != nil {
			return fmt.Errorf(
				"error writing output to file '%s': %w",
				originalPathOrMarker,
				err,
			)
		}
	}
	return nil
}
