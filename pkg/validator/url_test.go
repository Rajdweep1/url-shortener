package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLValidator_ValidateURL(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		name      string
		url       string
		expectErr bool
		errMsg    string
	}{
		// Valid URLs
		{"Valid HTTP", "http://example.com", false, ""},
		{"Valid HTTPS", "https://example.com", false, ""},
		{"Valid with path", "https://example.com/path/to/page", false, ""},
		{"Valid with query", "https://example.com?param=value", false, ""},
		{"Valid with fragment", "https://example.com#section", false, ""},
		{"Valid with port", "https://example.com:8080", false, ""},
		{"Valid subdomain", "https://sub.example.com", false, ""},
		{"Valid with all components", "https://user:pass@sub.example.com:8080/path?param=value#section", false, ""},

		// Invalid URLs
		{"Empty URL", "", true, "URL cannot be empty"},
		{"Invalid scheme", "ftp://example.com", true, "unsupported URL scheme"},
		{"No scheme", "example.com", true, "unsupported URL scheme"},
		{"Invalid format", "not-a-url", true, "URL too short"},
		{"Only scheme", "https://", true, "URL too short"},
		{"Invalid domain", "https://", true, "URL too short"},
		{"Localhost", "http://localhost", true, "localhost URLs are not allowed"},
		{"Local IP", "http://127.0.0.1", true, "localhost URLs are not allowed"},
		{"Private IP", "http://192.168.1.1", true, "private IP addresses are not allowed"},
		{"Private IP 10.x", "http://10.0.0.1", true, "private IP addresses are not allowed"},
		{"Private IP 172.x", "http://172.16.0.1", true, "private IP addresses are not allowed"},

		// Malicious patterns
		{"JavaScript protocol", "javascript:alert('xss')", true, "URL contains potentially malicious content"},
		{"Data URL", "data:text/html,<script>alert('xss')</script>", true, "URL contains potentially malicious content"},
		{"File protocol", "file:///etc/passwd", true, "URL contains potentially malicious content"},

		// Edge cases
		{"Very long URL", "https://example.com/" + generateLongPath(2100), true, "URL too long"},
		{"Unicode domain", "https://测试.com", false, ""},
		{"Punycode domain", "https://xn--fsq.com", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateURL(tt.url)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCustomAlias(t *testing.T) {
	tests := []struct {
		name      string
		alias     string
		expectErr bool
		errMsg    string
	}{
		// Valid aliases
		{"Simple alias", "my-link", false, ""},
		{"With numbers", "link123", false, ""},
		{"With underscores", "my_link", false, ""},
		{"Mixed case", "MyLink", false, ""},
		{"Long alias", "this-is-a-very-long-but-valid-alias", false, ""},

		// Invalid aliases
		{"Empty alias", "", false, ""}, // Empty is allowed (optional)
		{"Too short", "a", true, "custom alias too short"},
		{"Too long", generateLongString(101), true, "custom alias too long"},
		{"With spaces", "my link", true, "custom alias can only contain"},
		{"With special chars", "my@link", true, "custom alias can only contain"},
		{"With dots", "my.link", true, "custom alias can only contain"},
		{"Reserved word", "admin", true, "custom alias 'admin' is reserved"},
		{"Reserved word", "api", true, "custom alias 'api' is reserved"},
		{"Reserved word", "www", true, "custom alias 'www' is reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomAlias(tt.alias)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Note: Private methods are not tested directly, only through the public ValidateURL method

// Helper functions for tests
func generateLongPath(length int) string {
	result := ""
	segment := "very-long-path-segment/"
	for len(result) < length {
		result += segment
	}
	return result[:length]
}

func generateLongString(length int) string {
	result := ""
	for i := 0; i < length; i++ {
		result += "a"
	}
	return result
}

func BenchmarkValidateURL(b *testing.B) {
	validator := NewURLValidator()
	url := "https://example.com/path/to/page?param=value#section"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateURL(url)
	}
}

func BenchmarkValidateCustomAlias(b *testing.B) {
	alias := "my-custom-alias-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateCustomAlias(alias)
	}
}
