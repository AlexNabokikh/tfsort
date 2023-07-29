package tsort

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Ingestor is a struct that contains the logic for parsing Terraform files.
type Ingestor struct {
	AllowedTypes  map[string]bool
	AllowedBlocks map[string]bool
}

// Ingestor returns a new Ingestor instance.
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
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
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

	pattern := regexp.MustCompile(`(?:(?:variable|output) "([\w\d\-]+)" {\n[\w\W]+?\n})|(?:\w+\s*{\n[\w\W]+?\n})`)

	content, _ := os.ReadFile(path)

	matches := pattern.FindAllString(string(content), -1)
	sort.Slice(matches, func(i, j int) bool {
		nameI := pattern.FindAllStringSubmatch(matches[i], 1)[0][1]
		nameJ := pattern.FindAllStringSubmatch(matches[j], 1)[0][1]

		return nameI < nameJ
	})

	output := strings.Join(matches, "\n\n") + "\n"

	switch {
	case outputPath != "":
		err := os.WriteFile(outputPath, []byte(output), 0o644)
		if err != nil {
			return fmt.Errorf("error writing output to file '%s': %w", outputPath, err)
		}
	case dry:
		fmt.Println(output)
	default:
		err := os.WriteFile(path, []byte(output), 0o644)
		if err != nil {
			return fmt.Errorf("error writing output to file '%s': %w", path, err)
		}
	}

	return nil
}

// Parses all files within the given path (including subdirectories).
func (i *Ingestor) ParseAll(path string, dry bool) error {
	return filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		switch {
		case err != nil:
			return fmt.Errorf("error accessing file '%s': %w", path, err)
		case info.IsDir():
			return nil
		case filepath.Ext(path) == ".tf" || filepath.Ext(path) == ".hcl":
			return i.Parse(path, "", dry)
		default:
			return nil
		}
	})
}

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
		return nil
	default:
		return nil
	}
}
