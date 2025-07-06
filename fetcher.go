package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type TestCase struct {
	Input    string
	Expected string
	Number   int
}

type TestCaseFetcher struct {
	config *Config
	auth   *CSESAuth
}

func NewTestCaseFetcher(config *Config) *TestCaseFetcher {
	return &TestCaseFetcher{
		config: config,
		auth:   NewCSESAuth(config),
	}
}

func (f *TestCaseFetcher) FetchTestCases(problemID string) ([]TestCase, error) {
	cacheDir := filepath.Join(f.config.CacheDir, problemID)

	// Check if we have cached test cases
	if testCases, err := f.loadCachedTestCases(cacheDir); err == nil && len(testCases) > 0 {
		if f.config.Verbose {
			green.Printf("📋 Using cached test cases from %s\n", cacheDir)
		}
		return testCases, nil
	}

	// Fetch from CSES
	if f.config.Verbose {
		yellow.Printf("🔍 Fetching test cases from CSES for problem %s...\n", problemID)
	}

	testCases, err := f.fetchFromCSES(problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from CSES: %w", err)
	}

	// Cache the test cases
	if err := f.cacheTestCases(cacheDir, testCases); err != nil {
		yellow.Printf("⚠️  Failed to cache test cases: %v\n", err)
	}

	return testCases, nil
}

func (f *TestCaseFetcher) fetchFromCSES(problemID string) ([]TestCase, error) {
	// Ensure we're authenticated
	if err := f.auth.EnsureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	// Get the test cases zip file
	zipData, err := f.auth.DownloadTestCases(problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to download test cases: %w", err)
	}

	// Extract and parse the zip file
	return f.extractTestCasesFromZip(zipData)
}

func (f *TestCaseFetcher) extractTestCasesFromZip(zipData []byte) ([]TestCase, error) {
	// Create a reader from the zip data
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}

	var testCases []TestCase
	inputs := make(map[int]string)
	outputs := make(map[int]string)

	// Process each file in the zip
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Open the file
		rc, err := file.Open()
		if err != nil {
			continue
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		// Parse filename to get test case number
		filename := file.Name
		if strings.HasSuffix(filename, ".in") {
			// Input file
			testNum := f.parseTestNumber(filename, ".in")
			if testNum > 0 {
				inputs[testNum] = string(content)
			}
		} else if strings.HasSuffix(filename, ".out") {
			// Output file
			testNum := f.parseTestNumber(filename, ".out")
			if testNum > 0 {
				outputs[testNum] = string(content)
			}
		}
	}

	// Create test cases from matched input/output pairs
	for testNum := range inputs {
		if output, exists := outputs[testNum]; exists {
			testCases = append(testCases, TestCase{
				Input:    inputs[testNum],
				Expected: output,
				Number:   testNum,
			})
		}
	}

	if len(testCases) == 0 {
		return nil, fmt.Errorf("no valid test cases found in zip file")
	}

	if f.config.Verbose {
		green.Printf("📦 Extracted %d test cases from zip file\n", len(testCases))
	}

	return testCases, nil
}

func (f *TestCaseFetcher) parseTestNumber(filename, suffix string) int {
	// Remove the suffix to get the base name
	base := strings.TrimSuffix(filename, suffix)

	// Handle different filename patterns
	// e.g., "1.in", "test1.in", "input1.in", etc.
	patterns := []string{
		base,                               // Direct number
		strings.TrimPrefix(base, "test"),   // Remove "test" prefix
		strings.TrimPrefix(base, "input"),  // Remove "input" prefix
		strings.TrimPrefix(base, "output"), // Remove "output" prefix
	}

	for _, pattern := range patterns {
		if num, err := strconv.Atoi(pattern); err == nil && num > 0 {
			return num
		}
	}

	return 0
}

func (f *TestCaseFetcher) loadCachedTestCases(cacheDir string) ([]TestCase, error) {
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	var testCases []TestCase
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".in") {
			number := strings.TrimSuffix(file.Name(), ".in")

			inputPath := filepath.Join(cacheDir, file.Name())
			outputPath := filepath.Join(cacheDir, number+".out")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				continue
			}

			output, err := os.ReadFile(outputPath)
			if err != nil {
				continue
			}

			testNum, _ := strconv.Atoi(number)
			testCases = append(testCases, TestCase{
				Input:    string(input),
				Expected: string(output),
				Number:   testNum,
			})
		}
	}

	return testCases, nil
}

func (f *TestCaseFetcher) cacheTestCases(cacheDir string, testCases []TestCase) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	for _, testCase := range testCases {
		inputPath := filepath.Join(cacheDir, fmt.Sprintf("%d.in", testCase.Number))
		outputPath := filepath.Join(cacheDir, fmt.Sprintf("%d.out", testCase.Number))

		if err := os.WriteFile(inputPath, []byte(testCase.Input), 0644); err != nil {
			return err
		}

		if err := os.WriteFile(outputPath, []byte(testCase.Expected), 0644); err != nil {
			return err
		}
	}

	if f.config.Verbose {
		green.Printf("💾 Cached %d test cases to %s\n", len(testCases), cacheDir)
	}

	return nil
}
