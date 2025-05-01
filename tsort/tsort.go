package tsort

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// NewIngestor returns a new Ingestor instance.
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

// CanIngest checks if the file extension is allowed.
func (i *Ingestor) CanIngest(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("can't open file '%s': no such file or directory", path)
	}

	extension := filepath.Ext(path)
	if len(extension) > 0 {
		extension = extension[1:]
	}

	if !i.AllowedTypes[extension] {
		if extension != "" {
			return fmt.Errorf("file extension '%s' is not a supported Terraform/HCL type", extension)
		}
	}

	return nil
}

// Parse extracts variable and output blocks from the Terraform file at the given path,
// sorts them alphabetically by name, and writes the output.
func (i *Ingestor) Parse(path string, outputPath string, dry bool) error {
	if err := i.CanIngest(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) && !strings.Contains(err.Error(), "not a supported") {
			return err
		} else if errors.Is(err, os.ErrNotExist) {
			return err
		}
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file '%s': %w", path, err)
	}

	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL(src, path)

	if diags.HasErrors() {
		return fmt.Errorf("error parsing HCL file '%s': %w", path, diags)
	}

	writeFile, diags := hclwrite.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return fmt.Errorf("error parsing HCL for writing '%s': %w", path, diags)
	}

	body := writeFile.Body()
	blocks := body.Blocks()
	sortableBlocks := make([]*SortableBlock, 0)
	otherTokens := []*hclwrite.Block{}

	for _, block := range blocks {
		blockType := block.Type()
		if i.AllowedBlocks[blockType] && len(block.Labels()) > 0 {
			sortableBlocks = append(sortableBlocks, &SortableBlock{
				Name:  block.Labels()[0],
				Block: block,
			})
		} else {
			otherTokens = append(otherTokens, block)
		}
	}

	sort.Slice(sortableBlocks, func(i, j int) bool {
		return sortableBlocks[i].Name < sortableBlocks[j].Name
	})

	body.Clear()

	for i, block := range otherTokens {
		body.AppendBlock(block)
		if i < len(otherTokens)-1 || len(sortableBlocks) > 0 {
			body.AppendNewline()
		}
	}

	for i, sb := range sortableBlocks {
		body.AppendBlock(sb.Block)
		if i < len(sortableBlocks)-1 {
			body.AppendNewline()
		}
	}

	outputBytes := hclwrite.Format(writeFile.Bytes())
	outputBytes = append(bytes.TrimSpace(outputBytes), '\n')

	switch {
	case outputPath != "":
		err := os.WriteFile(outputPath, outputBytes, 0o644)
		if err != nil {
			return fmt.Errorf("error writing output to file '%s': %w", outputPath, err)
		}
	case dry:
		fmt.Print(string(outputBytes))
	default:
		err := os.WriteFile(path, outputBytes, 0o644)
		if err != nil {
			return fmt.Errorf("error writing output to file '%s': %w", path, err)
		}
	}

	return nil
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
		return errors.New("path is a directory, not a file")
	default:
		return nil
	}
}
