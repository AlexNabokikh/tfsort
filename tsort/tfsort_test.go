package tsort_test

import (
	"fmt"
	"os"
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
		if err := os.WriteFile("invalid_file.txt", []byte("data"), 0o600); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if err := ingestor.CanIngest("invalid_file.txt"); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	t.Run("File with read error", func(t *testing.T) {
		if err := os.WriteFile("unreadable_file.tf", []byte("data"), 0o000); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if err := ingestor.CanIngest("unreadable_file.tf"); err == nil {
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
	os.Remove("unreadable_file.tf")
	os.Remove("invalid_file.txt")
}

func TestParse(t *testing.T) {
	ingestor := tsort.NewIngestor()

	t.Run("Can't ingest", func(t *testing.T) {
		if err := ingestor.Parse("notExistFile.tf", outputFile, false); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	t.Run("Write to output file", func(t *testing.T) {
		os.Remove(outputFile)
		if err := ingestor.Parse(validFilePath, outputFile, false); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file not created")
		}

		outFile, _ := os.ReadFile(outputFile)
		expectedFile, _ := os.ReadFile("testdata/expected.tf")

		if string(outFile) != string(expectedFile) {
			t.Errorf("Output file content is not as expected")
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

	// cleanup
	os.Remove(outputFile)
}

func TestParseAll(t *testing.T) {
	ingestor := tsort.NewIngestor()

	// Save original content of the files
	originalContent, err := os.ReadFile("testdata/valid.tf")
	if err != nil {
		t.Fatalf("Failed to read original content: %v", err)
	}

	t.Run("Valid Directory", func(t *testing.T) {
		if err := ingestor.ParseAll("testdata/recursive", false); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		for _, file := range []string{"valid.tf", "valid1.tf", "valid2.tf"} {
			filePath := fmt.Sprintf("testdata/recursive/%s", file)
			expectedFile, _ := os.ReadFile("testdata/expected.tf")
			outFile, _ := os.ReadFile(filePath)

			if string(outFile) != string(expectedFile) {
				t.Errorf("Output file content in '%s' is not as expected", filePath)
			}
		}
	})

	t.Run("Write to stdout", func(t *testing.T) {
		if err := ingestor.ParseAll("testdata/recursive", true); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Error accessing file", func(t *testing.T) {
		if err := ingestor.ParseAll("nonexistent_directory", false); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	// cleanup
	for _, file := range []string{"valid.tf", "valid1.tf", "valid2.tf"} {
		filePath := fmt.Sprintf("testdata/recursive/%s", file)
		if err := os.WriteFile(filePath, originalContent, 0o644); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestValidateFilePath(t *testing.T) {
	t.Run("File path is empty", func(t *testing.T) {
		if err := tsort.ValidateFilePath(""); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	t.Run("File not exists", func(t *testing.T) {
		if err := tsort.ValidateFilePath("notExistFile.tf"); err == nil {
			t.Errorf("Expected error but not occurred")
		}
	})

	t.Run("Valid File Path", func(t *testing.T) {
		if err := tsort.ValidateFilePath(validFilePath); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// cleanup
	os.Remove("invalid_file.txt")
}
