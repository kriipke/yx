package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Embed the yq and delta binaries into the resulting executable.
//go:embed binaries/yq
//go:embed binaries/delta
var embeddedFiles embed.FS

func extractBinary(name string) (string, error) {
	// Read the embedded binary file
	data, err := embeddedFiles.ReadFile("binaries/" + name)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded binary: %w", err)
	}

	// Create a temporary file to store the binary
	tempDir, err := os.MkdirTemp("", "yiff_tool")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	binaryPath := filepath.Join(tempDir, name)
	if err := os.WriteFile(binaryPath, data, 0755); err != nil {
		return "", fmt.Errorf("failed to write binary to temp file: %w", err)
	}

	return binaryPath, nil
}

func runYq(yqPath, filePath string) (string, error) {
	// Run the yq command and capture the output
	cmd := exec.Command(yqPath, "-P", "sort_keys(..) | ... comments=\"\"", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yq error for file %s: %v", filePath, err)
	}
	return string(output), nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: yiff <file1> <file2>")
		os.Exit(1)
	}

	file1 := os.Args[1]
	file2 := os.Args[2]

	yqPath, err := extractBinary("yq")
	if err != nil {
		fmt.Printf("Error extracting yq binary: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(filepath.Dir(yqPath)) // Clean up temporary files

	deltaPath, err := extractBinary("delta")
	if err != nil {
		fmt.Printf("Error extracting delta binary: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(filepath.Dir(deltaPath)) // Clean up temporary files

	// Run yq on the first file
	yaml1, err := runYq(yqPath, file1)
	if err != nil {
		fmt.Printf("Error running yq on %s: %v\n", file1, err)
		os.Exit(1)
	}

	// Run yq on the second file
	yaml2, err := runYq(yqPath, file2)
	if err != nil {
		fmt.Printf("Error running yq on %s: %v\n", file2, err)
		os.Exit(1)
	}

	// Write the YAML outputs to temporary files for delta comparison
	tempFile1, err := os.CreateTemp("", "yiff_file1_*.yaml")
	if err != nil {
		fmt.Printf("Error creating temp file for file1: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempFile1.Name())

	tempFile2, err := os.CreateTemp("", "yiff_file2_*.yaml")
	if err != nil {
		fmt.Printf("Error creating temp file for file2: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempFile2.Name())

	if _, err := tempFile1.WriteString(yaml1); err != nil {
		fmt.Printf("Error writing to temp file for file1: %v\n", err)
		os.Exit(1)
	}
	if _, err := tempFile2.WriteString(yaml2); err != nil {
		fmt.Printf("Error writing to temp file for file2: %v\n", err)
		os.Exit(1)
	}

	// Run delta to compare the two temporary files
	cmd := exec.Command(deltaPath, tempFile1.Name(), tempFile2.Name(), "--features=diff-highlight", "-s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running delta: %v\n", err)
		os.Exit(1)
	}
}
