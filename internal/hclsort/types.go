package hclsort

import "github.com/hashicorp/hcl/v2/hclwrite"

// StdInPathIdentifier is a marker for when input is read from stdin.
const StdInPathIdentifier = "<stdin>"

// Ingestor is a struct that contains the logic for parsing Terraform files.
type Ingestor struct {
	AllowedTypes  map[string]bool
	AllowedBlocks map[string]bool
}

// SortableBlock holds information needed for sorting.
type SortableBlock struct {
	Name  string
	Block *hclwrite.Block
}
