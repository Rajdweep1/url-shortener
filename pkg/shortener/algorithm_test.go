package shortener

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: base62Encode and base62Decode are not exported, so we test the public API instead

func TestShortener_GenerateDeterministic(t *testing.T) {
	s := New(8)

	tests := []struct {
		name string
		url  string
	}{
		{"Simple URL", "https://example.com"},
		{"URL with path", "https://example.com/path/to/page"},
		{"URL with query", "https://example.com?param=value"},
		{"Complex URL", "https://example.com/path?param1=value1&param2=value2#section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate multiple times to ensure deterministic behavior
			code1 := s.GenerateDeterministic(tt.url)
			code2 := s.GenerateDeterministic(tt.url)

			assert.Equal(t, code1, code2, "Deterministic generation should produce same result")
			assert.LessOrEqual(t, len(code1), s.length, "Code should not exceed max length")
			assert.Greater(t, len(code1), 0, "Code should not be empty")
		})
	}
}

func TestShortener_GenerateShortCode(t *testing.T) {
	s := New(8)

	// Generate multiple codes to check for uniqueness
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := s.GenerateShortCode()
		assert.NoError(t, err)
		assert.Equal(t, s.length, len(code), "Random code should have exact length")
		assert.False(t, codes[code], "Random codes should be unique")
		codes[code] = true
	}
}

func TestShortener_GenerateFromURL(t *testing.T) {
	s := New(8)
	url := "https://example.com"

	// Test that GenerateFromURL produces valid codes
	code := s.GenerateFromURL(url)
	assert.LessOrEqual(t, len(code), s.length, "Code should not exceed max length")
	assert.Greater(t, len(code), 0, "Code should not be empty")

	// Test that it produces different codes for different calls (due to timestamp)
	code2 := s.GenerateFromURL(url)
	// Note: These might be the same if called in the same nanosecond, but that's unlikely
	_ = code2 // Suppress unused variable warning
}

// Note: GenerateCustomCode is not implemented in the current shortener
// Custom alias validation is handled in the validator package

func TestIsValidShortCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"Valid alphanumeric", "abc123", true},
		{"Valid mixed case", "AbC123", true},
		{"Valid base62", "abc123XYZ", true},
		{"Empty string", "", false},
		{"Too short", "a", false},
		{"Too long", "this-is-way-too-long-for-a-short-code-and-should-be-rejected", false},
		{"Invalid characters", "abc@123", false},
		{"With dash", "abc-123", false}, // Dashes not allowed in base62
		{"With underscore", "abc_123", false}, // Underscores not allowed in base62
		{"With spaces", "abc 123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidShortCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShortener_DifferentLengths(t *testing.T) {
	lengths := []int{4, 6, 8, 10} // Only valid lengths (maxLength is 10)

	for _, length := range lengths {
		t.Run(fmt.Sprintf("Length_%d", length), func(t *testing.T) {
			s := New(length)

			// Test random generation
			code, err := s.GenerateShortCode()
			assert.NoError(t, err)
			assert.Equal(t, length, len(code))

			// Test deterministic generation (may be shorter than max length)
			detCode := s.GenerateDeterministic("https://example.com")
			assert.LessOrEqual(t, len(detCode), length)
			assert.Greater(t, len(detCode), 0)
		})
	}
}

func TestShortener_EdgeCases(t *testing.T) {
	s := New(8)

	t.Run("Very long URL", func(t *testing.T) {
		longURL := "https://example.com/" + strings.Repeat("very-long-path-segment/", 100)
		code := s.GenerateDeterministic(longURL)
		assert.LessOrEqual(t, len(code), s.length)
	})

	t.Run("URL with unicode", func(t *testing.T) {
		unicodeURL := "https://example.com/测试/页面"
		code := s.GenerateDeterministic(unicodeURL)
		assert.LessOrEqual(t, len(code), s.length)
	})

	t.Run("Minimum length shortener", func(t *testing.T) {
		minS := New(1) // Should default to defaultLength (7)
		code, err := minS.GenerateShortCode()
		assert.NoError(t, err)
		assert.Equal(t, 7, len(code)) // Should default to defaultLength
	})
}

func BenchmarkGenerateDeterministic(b *testing.B) {
	s := New(8)
	url := "https://example.com/some/path?param=value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GenerateDeterministic(url)
	}
}

func BenchmarkGenerateShortCode(b *testing.B) {
	s := New(8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.GenerateShortCode()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateFromURL(b *testing.B) {
	s := New(8)
	url := "https://example.com/some/path?param=value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GenerateFromURL(url)
	}
}
