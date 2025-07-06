package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SessionData represents the stored authentication session
type SessionData struct {
	PHPSessionID string    `json:"php_session_id"`
	CSRFToken    string    `json:"csrf_token"`
	Username     string    `json:"username"`
	CreatedAt    time.Time `json:"created_at"`
	LastUsed     time.Time `json:"last_used"`
}

// CSESAuth handles authentication with CSES
type CSESAuth struct {
	client      *http.Client
	sessionData *SessionData
	sessionFile string
}

// NewCSESAuth creates a new CSES authentication handler
func NewCSESAuth(config *Config) *CSESAuth {
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &CSESAuth{
		client:      client,
		sessionFile: config.GetSessionFile(),
	}
}

// LoadSession loads session data from file
func (a *CSESAuth) LoadSession() error {
	if _, err := os.Stat(a.sessionFile); os.IsNotExist(err) {
		return fmt.Errorf("session file does not exist")
	}

	data, err := os.ReadFile(a.sessionFile)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return fmt.Errorf("failed to parse session data: %w", err)
	}

	a.sessionData = &sessionData
	return nil
}

// SaveSession saves session data to file
func (a *CSESAuth) SaveSession() error {
	if a.sessionData == nil {
		return fmt.Errorf("no session data to save")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(a.sessionFile), 0700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Update last used time
	a.sessionData.LastUsed = time.Now()

	data, err := json.MarshalIndent(a.sessionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := os.WriteFile(a.sessionFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// ClearSession removes the session file
func (a *CSESAuth) ClearSession() error {
	a.sessionData = nil
	if err := os.Remove(a.sessionFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session file: %w", err)
	}
	return nil
}

// HasValidSession checks if we have a valid session
func (a *CSESAuth) HasValidSession() bool {
	if a.sessionData == nil {
		return false
	}

	// Check if session is too old (24 hours)
	if time.Since(a.sessionData.CreatedAt) > 24*time.Hour {
		return false
	}

	// Check if we have required data
	return a.sessionData.PHPSessionID != "" && a.sessionData.CSRFToken != ""
}

// GetCredentials retrieves CSES credentials from environment variables
func (a *CSESAuth) GetCredentials() (string, string, error) {
	username := os.Getenv("CSES_USERNAME")
	password := os.Getenv("CSES_PASSWORD")

	if username == "" {
		return "", "", fmt.Errorf("CSES_USERNAME environment variable is not set")
	}

	if password == "" {
		return "", "", fmt.Errorf("CSES_PASSWORD environment variable is not set")
	}

	return username, password, nil
}

// FetchLoginPage fetches the login page and extracts CSRF token and session ID
func (a *CSESAuth) FetchLoginPage() (string, string, error) {
	yellow.Println("� Fetching login page...")

	resp, err := a.client.Get("https://cses.fi/login")
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch login page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("login page returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read login page: %w", err)
	}

	// Extract CSRF token from HTML
	csrfToken, err := a.extractCSRFToken(string(body))
	if err != nil {
		return "", "", fmt.Errorf("failed to extract CSRF token: %w", err)
	}

	// Extract PHP session ID from cookies
	phpSessionID := a.extractPHPSessionID(resp.Cookies())
	if phpSessionID == "" {
		return "", "", fmt.Errorf("failed to extract PHP session ID from cookies")
	}

	if len(csrfToken) == 0 || len(phpSessionID) == 0 {
		return "", "", fmt.Errorf("failed to extract required authentication data")
	}

	if len(csrfToken) >= 8 && len(phpSessionID) >= 8 {
		cyan.Printf("★ Extracted CSRF token: %s...\n", csrfToken[:8])
		cyan.Printf("!★ Extracted PHP session ID: %s...\n", phpSessionID[:8])
	}

	return csrfToken, phpSessionID, nil
}

// extractCSRFToken extracts CSRF token from HTML content
func (a *CSESAuth) extractCSRFToken(html string) (string, error) {
	// Look for the CSRF token in the hidden input field
	// <input type="hidden" name="csrf_token" value="83835ef6d6c7fbdb0eb3036c56df4d6f">

	patterns := []string{
		`<input[^>]*name="csrf_token"[^>]*value="([^"]+)"`,
		`<input[^>]*value="([^"]+)"[^>]*name="csrf_token"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			token := matches[1]
			if len(token) > 0 {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("CSRF token not found in login page")
}

// extractPHPSessionID extracts PHP session ID from cookies
func (a *CSESAuth) extractPHPSessionID(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie.Name == "PHPSESSID" {
			return cookie.Value
		}
	}
	return ""
}

// Login performs the actual login process including all validation
func (a *CSESAuth) Login() error {
	username, password, err := a.GetCredentials()
	if err != nil {
		return fmt.Errorf("credential error: %w", err)
	}

	// Fetch login page to get CSRF token and session ID
	csrfToken, phpSessionID, err := a.FetchLoginPage()
	if err != nil {
		return fmt.Errorf("failed to fetch login page: %w", err)
	}

	yellow.Println("💐 Logging in to CSES...")

	// Prepare login data
	loginData := url.Values{
		"csrf_token": {csrfToken},
		"nick":       {username},
		"pass":       {password},
	}

	// Create login request
	req, err := http.NewRequest("POST", "https://cses.fi/login", strings.NewReader(loginData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	// Set headers to match browser request
	a.setLoginHeaders(req, phpSessionID)

	// Perform login request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	defer resp.Body.Close()

	// Check if login was successful
	if err := a.validateLoginResponse(resp); err != nil {
		return fmt.Errorf("login validation failed: %w", err)
	}

	// Save session data
	a.sessionData = &SessionData{
		PHPSessionID: phpSessionID,
		CSRFToken:    csrfToken,
		Username:     username,
		CreatedAt:    time.Now(),
		LastUsed:     time.Now(),
	}

	// Save session to file
	if err := a.SaveSession(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	green.Println("✅ Login successful")
	return nil
}

// setLoginHeaders sets all required headers for login request
func (a *CSESAuth) setLoginHeaders(req *http.Request, phpSessionID string) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", phpSessionID))
	req.Header.Set("DNT", "1")
	req.Header.Set("Origin", "https://cses.fi")
	req.Header.Set("Referer", "https://cses.fi/login")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)
}

// validateLoginResponse validates the login response
func (a *CSESAuth) validateLoginResponse(resp *http.Response) error {
	// Successful login should redirect (status 302) or return 200
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 302 or 200, got %d", resp.StatusCode)
	}

	// Check if there's a redirect location
	location := resp.Header.Get("Location")
	if location != "" {
		// If redirecting to login page, it means login failed
		if strings.Contains(location, "/login") {
			return fmt.Errorf("failed to login: invalid credentials")
		}
	}

	// Read response body to check for error messages
	body, err := io.ReadAll(resp.Body)
	if err == nil {
		html := string(body)
		// Check for error messages in HTML
		if strings.Contains(html, "Invalid username or password") ||
			strings.Contains(html, "Login failed") ||
			strings.Contains(html, "error") {
			return fmt.Errorf("failed to login: invalid credentials")
		}
	}

	return nil
}

// EnsureAuthenticated ensures we have a valid authentication session
func (a *CSESAuth) EnsureAuthenticated() error {
	// Try to load existing session
	if err := a.LoadSession(); err == nil && a.HasValidSession() {
		if a.TestSession() == nil {
			return nil
		}
	}

	// Session invalid or expired, login again
	return a.Login()
}

// TestSession tests if the session is still valid by attempting a request
func (a *CSESAuth) TestSession() error {
	if a.sessionData == nil {
		return fmt.Errorf("no session data")
	}

	// Test session by trying to access a protected page
	req, err := http.NewRequest("GET", "https://cses.fi/problemset/stats", nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	// Set cookie with session ID
	req.Header.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", a.sessionData.PHPSessionID))

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error in making test session request")
	}

	body, err := io.ReadAll(resp.Body)

	if strings.Contains(string(body), "Please login") {
		return fmt.Errorf("login expired, need to relogin")
	}

	return nil
}

// DownloadTestCases downloads test cases for a given problem ID
func (a *CSESAuth) DownloadTestCases(problemID string) ([]byte, error) {
	if a.sessionData == nil {
		return nil, fmt.Errorf("no session data")
	}

	// Prepare POST data for test case download
	formData := url.Values{
		"csrf_token": {a.sessionData.CSRFToken},
		"download":   {"true"},
	}

	// Create POST request to download test cases
	url := fmt.Sprintf("https://cses.fi/problemset/tests/%s/", problemID)
	req, err := http.NewRequest("POST", url, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create test case download request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", a.sessionData.PHPSessionID))
	req.Header.Set("Referer", fmt.Sprintf("https://cses.fi/problemset/task/%s", problemID))
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

	// Execute the request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute test case download request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if strings.Contains(location, "/login") {
			return nil, fmt.Errorf("session expired, requires re-authentication")
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download test cases: HTTP %d", resp.StatusCode)
	}

	// Check if the response is a ZIP file
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/zip") && !strings.Contains(contentType, "application/octet-stream") {
		return nil, fmt.Errorf("expected ZIP file, got content type: %s", contentType)
	}

	// Read the ZIP file data
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ZIP file: %w", err)
	}

	if len(zipData) == 0 {
		return nil, fmt.Errorf("received empty ZIP file")
	}

	return zipData, nil
}
