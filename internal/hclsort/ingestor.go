package hclsort

import (
	"fmt"
	"io"
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
	inputPath string,
	outputPath string,
	dryRun bool,
	isStdin bool,
) error {
	var src []byte
	var err error
	filenameForParser := inputPath

	if isStdin {
		src, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("error reading from stdin: %w", err)
		}
	} else {
		if extErr := CheckFileExtension(inputPath, i.AllowedTypes); extErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", extErr)
		}
		src, err = ReadFileBytes(inputPath)
		if err != nil {
			return err
		}
	}

	hclFile, err := ParseHCLContent(src, filenameForParser)
	if err != nil {
		return err
	}

	processedFile := ProcessAndSortBlocks(hclFile, i.AllowedBlocks)

	formattedBytes := FormatHCLBytes(processedFile)

	return WriteSortedContent(inputPath, outputPath, dryRun, formattedBytes, isStdin)
}
