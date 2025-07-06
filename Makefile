.PHONY: build install clean test help run-sample auth

# Build configuration
BINARY_NAME=cses-go-runner
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin

# Default target
all: build

# Build the application
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build with optimizations
build-optimized:
	@echo "🔨 Building optimized $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Optimized build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install the application
install: build-optimized
	@echo "📦 Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@echo "✅ Installation complete"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f *_cses_executable
	@rm -rf cses-cache
	@echo "✅ Clean complete"

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Download dependencies
deps:
	@echo "📥 Downloading dependencies..."
	@go mod tidy
	@go mod download

# Authenticate with CSES
auth: build
	@echo "🔐 Authenticating with CSES..."
	@$(BUILD_DIR)/$(BINARY_NAME) auth

# Create a sample Go solution for testing
sample-solution:
	@echo "📝 Creating sample solution..."
	@cat > weird_algorithm.go << 'EOF'
package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	var n int
	fmt.Scanf("%d", &n)
	
	var result []string
	for n != 1 {
		result = append(result, strconv.Itoa(n))
		if n%2 == 0 {
			n = n / 2
		} else {
			n = n*3 + 1
		}
	}
	result = append(result, "1")
	
	fmt.Println(strings.Join(result, " "))
}
EOF
	@echo "✅ Sample solution created: weird_algorithm.go"

# Run sample test
run-sample: build sample-solution
	@echo "🚀 Running sample test..."
	@echo "Note: You need to authenticate first with 'make auth'"
	@$(BUILD_DIR)/$(BINARY_NAME) -file=weird_algorithm.go -problem=1068 -verbose

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  build-optimized - Build with optimizations"
	@echo "  install         - Install to $(INSTALL_DIR)"
	@echo "  clean           - Clean build artifacts"
	@echo "  test            - Run tests"
	@echo "  deps            - Download dependencies"
	@echo "  auth            - Authenticate with CSES"
	@echo "  sample-solution - Create a sample Go solution"
	@echo "  run-sample      - Build and run sample test"
	@echo "  help            - Show this help"
