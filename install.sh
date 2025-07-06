#!/bin/bash

set -e

echo "🚀 Installing CSES Go Runner v2.0..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    echo "Visit: https://golang.org/doc/install"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.19"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or later."
    exit 1
fi

echo "✅ Go version $GO_VERSION detected"

# Create directory
mkdir -p cses-go-runner
cd cses-go-runner

# Initialize module if not exists
if [ ! -f go.mod ]; then
    echo "📦 Initializing Go module..."
    go mod init cses-go-runner
fi

# Download dependencies
echo "📥 Downloading dependencies..."
go mod tidy

# Build the application
echo "🔨 Building application..."
go build -ldflags="-s -w" -o cses-go-runner .

# Make it executable
chmod +x cses-go-runner

echo "✅ Build complete!"

# Test the installation
echo "🧪 Testing installation..."
./cses-go-runner -version

echo ""
echo "🔐 Authentication Setup:"
echo "Set your CSES credentials as environment variables:"
echo "  export CSES_USERNAME='your_username'"
echo "  export CSES_PASSWORD='your_password'"
echo ""
echo "Then authenticate with:"
echo "  ./cses-go-runner auth"

# Optionally install to system
echo ""
read -p "Install to /usr/local/bin? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    sudo mv cses-go-runner /usr/local/bin/
    echo "✅ Installed to /usr/local/bin/cses-go-runner"
    echo "You can now run 'cses-go-runner' from anywhere!"
else
    echo "✅ Build complete. Run with ./cses-go-runner"
fi

echo ""
echo "🎉 Installation complete!"
echo ""
echo "Usage examples:"
echo "  cses-go-runner auth"
echo "  cses-go-runner -file=solution.go -problem=1068"
echo "  cses-go-runner -file=solution.go -problem=1068 -verbose -diff"
echo "  cses-go-runner -help"
