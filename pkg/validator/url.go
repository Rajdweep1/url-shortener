package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	maxURLLength     = 2048
	minURLLength     = 10
	maxCustomAlias   = 50
	minCustomAlias   = 3
)

var (
	// Common URL schemes that we support
	supportedSchemes = map[string]bool{
		"http":  true,
		"https": true,
		"ftp":   true,
		"ftps":  true,
	}
	
	// Regex for validating custom aliases (alphanumeric, hyphens, underscores)
	customAliasRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	// Common malicious patterns to block
	maliciousPatterns = []string{
		"javascript:",
		"data:",
		"vbscript:",
		"file:",
		"about:",
	}
)

// URLValidator handles URL validation
type URLValidator struct {
	maxLength int
	minLength int
}

// NewURLValidator creates a new URL validator
func NewURLValidator() *URLValidator {
	return &URLValidator{
		maxLength: maxURLLength,
		minLength: minURLLength,
	}
}

// ValidateURL validates if a URL is valid and safe
func (v *URLValidator) ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	
	// Check length constraints
	if len(rawURL) < v.minLength {
		return fmt.Errorf("URL too short (minimum %d characters)", v.minLength)
	}
	
	if len(rawURL) > v.maxLength {
		return fmt.Errorf("URL too long (maximum %d characters)", v.maxLength)
	}
	
	// Check for malicious patterns
	lowerURL := strings.ToLower(rawURL)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(lowerURL, pattern) {
			return fmt.Errorf("URL contains potentially malicious content")
		}
	}
	
	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	// Check if scheme is supported
	if !supportedSchemes[strings.ToLower(parsedURL.Scheme)] {
		return fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}
	
	// Check if host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}
	
	// Additional security checks
	if err := v.validateHost(parsedURL.Host); err != nil {
		return err
	}
	
	return nil
}

// validateHost performs additional host validation
func (v *URLValidator) validateHost(host string) error {
	// Check for localhost and private IP ranges (optional security measure)
	lowerHost := strings.ToLower(host)
	
	// Block localhost variations
	localhostPatterns := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
	}
	
	for _, pattern := range localhostPatterns {
		if strings.Contains(lowerHost, pattern) {
			return fmt.Errorf("localhost URLs are not allowed")
		}
	}
	
	// Block private IP ranges (10.x.x.x, 192.168.x.x, 172.16-31.x.x)
	privateIPPatterns := []string{
		"10.",
		"192.168.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.",
		"172.24.", "172.25.", "172.26.", "172.27.",
		"172.28.", "172.29.", "172.30.", "172.31.",
	}
	
	for _, pattern := range privateIPPatterns {
		if strings.HasPrefix(lowerHost, pattern) {
			return fmt.Errorf("private IP addresses are not allowed")
		}
	}
	
	return nil
}

// ValidateCustomAlias validates a custom alias
func ValidateCustomAlias(alias string) error {
	if alias == "" {
		return nil // Custom alias is optional
	}
	
	if len(alias) < minCustomAlias {
		return fmt.Errorf("custom alias too short (minimum %d characters)", minCustomAlias)
	}
	
	if len(alias) > maxCustomAlias {
		return fmt.Errorf("custom alias too long (maximum %d characters)", maxCustomAlias)
	}
	
	if !customAliasRegex.MatchString(alias) {
		return fmt.Errorf("custom alias can only contain alphanumeric characters, hyphens, and underscores")
	}
	
	// Check for reserved words
	reservedWords := []string{
		"api", "admin", "www", "mail", "ftp", "localhost",
		"stats", "analytics", "dashboard", "health", "metrics",
		"docs", "swagger", "graphql", "webhook", "callback",
	}
	
	lowerAlias := strings.ToLower(alias)
	for _, reserved := range reservedWords {
		if lowerAlias == reserved {
			return fmt.Errorf("custom alias '%s' is reserved", alias)
		}
	}
	
	return nil
}

// SanitizeURL cleans and normalizes a URL
func SanitizeURL(rawURL string) string {
	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)
	
	// Add https:// if no scheme is provided
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	
	return rawURL
}
