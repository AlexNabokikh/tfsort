package tsort_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlexNabokikh/tfsort/tsort"
)

const (
	testDataBaseDir  = "testdata"
	validFilePath    = "testdata/valid.tf"
	validTofuPath    = "testdata/valid.tofu"
	expectedTfPath   = "testdata/expected.tf"
	expectedTofuPath = "testdata/expected.tofu"
	outputFile       = "output.tf"
)

func setupTestDir(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(testDataBaseDir); os.IsNotExist(err) {
		err = os.Mkdir(testDataBaseDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create testdata directory: %v", err)
		}
	}
	for _, fname := range []string{validFilePath, validTofuPath, expectedTfPath, expectedTofuPath} {
		info, err := os.Stat(fname)
		if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
			content := []byte(`variable "placeholder" {}`)
			if err := os.WriteFile(fname, content, 0644); err != nil {
				t.Fatalf("Failed to create placeholder test file %s: %v", fname, err)
			}
		} else if err != nil {
			t.Fatalf("Failed to stat test file %s: %v", fname, err)
		}
	}
}

func cleanupTestFiles(t *testing.T, files ...string) {
	t.Helper()
	for _, file := range files {
		os.Chmod(file, 0644)
		os.Remove(file)
	}
}

func TestCanIngest(t *testing.T) {
	setupTestDir(t)
	ingestor := tsort.NewIngestor()

	invalidExtFile := filepath.Join(testDataBaseDir, "invalid_file.txt")

	defer cleanupTestFiles(t, invalidExtFile)

	t.Run("Valid Terraform File", func(t *testing.T) {
		if err := ingestor.CanIngest(validFilePath); err != nil {
			t.Errorf("Unexpected error for valid .tf file: %v", err)
		}
	})

	t.Run("Valid OpenTofu File", func(t *testing.T) {
		if err := ingestor.CanIngest(validTofuPath); err != nil {
			t.Errorf("Unexpected error for valid .tofu file: %v", err)
		}
	})

	t.Run("File not exists", func(t *testing.T) {
		if err := ingestor.CanIngest("nonExistentFile.tf"); err == nil {
			t.Error("Expected error for non-existent file but got nil")
		} else if !strings.Contains(err.Error(), "no such file or directory") {
			t.Errorf("Expected 'no such file' error, but got: %v", err)
		}
	})

	t.Run("Invalid File Type", func(t *testing.T) {
		if err := os.WriteFile(invalidExtFile, []byte("data"), 0644); err != nil {
			t.Fatalf("Failed to create invalid extension file: %v", err)
		}
		if err := ingestor.CanIngest(invalidExtFile); err == nil {
			t.Error("Expected error for invalid file type (.txt) but got nil")
		} else if !strings.Contains(err.Error(), "not a supported") {
			t.Errorf("Expected 'not supported' error, but got: %v", err)
		}
	})
}

func TestParse(t *testing.T) {
	setupTestDir(t)
	ingestor := tsort.NewIngestor()
	invalidHclFile := filepath.Join(testDataBaseDir, "invalid_syntax.tf")
	unwritableOutputFile := filepath.Join(testDataBaseDir, "unwritable_output.tf")
	unreadableInputFile := filepath.Join(testDataBaseDir, "unreadable_input.tf")

	defer cleanupTestFiles(t, outputFile, invalidHclFile, unwritableOutputFile, unreadableInputFile)

	t.Run("File does not exist", func(t *testing.T) {
		err := ingestor.Parse("nonExistentFile.tf", outputFile, false)
		if err == nil {
			t.Error("Expected error for non-existent file but got nil")
		} else if !strings.Contains(err.Error(), "no such file or directory") {
			t.Errorf("Expected 'no such file' error, but got: %v", err)
		}
	})

	t.Run("Input file read error", func(t *testing.T) {
		if err := os.WriteFile(unreadableInputFile, []byte(`variable "a" {}`), 0644); err != nil {
			t.Fatalf("Failed to create file for read error test: %v", err)
		}
		if err := os.Chmod(unreadableInputFile, 0000); err != nil {
			t.Logf("Warning: Could not set input file permissions to 0000: %v", err)
			_, readErr := os.ReadFile(unreadableInputFile)
			if readErr == nil {
				t.Skipf("Skipping read error test: unable to make file %s unreadable by owner", unreadableInputFile)
			}
		}

		err := ingestor.Parse(unreadableInputFile, outputFile, false)
		if err == nil {
			os.Chmod(unreadableInputFile, 0644)
			t.Errorf("Expected error when reading input file with permissions 0000 but got nil")
		} else if !strings.Contains(err.Error(), "error reading file") {
			os.Chmod(unreadableInputFile, 0644)
			t.Errorf("Expected 'error reading file' error, but got: %v", err)
		} else {
			os.Chmod(unreadableInputFile, 0644)
		}
	})

	t.Run("Invalid HCL Syntax", func(t *testing.T) {
		invalidContent := []byte(`variable "a" { type = string`)
		if err := os.WriteFile(invalidHclFile, invalidContent, 0644); err != nil {
			t.Fatalf("Failed to create invalid HCL file: %v", err)
		}

		err := ingestor.Parse(invalidHclFile, outputFile, false)
		if err == nil {
			t.Error("Expected error for invalid HCL syntax but got nil")
		} else if !strings.Contains(err.Error(), "error parsing HCL") {
			t.Errorf("Expected HCL parsing error, but got: %v", err)
		}
	})

	t.Run("Write to output file (.tf)", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		if err := ingestor.Parse(validFilePath, outputFile, false); err != nil {
			t.Fatalf("Parse failed unexpectedly: %v", err)
		}

		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Fatal("Output file was not created")
		}

		outFileBytes, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output file %s: %v", outputFile, err)
		}
		expectedFileBytes, err := os.ReadFile(expectedTfPath)
		if err != nil {
			t.Fatalf("Failed to read expected file %s: %v", expectedTfPath, err)
		}

		if string(outFileBytes) != string(expectedFileBytes) {
			t.Errorf("Output file content mismatch.\nExpected:\n%s\nGot:\n%s", string(expectedFileBytes), string(outFileBytes))
		}
	})

	t.Run("Write to output file (.tofu)", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		if err := ingestor.Parse(validTofuPath, outputFile, false); err != nil {
			t.Fatalf("Parse failed unexpectedly for .tofu file: %v", err)
		}

		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Fatal("Output file was not created for .tofu input")
		}

		outFileBytes, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output file %s: %v", outputFile, err)
		}
		expectedFileBytes, err := os.ReadFile(expectedTofuPath)
		if err != nil {
			t.Fatalf("Failed to read expected file %s: %v", expectedTofuPath, err)
		}

		if string(outFileBytes) != string(expectedFileBytes) {
			t.Errorf("Output file content mismatch for .tofu.\nExpected:\n%s\nGot:\n%s", string(expectedFileBytes), string(outFileBytes))
		}
	})

	t.Run("Write to stdout (dry run)", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		if err := ingestor.Parse(validFilePath, "", true); err != nil {
			t.Fatalf("Parse failed unexpectedly during dry run: %v", err)
		}

		if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
			t.Error("Output file was created during dry run, but should not have been")
		}
	})

	t.Run("Overwrite input file", func(t *testing.T) {
		tempInputFile := filepath.Join(testDataBaseDir, "temp_overwrite.tf")
		validContent, err := os.ReadFile(validFilePath)
		if err != nil {
			t.Fatalf("Failed to read valid file for copy: %v", err)
		}
		if err := os.WriteFile(tempInputFile, validContent, 0644); err != nil {
			t.Fatalf("Failed to create temp input file: %v", err)
		}
		defer cleanupTestFiles(t, tempInputFile)

		if err := ingestor.Parse(tempInputFile, "", false); err != nil {
			t.Fatalf("Parse failed unexpectedly during overwrite: %v", err)
		}

		modifiedBytes, err := os.ReadFile(tempInputFile)
		if err != nil {
			t.Fatalf("Failed to read modified input file %s: %v", tempInputFile, err)
		}
		expectedBytes, err := os.ReadFile(expectedTfPath)
		if err != nil {
			t.Fatalf("Failed to read expected file %s: %v", expectedTfPath, err)
		}

		if string(modifiedBytes) != string(expectedBytes) {
			t.Errorf("Overwritten input file content mismatch.\nExpected:\n%s\nGot:\n%s", string(expectedBytes), string(modifiedBytes))
		}
	})

	t.Run("Error writing to output file (permissions)", func(t *testing.T) {
		if f, err := os.Create(unwritableOutputFile); err == nil {
			f.Close()
			if err := os.Chmod(unwritableOutputFile, 0444); err != nil {
				t.Logf("Warning: Could not set output file to read-only, test might not be effective: %v", err)
			}
		} else {
			t.Fatalf("Failed to create dummy output file for permissions test: %v", err)
		}

		err := ingestor.Parse(validFilePath, unwritableOutputFile, false)
		if err == nil {
			cleanupTestFiles(t, unwritableOutputFile)
			t.Error("Expected error when writing to read-only output file but got nil")
		} else if !strings.Contains(err.Error(), "error writing output") {
			cleanupTestFiles(t, unwritableOutputFile)
			t.Errorf("Expected 'error writing output' error, but got: %v", err)
		} else {
			cleanupTestFiles(t, unwritableOutputFile)
		}
	})
}

func TestValidateFilePath(t *testing.T) {
	setupTestDir(t)
	if _, err := os.Stat(validFilePath); os.IsNotExist(err) {
		os.WriteFile(validFilePath, []byte(`variable "a" {}`), 0644)
	}

	t.Run("File path is empty", func(t *testing.T) {
		if err := tsort.ValidateFilePath(""); err == nil {
			t.Error("Expected error for empty file path but got nil")
		} else if err.Error() != "file path is required" {
			t.Errorf("Expected 'file path is required' error, but got: %v", err)
		}
	})

	t.Run("File not exists", func(t *testing.T) {
		if err := tsort.ValidateFilePath("nonExistentFile.tf"); err == nil {
			t.Error("Expected error for non-existent file but got nil")
		} else if err.Error() != "file does not exist" {
			t.Errorf("Expected 'file does not exist' error, but got: %v", err)
		}
	})

	t.Run("Path is directory", func(t *testing.T) {
		if err := tsort.ValidateFilePath(testDataBaseDir); err == nil {
			t.Errorf("Expected error when path is a directory but got nil")
		} else if err.Error() != "path is a directory, not a file" {
			t.Errorf("Expected 'path is a directory' error, but got: %v", err)
		}
	})

	t.Run("Valid File Path", func(t *testing.T) {
		if err := tsort.ValidateFilePath(validFilePath); err != nil {
			t.Errorf("Unexpected error for valid file path: %v", err)
		}
	})
}
