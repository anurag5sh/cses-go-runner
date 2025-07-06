package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type TestRunner struct {
	config   *Config
	compiler *GoCompiler
	fetcher  *TestCaseFetcher
	executor *TestExecutor
	auth     *CSESAuth
}

func NewTestRunner(config *Config) *TestRunner {
	return &TestRunner{
		config:   config,
		compiler: NewGoCompiler(config),
		fetcher:  NewTestCaseFetcher(config),
		executor: NewTestExecutor(config),
		auth:     NewCSESAuth(config),
	}
}

func (r *TestRunner) Run() error {
	// Create cache directory
	if err := os.MkdirAll(r.config.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Ensure authentication
	if err := r.auth.EnsureAuthenticated(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Validate Go installation
	if err := r.compiler.ValidateGo(); err != nil {
		return fmt.Errorf("Go validation failed: %w", err)
	}

	// Check Go code syntax
	if err := r.compiler.ValidateSyntax(); err != nil {
		return fmt.Errorf("syntax validation failed: %w", err)
	}

	// Fetch test cases
	yellow.Println("📥 Fetching test cases from CSES...")
	testCases, err := r.fetcher.FetchTestCases(r.config.ProblemID)
	if err != nil {
		return fmt.Errorf("failed to fetch test cases: %w", err)
	}

	if len(testCases) == 0 {
		yellow.Println("⚠️  No test cases found for this problem")
		return nil
	}

	green.Printf("✅ Found %d test cases\n", len(testCases))

	// Compile solution
	yellow.Println("🔨 Compiling Go solution...")
	executablePath, err := r.compiler.Compile()
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}
	defer os.Remove(executablePath) // Clean up

	green.Println("✅ Compilation successful")

	// Run tests
	results := r.runTests(executablePath, testCases)

	// Display results
	r.displayResults(results)

	return nil
}

func (r *TestRunner) runTests(executablePath string, testCases []TestCase) []TestResult {
	results := make([]TestResult, len(testCases))

	// Create a semaphore to limit parallel execution
	semaphore := make(chan struct{}, r.config.Parallel)
	var wg sync.WaitGroup

	yellow.Printf("🧪 Running %d test cases (parallel: %d)...\n", len(testCases), r.config.Parallel)

	startTime := time.Now()
	progressChan := make(chan int, len(testCases))

	// Progress reporter
	go func() {
		completed := 0
		for range progressChan {
			completed++
			if r.config.Verbose {
				cyan.Printf("📊 Progress: %d/%d test cases completed\n", completed, len(testCases))
			}
		}
	}()

	for i, testCase := range testCases {
		wg.Add(1)
		go func(index int, tc TestCase) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx, cancel := context.WithTimeout(context.Background(), r.config.GetTimeout())
			defer cancel()

			result := r.executor.Execute(ctx, executablePath, tc, index+1)
			results[index] = result

			if r.config.Verbose {
				if result.Passed {
					green.Printf("✅ Test %d passed (%.2fms)\n", index+1, result.Duration.Seconds()*1000)
				} else {
					red.Printf("❌ Test %d failed: %s (%.2fms)\n", index+1, result.Error, result.Duration.Seconds()*1000)
				}
			}

			progressChan <- 1
		}(i, testCase)
	}

	wg.Wait()
	close(progressChan)

	totalTime := time.Since(startTime)
	cyan.Printf("⏱️  Total execution time: %.2fs\n", totalTime.Seconds())

	return results
}

func (r *TestRunner) displayResults(results []TestResult) {
	passed := 0
	failed := 0
	var failedTests []TestResult
	var totalTime time.Duration

	for _, result := range results {
		totalTime += result.Duration
		if result.Passed {
			passed++
		} else {
			failed++
			failedTests = append(failedTests, result)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	white.Printf("📊 TEST RESULTS SUMMARY\n")
	fmt.Println(strings.Repeat("=", 60))

	if passed > 0 {
		green.Printf("✅ PASSED: %d/%d tests\n", passed, len(results))
	}
	if failed > 0 {
		red.Printf("❌ FAILED: %d/%d tests\n", failed, len(results))
	}

	cyan.Printf("⏱️  Average execution time: %.2fms\n", totalTime.Seconds()*1000/float64(len(results)))

	if len(failedTests) > 0 {
		fmt.Println("\n" + strings.Repeat("-", 40))
		red.Printf("❌ FAILED TEST CASES:\n")
		fmt.Println(strings.Repeat("-", 40))

		for _, result := range failedTests {
			r.displayFailedTest(result)
		}
	}

	// Overall result
	fmt.Println("\n" + strings.Repeat("=", 60))
	if failed == 0 {
		green.Printf("🎉 ALL TESTS PASSED! 🎉\n")
	} else {
		red.Printf("💥 %d TEST(S) FAILED\n", failed)
	}
	fmt.Println(strings.Repeat("=", 60))
}

func (r *TestRunner) displayFailedTest(result TestResult) {
	fmt.Printf("\n📍 Test Case %d:\n", result.TestNumber)
	fmt.Printf("   📁 Input file: %s\n", result.InputFile)
	fmt.Printf("   📁 Expected file: %s\n", result.ExpectedFile)
	fmt.Printf("   ⏱️  Duration: %.2fms\n", result.Duration.Seconds()*1000)
	fmt.Printf("   ❌ Error: %s\n", result.Error)

	if r.config.ShowDiff && result.ActualOutput != "" {
		fmt.Printf("   📤 Expected output (truncated to %d chars):\n", r.config.MaxOutput)
		expectedOutput := result.ExpectedOutput
		if len(expectedOutput) > r.config.MaxOutput {
			expectedOutput = expectedOutput[:r.config.MaxOutput] + "..."
		}
		green.Printf("   %s\n", strings.ReplaceAll(expectedOutput, "\n", "\n   "))

		fmt.Printf("   📥 Actual output (truncated to %d chars):\n", r.config.MaxOutput)
		actualOutput := result.ActualOutput
		if len(actualOutput) > r.config.MaxOutput {
			actualOutput = actualOutput[:r.config.MaxOutput] + "..."
		}
		red.Printf("   %s\n", strings.ReplaceAll(actualOutput, "\n", "\n   "))
	}
}
