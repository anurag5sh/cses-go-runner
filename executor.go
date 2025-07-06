package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TestResult struct {
	TestNumber     int
	Passed         bool
	Error          string
	Duration       time.Duration
	ActualOutput   string
	ExpectedOutput string
	InputFile      string
	ExpectedFile   string
	MemoryUsage    string
	ExitCode       int
}

type TestExecutor struct {
	config *Config
}

func NewTestExecutor(config *Config) *TestExecutor {
	return &TestExecutor{config: config}
}

func (e *TestExecutor) Execute(ctx context.Context, executablePath string, testCase TestCase, testNumber int) TestResult {
	startTime := time.Now()

	result := TestResult{
		TestNumber:     testNumber,
		ExpectedOutput: testCase.Expected,
		InputFile:      filepath.Join(e.config.CacheDir, e.config.ProblemID, fmt.Sprintf("%d.in", testCase.Number)),
		ExpectedFile:   filepath.Join(e.config.CacheDir, e.config.ProblemID, fmt.Sprintf("%d.out", testCase.Number)),
	}

	// Execute the program
	actualOutput, exitCode, err := e.runGoProgram(ctx, executablePath, testCase.Input)
	result.Duration = time.Since(startTime)
	result.ActualOutput = actualOutput
	result.ExitCode = exitCode

	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Compare outputs
	if e.compareOutputs(actualOutput, testCase.Expected) {
		result.Passed = true
	} else {
		result.Error = "Output mismatch"
	}

	return result
}

func (e *TestExecutor) runGoProgram(ctx context.Context, executablePath, input string) (string, int, error) {
	cmd := exec.CommandContext(ctx, executablePath)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		if ctx.Err() == context.DeadlineExceeded {
			return "", exitCode, fmt.Errorf("timeout exceeded (%s)", e.config.GetTimeout())
		}

		if stderr.Len() > 0 {
			return "", exitCode, fmt.Errorf("runtime error (exit code %d): %s", exitCode, stderr.String())
		}

		return "", exitCode, fmt.Errorf("execution failed (exit code %d): %w", exitCode, err)
	}

	return stdout.String(), exitCode, nil
}

func (e *TestExecutor) compareOutputs(actual, expected string) bool {
	// Normalize whitespace
	actual = e.normalizeOutput(actual)
	expected = e.normalizeOutput(expected)

	return actual == expected
}

func (e *TestExecutor) normalizeOutput(output string) string {
	// Remove trailing whitespace from each line and normalize line endings
	lines := strings.Split(output, "\n")
	var normalizedLines []string

	for _, line := range lines {
		normalizedLines = append(normalizedLines, strings.TrimRight(line, " \t\r"))
	}

	// Join and trim final result
	result := strings.Join(normalizedLines, "\n")
	return strings.TrimSpace(result)
}
