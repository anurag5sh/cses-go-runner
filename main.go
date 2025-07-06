package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const (
	AppName    = "cses-go-runner"
	AppVersion = "1.0.0"
)

var (
	// Color definitions
	red    = color.New(color.FgRed, color.Bold)
	green  = color.New(color.FgGreen, color.Bold)
	yellow = color.New(color.FgYellow, color.Bold)
	blue   = color.New(color.FgBlue, color.Bold)
	cyan   = color.New(color.FgCyan, color.Bold)
	white  = color.New(color.FgWhite, color.Bold)
)

func printUsage() {
	fmt.Printf("%s v%s - CSES Go Solution Test Runner\n\n", AppName, AppVersion)
	fmt.Println("Usage:")
	fmt.Printf("  %s [command] [flags]\n\n", AppName)
	fmt.Println("Commands:")
	fmt.Println("  run    - Run tests for a solution (default)")
	fmt.Println("  auth   - Authenticate with CSES using environment variables")
	fmt.Println("  clean  - Clean cache directory")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  CSES_USERNAME - Your CSES username")
	fmt.Println("  CSES_PASSWORD - Your CSES password")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s auth\n", AppName)
	fmt.Printf("  %s -file=solution.go -problem=1068\n", AppName)
	fmt.Printf("  %s run -file=solution.go -problem=1068 -timeout=5s -verbose\n", AppName)
	fmt.Printf("  %s clean\n", AppName)
}

func main() {
	var (
		filePath  = flag.String("file", "", "Path to the Go solution file")
		problemID = flag.String("problem", "", "CSES problem ID")
		timeout   = flag.String("timeout", "1s", "Timeout for each test case (default: 2s)")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
		cacheDir  = flag.String("cache-dir", "~/.cache/cses-go-runner", "Directory to cache test cases")
		parallel  = flag.Int("parallel", 4, "Number of parallel test executions")
		help      = flag.Bool("help", false, "Show help message")
		version   = flag.Bool("version", false, "Show version")
		showDiff  = flag.Bool("diff", false, "Show diff for failed test cases")
		maxOutput = flag.Int("max-output", 1000, "Maximum output length to display")
		optimize  = flag.Bool("optimize", true, "Enable compiler optimizations")
		race      = flag.Bool("race", false, "Enable race detector")
		forceAuth = flag.Bool("force-auth", false, "Force re-authentication")
	)

	// Handle version and help before parsing to avoid issues with commands
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-version" {
			fmt.Printf("%s v%s\n", AppName, AppVersion)
			return
		}
		if os.Args[1] == "--help" || os.Args[1] == "-help" {
			printUsage()
			return
		}
	}

	// Determine command first
	command := "run"
	var flagArgs []string

	if len(os.Args) > 1 {
		// Check if first argument is a known command
		firstArg := os.Args[1]
		if firstArg == "auth" || firstArg == "clean" || firstArg == "run" {
			command = firstArg
			flagArgs = os.Args[2:] // Skip program name and command
		} else {
			// No command specified, treat as run with all args as flags
			flagArgs = os.Args[1:] // Skip program name only
		}
	} else {
		flagArgs = os.Args[1:]
	}

	// Parse flags from the remaining arguments
	flag.CommandLine.Parse(flagArgs)

	if *version {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		return
	}

	if *help {
		printUsage()
		return
	}

	config := &Config{
		FilePath:  *filePath,
		ProblemID: *problemID,
		Timeout:   *timeout,
		Verbose:   *verbose,
		CacheDir:  *cacheDir,
		Parallel:  *parallel,
		ShowDiff:  *showDiff,
		MaxOutput: *maxOutput,
		Optimize:  *optimize,
		Race:      *race,
		ForceAuth: *forceAuth,
	}

	//Ensure cache exists
	enusureCacheDir(config)

	switch command {
	case "auth":
		if err := handleAuth(config); err != nil {
			red.Printf("❌ Authentication failed: %v\n", err)
			os.Exit(1)
		}
		return
	case "clean":
		if err := os.RemoveAll(*cacheDir); err != nil {
			red.Printf("Error cleaning cache: %v\n", err)
			os.Exit(1)
		}
		green.Println("Cache cleaned successfully")
		return
	case "run":
		// Continue with normal execution
	default:
		red.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	// Validate required flags for run command
	if *filePath == "" || *problemID == "" {
		red.Println("Error: Both -file and -problem flags are required for run command")
		printUsage()
		os.Exit(1)
	}

	// Validate file exists and is a Go file
	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		red.Printf("Error: File %s does not exist\n", *filePath)
		os.Exit(1)
	}

	if !strings.HasSuffix(*filePath, ".go") {
		red.Printf("Error: File %s is not a Go file (.go extension required)\n", *filePath)
		os.Exit(1)
	}

	// Validate problem ID
	if _, err := strconv.Atoi(*problemID); err != nil {
		red.Printf("Error: Invalid problem ID %s\n", *problemID)
		os.Exit(1)
	}

	runner := NewTestRunner(config)

	cyan.Printf("🚀 Starting CSES Go Test Runner for problem %s\n", *problemID)
	cyan.Printf("📁 Solution file: %s\n", *filePath)

	if err := runner.Run(); err != nil {
		red.Printf("❌ Runner failed: %v\n", err)
		os.Exit(1)
	}
}

func handleAuth(config *Config) error {
	auth := NewCSESAuth(config)

	if config.ForceAuth {
		yellow.Println("🔐 Forcing re-authentication...")
		if err := auth.ClearSession(); err != nil {
			yellow.Printf("⚠️  Failed to clear session: %v\n", err)
		}
	}

	if err := auth.EnsureAuthenticated(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	green.Println("✅ Authentication successful")
	return nil
}

func enusureCacheDir(config *Config) {
	// Get the current user
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		return
	}

	// Get the home directory
	homeDir := currentUser.HomeDir

	absolutePath := strings.Replace(config.CacheDir, "~", homeDir, 1)
	config.CacheDir = absolutePath

	// Clean and resolve the path
	finalPath := filepath.Clean(absolutePath)
	os.MkdirAll(finalPath, os.ModeDir)
}
