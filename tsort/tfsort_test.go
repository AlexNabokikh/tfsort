package tsort_test

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/AlexNabokikh/tfsort/tsort"
)

const (
	validFilePath = "testdata/valid.tf"
	outputFile    = "output.tf"
)

func TestCanIngest(t *testing.T) {
	ingestor := tsort.NewIngestor()

	t.Run("Valid Terraform File", func(t *testing.T) {
		if err := ingestor.CanIngest(validFilePath); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("File not exists", func(t *testing.T) {
		if err := ingestor.CanIngest("notExistFile.tf"); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	t.Run("Invalid File Type", func(t *testing.T) {
		if err := ingestor.CanIngest("invalid_file.txt"); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	t.Run("Invalid File block", func(t *testing.T) {
		if err := os.WriteFile("invalid_file.tf", []byte("data"), 0o600); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if err := ingestor.CanIngest("invalid_file.tf"); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	// cleanup
	os.Remove("invalid_file.tf")
}

func TestParse(t *testing.T) {
	ingestor := tsort.NewIngestor()

	t.Run("Write to output file", func(t *testing.T) {
		os.Remove(outputFile)
		if err := ingestor.Parse(validFilePath, outputFile, false); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file not created")
		}

		data, _ := os.ReadFile(outputFile)
		output := string(data)
		log.Println(output)
		if !strings.Contains(output, "variable") {
			t.Errorf("Unexpected output: %s", output)
		}
	})

	t.Run("Write to stdout", func(t *testing.T) {
		os.Remove(outputFile)
		outputPath := ""
		if err := ingestor.Parse(validFilePath, outputPath, true); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		outputFileInfo, err := os.Stat(outputFile)
		if outputFileInfo != nil || !os.IsNotExist(err) {
			t.Errorf("output file should not be created")
		}
	})

	t.Run("Error writing to output file", func(t *testing.T) {
		if err := os.WriteFile(outputFile, []byte("data"), 0o000); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if err := ingestor.Parse(validFilePath, outputFile, false); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	// cleanup
	os.Remove(outputFile)
}
