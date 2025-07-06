# CSES Go Runner v1.0

A specialized CLI tool for running and testing CSES (Code Submission Evaluation System) problems with Go solutions. Now with direct CSES authentication and test case downloading!

## New Features in v1.0

- 🔐 **Direct CSES Authentication** - Login with your CSES credentials
- 📥 **Official Test Case Download** - Download test cases directly from CSES
- 🔄 **Auto Re-authentication** - Automatically handles session expiration
- 📦 **Zip File Processing** - Extract and process test cases from CSES zip files

## Features

- 🚀 **Go-optimized**: Built specifically for Go solutions
- 🧪 **Parallel test execution** with configurable concurrency
- 📦 **Direct CSES integration** with official test cases
- 💾 **Smart caching system** for faster subsequent runs
- 🎨 **Beautiful colorized output** with detailed progress reporting
- ⚡ **Timeout handling** with context-based cancellation
- 📊 **Comprehensive result reporting** with performance metrics
- 🔍 **Verbose mode** for debugging and optimization
- 🧹 **Cache management** utilities
- 🏁 **Race detection** support for concurrent programs

## Installation

### Quick Install
```bash
curl -sSL https://raw.githubusercontent.com/anurag5sh/cses-go-runner/main/install.sh | bash
```

### Manual Install
```bash
git clone https://github.com/anurag5sh/cses-go-runner.git
cd cses-go-runner
go mod tidy
go build -ldflags="-s -w" -o cses-go-runner .
```

### Using Make
```bash
make build-optimized
make install  # Optional: install to /usr/local/bin
```

## Setup

### Environment Variables
Set your CSES credentials:
```bash
export CSES_USERNAME='your_username'
export CSES_PASSWORD='your_password'
```

### Authentication
Authenticate with CSES:
```bash
cses-go-runner auth
```

## Usage

### Basic Usage
```bash
# Authenticate first
cses-go-runner auth

# Run tests
cses-go-runner -file=solution.go -problem=1068
```

### Commands
```bash
# Authenticate with CSES
cses-go-runner auth

# Run tests (default command)
cses-go-runner -file=solution.go -problem=1068

# Run tests with explicit command
cses-go-runner run -file=solution.go -problem=1068

# Clean cache
cses-go-runner clean
```

### Advanced Usage
```bash
# With verbose output and diff display
cses-go-runner -file=solution.go -problem=1068 -verbose -diff

# With custom timeout and parallel execution
cses-go-runner -file=solution.go -problem=1068 -timeout=5s -parallel=8

# With race detection (for concurrent programs)
cses-go-runner -file=solution.go -problem=1068 -race

# Force re-authentication
cses-go-runner -file=solution.go -problem=1068 -force-auth
```

### Available Options

| Flag | Description | Default |
|------|-------------|---------|
| `-file` | Path to Go solution file | - |
| `-problem` | CSES problem ID | - |
| `-timeout` | Timeout per test case | `2s` |
| `-verbose` | Enable verbose output | `false` |
| `-cache-dir` | Cache directory | `./cses-cache` |
| `-parallel` | Number of parallel executions | `4` |
| `-diff` | Show diff for failed tests | `false` |
| `-max-output` | Maximum output length to display | `1000` |
| `-optimize` | Enable compiler optimizations | `true` |
| `-race` | Enable race detector | `false` |
| `-force-auth` | Force re-authentication | `false` |
| `-help` | Show help message | `false` |
| `-version` | Show version | `false` |

## Authentication Flow

1. **Set Environment Variables**:
   ```bash
   export CSES_USERNAME='your_username'
   export CSES_PASSWORD='your_password'
   ```

2. **Authenticate**:
   ```bash
   cses-go-runner auth
   ```

3. **Use Normally**:
   ```bash
   cses-go-runner -file=solution.go -problem=1068
   ```

The tool will automatically:
- Check for valid authentication before running tests
- Re-authenticate if the session expires
- Download test cases directly from CSES
- Cache everything for offline development

## Test Case Download

The tool downloads test cases directly from CSES using the official API:
- **URL**: `https://cses.fi/problemset/tests/{problem_id}/`
- **Method**: POST request with CSRF token and session ID
- **Format**: ZIP file containing input/output pairs
- **Caching**: Automatically cached for subsequent runs

## Output Format

```
🚀 Starting CSES Go Test Runner for problem 1068
📁 Solution file: solution.go
🔍 go version go1.21.0 linux/amd64
📥 Fetching test cases from CSES...
📦 Extracted 15 test cases from zip file
💾 Cached 15 test cases to ./cses-cache/1068
✅ Found 15 test cases
🔨 Compiling Go solution...
✅ Compilation successful
🧪 Running 15 test cases (parallel: 4)...
⏱️  Total execution time: 0.25s

============================================================
📊 TEST RESULTS SUMMARY
============================================================
✅ PASSED: 15/15 tests
⏱️  Average execution time: 16.67ms

============================================================
🎉 ALL TESTS PASSED! 🎉
============================================================
```

## Cache Structure

```
cses-cache/
├── .auth/
│   └── session.json          # Authentication session
├── 1068/
│   ├── 1.in
│   ├── 1.out
│   ├── 2.in
│   ├── 2.out
│   └── ...
└── 1083/
    ├── 1.in
    ├── 1.out
    └── ...
```

## Example Go Solutions

### Weird Algorithm (Problem 1068)
```go
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
```

## Troubleshooting

### Authentication Issues
```bash
# Check environment variables
echo $CSES_USERNAME
echo $CSES_PASSWORD

# Force re-authentication
cses-go-runner auth -force-auth

# Check verbose output
cses-go-runner -file=solution.go -problem=1068 -verbose
```

### Test Case Issues
```bash
# Clean cache and retry
cses-go-runner clean
cses-go-runner -file=solution.go -problem=1068
```

### Common Error Messages
- `authentication required` - Run `cses-go-runner auth` first
- `session expired` - Tool will automatically re-authenticate
- `failed to download test cases` - Check your internet connection and credentials

## Security Notes

- Credentials are only stored in environment variables
- Session tokens are stored locally in `cses-cache/.auth/session.json`
- Use `cses-go-runner clean` to remove all cached data including sessions
- Never commit your credentials to version control

## Contributing

1. Fork the repository
2. Create a feature branch
3. Test with various Go solutions
4. Submit a pull request

## License

MIT License - see LICENSE file for details
