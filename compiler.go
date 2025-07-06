package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GoCompiler struct {
	config *Config
}

func NewGoCompiler(config *Config) *GoCompiler {
	return &GoCompiler{config: config}
}

func (c *GoCompiler) ValidateGo() error {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Go is not installed or not in PATH: %w", err)
	}

	if c.config.Verbose {
		cyan.Printf("🔍 %s\n", strings.TrimSpace(string(output)))
	}

	return nil
}

func (c *GoCompiler) ValidateSyntax() error {
	// Check if the file compiles without building
	cmd := exec.Command("go", "run", "-n", c.config.FilePath)

	if c.config.Verbose {
		yellow.Printf("🔍 Validating syntax: %s\n", cmd.String())
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("syntax validation failed: %w", err)
	}

	return nil
}

func (c *GoCompiler) Compile() (string, error) {
	outputPath := c.getOutputPath()

	args := []string{"build", "-o", outputPath}
	args = append(args, c.config.GetBuildFlags()...)
	args = append(args, c.config.FilePath)

	cmd := exec.Command("go", args...)

	if c.config.Verbose {
		yellow.Printf("🔨 Compiling: %s\n", cmd.String())
	}

	// Capture compilation output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	// Verify executable was created
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("executable not created: %w", err)
	}

	return outputPath, nil
}

func (c *GoCompiler) getOutputPath() string {
	dir, _ := filepath.Abs(filepath.Dir(c.config.FilePath))
	base := strings.TrimSuffix(filepath.Base(c.config.FilePath), ".go")
	return filepath.Join(dir, base+"_cses_executable")
}

func (c *GoCompiler) GetModuleInfo() (string, error) {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = filepath.Dir(c.config.FilePath)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
