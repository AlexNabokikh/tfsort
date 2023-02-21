package tsort

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Ingestor struct {
	AllowedTypes  map[string]bool
	AllowedBlocks map[string]bool
}

// Ingestor constructor that sets the default values for the allowed file types and blocks.
func NewIngestor() *Ingestor {
	return &Ingestor{
		AllowedTypes: map[string]bool{
			"tf":  true,
			"hcl": true,
		},
		AllowedBlocks: map[string]bool{
			"variable": true,
			"output":   true,
		},
	}
}

// CanIngest reads the file at the given path and checks if it is a valid Terraform file
// based on its extension and contents.
func (i *Ingestor) CanIngest(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("can't open file '%s': no such file or directory", path)
	}

	extension := filepath.Ext(path)[1:]
	if !i.AllowedTypes[extension] {
		return fmt.Errorf("file %s is not a valid Terraform file", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file '%s': %w", path, err)
	}

	for block := range i.AllowedBlocks {
		if i.AllowedBlocks[block] && strings.Contains(string(content), block) {
			return nil
		}
	}

	return fmt.Errorf("file %s is not a valid Terraform file", path)
}

// Parse extracts variable and output blocks from the Terraform file at the given path,
// sorts them alphabetically by name, and writes the output to the specified file or to stdout.
func (i *Ingestor) Parse(path string, outputPath string, dry bool) error {
	if err := i.CanIngest(path); err != nil {
		return err
	}

	pattern := regexp.MustCompile(`(?:variable|output) "([\w\d]+)" {\n[\w\W]+?\n}`)

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file '%s': %w", path, err)
	}

	matches := pattern.FindAllString(string(content), -1)
	sort.Slice(matches, func(i, j int) bool {
		nameI := pattern.FindAllStringSubmatch(matches[i], 1)[0][1]
		nameJ := pattern.FindAllStringSubmatch(matches[j], 1)[0][1]

		return nameI < nameJ
	})
	output := strings.Join(matches, "\n\n") + "\n"

	switch {
	case outputPath != "":
		err = os.WriteFile(outputPath, []byte(output), 0o644)
		if err != nil {
			return fmt.Errorf("error writing output to file '%s': %w", outputPath, err)
		}
	case dry:
		fmt.Println(output)
	default:
		err = os.WriteFile(path, []byte(output), 0o644)
		if err != nil {
			return fmt.Errorf("error writing output to file '%s': %w", path, err)
		}
	}

	return nil
}

// validateFilePath checks that the given file path is valid and points to an existing file.
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error accessing file '%s': %w", path, err)
	}

	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("file '%s' does not exist", path)
	case info.IsDir():
		return fmt.Errorf("'%s' is a directory, not a file", path)
	default:
		return nil
	}
}

// validateOutputPath checks that the given output path, if not empty, is a valid file path.
// func ValidateOutputPath(path string) error {
// 	if path == "" {
// 		return nil
// 	}

// 	return ValidateFilePath(path)
// }
