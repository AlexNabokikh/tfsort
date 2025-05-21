package hclsort_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlexNabokikh/tfsort/internal/hclsort"
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
}

func cleanupTestFiles(t *testing.T, files ...string) {
	t.Helper()
	for _, file := range files {
		_ = os.Chmod(file, 0600)
		_ = os.Remove(file)
	}
}

func mockStdin(t *testing.T, content string) func() {
	t.Helper()
	originalStdin := os.Stdin
	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		t.Fatalf("Failed to create pipe for stdin mock: %v", errPipe)
	}

	go func() {
		defer w.Close()
		_, errWrite := w.WriteString(content)
		if errWrite != nil {
			t.Logf("Error writing to stdin mock: %v", errWrite)
		}
	}()

	os.Stdin = r //nolint:reassign //common pattern for mocking standard I/O in tests

	return func() {
		os.Stdin = originalStdin //nolint:reassign //common pattern for mocking standard I/O in tests
		if errClose := r.Close(); errClose != nil {
			t.Logf("Error closing mocked stdin reader: %v", errClose)
		}
	}
}

func captureOutput(t *testing.T, action func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		t.Fatalf("Failed to create pipe for stdout capture: %v", errPipe)
	}
	os.Stdout = w //nolint:reassign //common pattern for mocking standard I/O in tests

	action()

	if errClose := w.Close(); errClose != nil {
		t.Logf("Warning: failed to close writer for stdout capture: %v", errClose)
	}
	os.Stdout = oldStdout //nolint:reassign //common pattern for mocking standard I/O in tests

	outBytes, errRead := io.ReadAll(r)
	if errRead != nil {
		t.Fatalf("Failed to read from stdout capture pipe: %v", errRead)
	}
	if errClose := r.Close(); errClose != nil {
		t.Logf("Warning: failed to close reader for stdout capture: %v", errClose)
	}
	return string(outBytes)
}

func TestCheckFileExtension(t *testing.T) {
	setupTestDir(t)

	ingestor := hclsort.NewIngestor()
	allowedTypes := ingestor.AllowedTypes

	invalidExtFile := filepath.Join(testDataBaseDir, "invalid_file.txt")
	if err := os.WriteFile(invalidExtFile, []byte("data"), 0600); err != nil {
		t.Fatalf("Failed to create invalid extension file: %v", err)
	}
	defer cleanupTestFiles(t, invalidExtFile)

	t.Run("Valid Terraform File Path", func(t *testing.T) {
		if err := hclsort.CheckFileExtension(validFilePath, allowedTypes); err != nil {
			t.Errorf("Unexpected error for valid .tf file path: %v", err)
		}
	})

	t.Run("Valid OpenTofu File Path", func(t *testing.T) {
		if err := hclsort.CheckFileExtension(validTofuPath, allowedTypes); err != nil {
			t.Errorf("Unexpected error for valid .tofu file path: %v", err)
		}
	})

	t.Run("Path with valid extension (file existence not checked)", func(t *testing.T) {
		err := hclsort.CheckFileExtension("nonExistentFile.tf", allowedTypes)
		if err != nil {
			t.Errorf(
				"Expected no error for a path with a valid extension (.tf), but got: %v",
				err,
			)
		}
	})

	t.Run("Path with unsupported extension", func(t *testing.T) {
		err := hclsort.CheckFileExtension(invalidExtFile, allowedTypes)
		if err == nil {
			t.Error(
				"Expected error for unsupported file type (.txt) but got nil",
			)
		} else if !strings.Contains(
			err.Error(),
			"not a supported Terraform/HCL type",
		) {
			t.Errorf(
				"Expected 'not a supported Terraform/HCL type' error, but got: %v",
				err,
			)
		}
	})

	t.Run("Path with empty extension", func(t *testing.T) {
		if err := hclsort.CheckFileExtension("fileWithNoExtension", allowedTypes); err != nil {
			t.Errorf(
				"Expected no error for a path with no extension, but got: %v",
				err,
			)
		}
	})

	t.Run("Path with only a dot (treated as no extension)", func(t *testing.T) {
		if err := hclsort.CheckFileExtension(".", allowedTypes); err != nil {
			t.Errorf(
				"Expected no error for a path that is just a dot ('.'), but got: %v",
				err,
			)
		}
	})
}

//gocyclo:ignore
func TestParse(t *testing.T) {
	setupTestDir(t)

	validTestContentBytes, errReadFile := os.ReadFile(validFilePath)
	if errReadFile != nil {
		t.Fatalf("Failed to read %s: %v. Ensure it exists and contains unsorted HCL.", validFilePath, errReadFile)
	}
	validTestContentForStdin := string(validTestContentBytes)

	expectedSortedTestContentBytes, errReadFile := os.ReadFile(expectedTfPath)
	if errReadFile != nil {
		t.Fatalf("Failed to read %s: %v. Ensure it exists and contains sorted HCL.", expectedTfPath, errReadFile)
	}
	expectedSortedTestContent := string(expectedSortedTestContentBytes)
	normalizedExpectedSortedContent := strings.TrimSpace(expectedSortedTestContent) + "\n"

	expectedSortedTofuContentBytes, errReadFile := os.ReadFile(expectedTofuPath)
	if errReadFile != nil {
		t.Fatalf("Failed to read %s: %v. Ensure it exists and contains sorted HCL.", expectedTofuPath, errReadFile)
	}
	normalizedExpectedSortedTofuContent := strings.TrimSpace(string(expectedSortedTofuContentBytes)) + "\n"

	ingestor := hclsort.NewIngestor()
	invalidHclFile := filepath.Join(testDataBaseDir, "invalid_syntax.tf")
	unwritableOutputFile := filepath.Join(testDataBaseDir, "unwritable_output.tf")
	unreadableInputFile := filepath.Join(testDataBaseDir, "unreadable_input.tf")

	defer cleanupTestFiles(
		t,
		outputFile,
		invalidHclFile,
		unwritableOutputFile,
		unreadableInputFile,
	)

	t.Run("File does not exist", func(t *testing.T) {
		err := ingestor.Parse("nonExistentFile.tf", outputFile, false, false)
		if err == nil {
			t.Error("Expected error for non-existent file but got nil")
		} else if !strings.Contains(err.Error(), "no such file or directory") &&
			!strings.Contains(err.Error(), "error reading file") {
			t.Errorf(
				"Expected 'no such file' or 'error reading file' error, but got: %v",
				err,
			)
		}
	})

	t.Run("Input file read error", func(t *testing.T) {
		if err := os.WriteFile(unreadableInputFile, []byte(`variable "a" {}`), 0600); err != nil {
			t.Fatalf("Failed to create file for read error test: %v", err)
		}

		errChmod := os.Chmod(unreadableInputFile, 0000)
		if errChmod != nil {
			t.Logf("Warning: Could not set input file permissions to 0000: %v", errChmod)
			_, readErrAttempt := os.ReadFile(unreadableInputFile)
			if readErrAttempt == nil {
				_ = os.Chmod(unreadableInputFile, 0600)
				t.Skipf("Skipping read error test: unable to make file %s unreadable by owner", unreadableInputFile)
			}
		}
		defer func() { _ = os.Chmod(unreadableInputFile, 0600) }()

		errParse := ingestor.Parse(unreadableInputFile, outputFile, false, false)
		switch {
		case errParse == nil:
			t.Errorf("Expected error when reading input file with permissions 0000 but got nil")
		case !strings.Contains(errParse.Error(), "error reading file"):
			t.Errorf("Expected 'error reading file' error, but got: %v", errParse)
		}
	})

	t.Run("Invalid HCL Syntax from file", func(t *testing.T) {
		invalidContent := []byte(`variable "a" { type = string`)
		if err := os.WriteFile(invalidHclFile, invalidContent, 0600); err != nil {
			t.Fatalf("Failed to create invalid HCL file: %v", err)
		}

		errParse := ingestor.Parse(invalidHclFile, outputFile, false, false)
		if errParse == nil {
			t.Error("Expected error for invalid HCL syntax but got nil")
		} else if !strings.Contains(errParse.Error(), "error parsing HCL content") {
			t.Errorf("Expected HCL parsing error, but got: %v", errParse)
		}
	})

	t.Run("Write to output file (.tf)", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		if err := ingestor.Parse(validFilePath, outputFile, false, false); err != nil {
			t.Fatalf("Parse failed unexpectedly: %v", err)
		}

		if _, errStat := os.Stat(outputFile); os.IsNotExist(errStat) {
			t.Fatal("Output file was not created")
		}

		outFileBytes, errRead := os.ReadFile(outputFile)
		if errRead != nil {
			t.Fatalf("Failed to read output file %s: %v", outputFile, errRead)
		}

		if string(outFileBytes) != normalizedExpectedSortedContent {
			t.Errorf("Output file content mismatch.\nExpected:\n%s\nGot:\n%s",
				normalizedExpectedSortedContent, string(outFileBytes))
		}
	})

	t.Run("Write to output file (.tofu)", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		if err := ingestor.Parse(validTofuPath, outputFile, false, false); err != nil {
			t.Fatalf("Parse failed unexpectedly for .tofu file: %v", err)
		}

		if _, errStat := os.Stat(outputFile); os.IsNotExist(errStat) {
			t.Fatal("Output file was not created for .tofu input")
		}

		outFileBytes, errRead := os.ReadFile(outputFile)
		if errRead != nil {
			t.Fatalf("Failed to read output file %s: %v", outputFile, errRead)
		}

		if string(outFileBytes) != normalizedExpectedSortedTofuContent {
			t.Errorf("Output file content mismatch for .tofu.\nExpected:\n%s\nGot:\n%s",
				normalizedExpectedSortedTofuContent, string(outFileBytes))
		}
	})

	t.Run("Write to stdout (dry run from file)", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		var parseErr error
		capturedStdout := captureOutput(t, func() {
			parseErr = ingestor.Parse(validFilePath, "", true, false)
		})

		if parseErr != nil {
			t.Fatalf("Parse failed unexpectedly during dry run from file: %v", parseErr)
		}

		if capturedStdout != normalizedExpectedSortedContent {
			t.Errorf("Dry run output mismatch for file input.\nExpected:\n%s\nGot:\n%s",
				normalizedExpectedSortedContent, capturedStdout)
		}

		if _, errStat := os.Stat(outputFile); !os.IsNotExist(errStat) {
			t.Error("Output file was created during dry run from file, but should not have been")
		}
	})

	t.Run("Overwrite input file", func(t *testing.T) {
		tempInputFile := filepath.Join(testDataBaseDir, "temp_overwrite.tf")
		if err := os.WriteFile(tempInputFile, validTestContentBytes, 0600); err != nil {
			t.Fatalf("Failed to create temp input file: %v", err)
		}
		defer cleanupTestFiles(t, tempInputFile)

		if err := ingestor.Parse(tempInputFile, "", false, false); err != nil {
			t.Fatalf("Parse failed unexpectedly during overwrite: %v", err)
		}

		modifiedBytes, errRead := os.ReadFile(tempInputFile)
		if errRead != nil {
			t.Fatalf("Failed to read modified input file %s: %v", tempInputFile, errRead)
		}

		if string(modifiedBytes) != normalizedExpectedSortedContent {
			t.Errorf("Overwritten input file content mismatch.\nExpected:\n%s\nGot:\n%s",
				normalizedExpectedSortedContent, string(modifiedBytes))
		}
	})

	t.Run("Error writing to output file (permissions)", func(t *testing.T) {
		if f, errCreate := os.Create(unwritableOutputFile); errCreate == nil {
			_ = f.Close()
			if errChmod := os.Chmod(unwritableOutputFile, 0444); errChmod != nil {
				t.Logf("Warning: Could not set output file to read-only, test might not be effective: %v", errChmod)
			}
		} else {
			t.Fatalf("Failed to create dummy output file for permissions test: %v", errCreate)
		}
		defer func() { _ = os.Chmod(unwritableOutputFile, 0600) }()

		errParse := ingestor.Parse(validFilePath, unwritableOutputFile, false, false)
		switch {
		case errParse == nil:
			t.Error("Expected error when writing to read-only output file but got nil")
		case !strings.Contains(errParse.Error(), "error writing output"):
			t.Errorf("Expected 'error writing output' error, but got: %v", errParse)
		}
	})

	t.Run("Read from stdin, write to stdout", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		restoreStdin := mockStdin(t, validTestContentForStdin)
		defer restoreStdin()

		var parseErr error
		capturedStdout := captureOutput(t, func() {
			parseErr = ingestor.Parse(hclsort.StdInPathIdentifier, "", false, true)
		})

		if parseErr != nil {
			t.Fatalf("Parse from stdin to stdout failed unexpectedly: %v", parseErr)
		}

		if capturedStdout != normalizedExpectedSortedContent {
			t.Errorf("Output to stdout from stdin mismatch.\nExpected:\n%s\nGot:\n%s",
				normalizedExpectedSortedContent, capturedStdout)
		}

		if _, errStat := os.Stat(outputFile); !os.IsNotExist(errStat) {
			t.Error("Output file was created during stdin to stdout test, but should not have been")
		}
	})

	t.Run("Read from stdin, write to output file", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		restoreStdin := mockStdin(t, validTestContentForStdin)
		defer restoreStdin()

		err := ingestor.Parse(hclsort.StdInPathIdentifier, outputFile, false, true)
		if err != nil {
			t.Fatalf("Parse from stdin to output file failed unexpectedly: %v", err)
		}

		if _, errStat := os.Stat(outputFile); os.IsNotExist(errStat) {
			t.Fatal("Output file was not created when parsing from stdin with -o")
		}

		outFileBytes, errRead := os.ReadFile(outputFile)
		if errRead != nil {
			t.Fatalf("Failed to read output file %s: %v", outputFile, errRead)
		}

		if string(outFileBytes) != normalizedExpectedSortedContent {
			t.Errorf("Output file content mismatch from stdin.\nExpected:\n%s\nGot:\n%s",
				normalizedExpectedSortedContent, string(outFileBytes))
		}
	})

	t.Run("Read from stdin, dry run to stdout", func(t *testing.T) {
		cleanupTestFiles(t, outputFile)

		restoreStdin := mockStdin(t, validTestContentForStdin)
		defer restoreStdin()

		var parseErr error
		capturedStdout := captureOutput(t, func() {
			parseErr = ingestor.Parse(hclsort.StdInPathIdentifier, "", true, true)
		})

		if parseErr != nil {
			t.Fatalf("Parse from stdin with dry-run failed unexpectedly: %v", parseErr)
		}

		if capturedStdout != normalizedExpectedSortedContent {
			t.Errorf("Dry run output to stdout from stdin mismatch.\nExpected:\n%s\nGot:\n%s",
				normalizedExpectedSortedContent, capturedStdout)
		}
		if _, errStat := os.Stat(outputFile); !os.IsNotExist(errStat) {
			t.Error("Output file was created during stdin dry run, but should not have been")
		}
	})

	t.Run("Invalid HCL from stdin", func(t *testing.T) {
		invalidContent := `variable "a" { type = string`
		restoreStdin := mockStdin(t, invalidContent)
		defer restoreStdin()

		err := ingestor.Parse(hclsort.StdInPathIdentifier, outputFile, false, true)
		if err == nil {
			t.Error("Expected error for invalid HCL from stdin but got nil")
		} else {
			if !strings.Contains(err.Error(), "error parsing HCL content") {
				t.Errorf("Expected HCL parsing error from stdin, but got: %v", err)
			}
			if !strings.Contains(err.Error(), hclsort.StdInPathIdentifier) {
				t.Errorf("Expected error message for stdin to contain '%s', but got: %v", hclsort.StdInPathIdentifier, err.Error())
			}
		}
	})
}

func TestValidateFilePath(t *testing.T) {
	setupTestDir(t)

	if _, statErr := os.Stat(validFilePath); os.IsNotExist(statErr) {
		if writeErr := os.WriteFile(validFilePath, []byte(`variable "a" {}`), 0600); writeErr != nil {
			t.Fatalf("Failed to create %s for TestValidateFilePath: %v", validFilePath, writeErr)
		}
	}

	t.Run("File path is empty", func(t *testing.T) {
		if err := hclsort.ValidateFilePath(""); err == nil {
			t.Error("Expected error for empty file path but got nil")
		} else if err.Error() != "file path is required" {
			t.Errorf("Expected 'file path is required' error, but got: %v", err)
		}
	})

	t.Run("File not exists", func(t *testing.T) {
		if err := hclsort.ValidateFilePath("nonExistentFile.tf"); err == nil {
			t.Error("Expected error for non-existent file but got nil")
		} else if err.Error() != "file does not exist" {
			t.Errorf("Expected 'file does not exist' error, but got: %v", err)
		}
	})

	t.Run("Path is directory", func(t *testing.T) {
		if err := hclsort.ValidateFilePath(testDataBaseDir); err == nil {
			t.Errorf("Expected error when path is a directory but got nil")
		} else if err.Error() != "path is a directory, not a file" {
			t.Errorf(
				"Expected 'path is a directory, not a file' error, but got: %v",
				err,
			)
		}
	})

	t.Run("Valid File Path", func(t *testing.T) {
		if err := hclsort.ValidateFilePath(validFilePath); err != nil {
			t.Errorf("Unexpected error for valid file path: %v", err)
		}
	})
}

func TestSortRequiredProvidersInBlock(t *testing.T) {
	const hclInput = `
terraform {
  required_providers {
    z = { source = "provider/z" }
    a = { source = "provider/a" }
  }
  required_versions = ">= 1.0"
}
`

	file, err := hclsort.ParseHCLContent([]byte(hclInput), "testfile.tf")
	if err != nil {
		t.Fatalf("ParseHCLContent failed: %v", err)
	}

	sortedFile := hclsort.ProcessAndSortBlocks(file, map[string]bool{})

	output := string(hclsort.FormatHCLBytes(sortedFile))

	idxA := strings.Index(output, "a =")
	idxZ := strings.Index(output, "z =")

	if idxA < 0 || idxZ < 0 {
		t.Fatalf("did not find both providers in output:\n%s", output)
	}
	if idxA > idxZ {
		t.Errorf("expected provider “a” to appear before “z”,\noutput was:\n%s", output)
	}
}
