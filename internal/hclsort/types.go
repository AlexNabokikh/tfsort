package hclsort

// StdInPathIdentifier is a marker for when input is read from stdin.
const StdInPathIdentifier = "<stdin>"

// Ingestor is a struct that contains the logic for parsing Terraform files.
type Ingestor struct {
	AllowedTypes  map[string]bool
	AllowedBlocks map[string]bool
}
