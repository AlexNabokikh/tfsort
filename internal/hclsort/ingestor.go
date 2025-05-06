package hclsort

import (
	"fmt"
	"os"
)

// NewIngestor returns a new Ingestor instance with default allowed types and blocks.
func NewIngestor() *Ingestor {
	return &Ingestor{
		AllowedTypes: map[string]bool{
			"tf":   true,
			"hcl":  true,
			"tofu": true,
		},
		AllowedBlocks: map[string]bool{
			"variable": true,
			"output":   true,
		},
	}
}

// Parse orchestrates the reading, parsing, sorting, and writing of a Terraform/HCL file.
func (i *Ingestor) Parse(
	path string,
	outputPath string,
	dryRun bool,
) error {
	if extErr := CheckFileExtension(path, i.AllowedTypes); extErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", extErr)
	}

	src, err := ReadFileBytes(path)
	if err != nil {
		return err
	}

	hclFile, err := ParseHCLContent(src, path)
	if err != nil {
		return err
	}

	processedFile := ProcessAndSortBlocks(hclFile, i.AllowedBlocks)

	formattedBytes := FormatHCLBytes(processedFile)

	return WriteSortedContent(path, outputPath, dryRun, formattedBytes)
}
