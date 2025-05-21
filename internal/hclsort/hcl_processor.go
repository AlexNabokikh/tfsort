package hclsort

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// ParseHCLContent parses the HCL source byte slice using hclwrite.
func ParseHCLContent(
	src []byte,
	filename string,
) (*hclwrite.File, error) {
	file, diags := hclwrite.ParseConfig(
		src,
		filename,
		hcl.Pos{Line: 1, Column: 1},
	)
	if diags.HasErrors() {
		return nil, fmt.Errorf(
			"error parsing HCL content from '%s': %w",
			filename,
			diags,
		)
	}
	return file, nil
}

// sortRequiredProvidersInBlock sorts the entries in any required_providers block
// within a terraform block, alphabetically by provider name, preserving tokens.
func sortRequiredProvidersInBlock(block *hclwrite.Block) {
	for _, b := range block.Body().Blocks() {
		if b.Type() != "required_providers" {
			continue
		}
		body := b.Body()
		attrs := body.Attributes()

		providerNames := make([]string, 0, len(attrs))
		for name := range attrs {
			providerNames = append(providerNames, name)
		}
		sort.Strings(providerNames)

		body.Clear()
		body.AppendNewline()

		for i, name := range providerNames {
			attr := attrs[name]
			tokens := attr.BuildTokens(nil)

			start, end := 0, len(tokens)
			for start < end && tokens[start].Type == hclsyntax.TokenNewline {
				start++
			}
			for end > start && tokens[end-1].Type == hclsyntax.TokenNewline {
				end--
			}
			body.AppendUnstructuredTokens(tokens[start:end])
			if i+1 < len(providerNames) {
				body.AppendNewline()
			}
		}
		body.AppendNewline()
	}
}

// ProcessAndSortBlocks extracts sortable blocks (variables, outputs) from the HCL file
// and also applies provider sorting in terraform blocks.
func ProcessAndSortBlocks(
	file *hclwrite.File,
	allowedBlocks map[string]bool,
) *hclwrite.File {
	for _, block := range file.Body().Blocks() {
		if block.Type() == "terraform" {
			sortRequiredProvidersInBlock(block)
		}
	}

	body := file.Body()
	originalBlocks := body.Blocks()

	sortableItems := make([]*SortableBlock, 0)
	otherBlocks := make([]*hclwrite.Block, 0)

	for _, block := range originalBlocks {
		blockType := block.Type()
		if allowedBlocks[blockType] && len(block.Labels()) > 0 {
			sortableItems = append(sortableItems, &SortableBlock{
				Name:  block.Labels()[0],
				Block: block,
			})
		} else {
			otherBlocks = append(otherBlocks, block)
		}
	}

	sort.Slice(sortableItems, func(i, j int) bool {
		return sortableItems[i].Name < sortableItems[j].Name
	})

	body.Clear()

	for i, block := range otherBlocks {
		body.AppendBlock(block)
		if i < len(otherBlocks)-1 || len(sortableItems) > 0 {
			body.AppendNewline()
		}
	}

	for i, sb := range sortableItems {
		body.AppendBlock(sb.Block)
		if i < len(sortableItems)-1 {
			body.AppendNewline()
		}
	}

	return file
}

// FormatHCLBytes formats the HCL file's content into a byte slice.
func FormatHCLBytes(file *hclwrite.File) []byte {
	return hclwrite.Format(file.Bytes())
}
