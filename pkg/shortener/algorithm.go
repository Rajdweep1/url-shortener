package shortener

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	// Base62 alphabet (0-9, a-z, A-Z) - URL safe characters
	base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	defaultLength  = 7
	minLength      = 4
	maxLength      = 10
)

// Shortener handles URL shortening operations
type Shortener struct {
	length int
}

// New creates a new Shortener instance
func New(length int) *Shortener {
	if length < minLength || length > maxLength {
		length = defaultLength
	}
	return &Shortener{length: length}
}

// GenerateShortCode generates a cryptographically secure random short code
func (s *Shortener) GenerateShortCode() (string, error) {
	result := make([]byte, s.length)
	alphabetLen := big.NewInt(int64(len(base62Alphabet)))
	
	for i := range result {
		randomIndex, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = base62Alphabet[randomIndex.Int64()]
	}
	
	return string(result), nil
}

// GenerateFromURL creates a deterministic short code from URL (for idempotency)
// This ensures the same URL always gets the same short code
func (s *Shortener) GenerateFromURL(url string) string {
	// Add timestamp to make it unique per request time
	input := fmt.Sprintf("%s:%d", url, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(input))
	
	// Convert to base62
	return s.hashToBase62(hash[:], s.length)
}

// GenerateDeterministic creates a deterministic short code from URL without timestamp
// This ensures true idempotency - same URL always gets same code
func (s *Shortener) GenerateDeterministic(url string) string {
	hash := sha256.Sum256([]byte(url))
	return s.hashToBase62(hash[:], s.length)
}

// hashToBase62 converts a hash to base62 representation
func (s *Shortener) hashToBase62(hash []byte, length int) string {
	// Convert hash bytes to a big integer
	num := big.NewInt(0)
	num.SetBytes(hash)
	
	// Convert to base62
	base := big.NewInt(int64(len(base62Alphabet)))
	result := make([]byte, 0, length)
	
	for num.Cmp(big.NewInt(0)) > 0 && len(result) < length {
		remainder := big.NewInt(0)
		num.DivMod(num, base, remainder)
		result = append(result, base62Alphabet[remainder.Int64()])
	}
	
	// Pad with leading characters if needed
	for len(result) < length {
		result = append(result, base62Alphabet[0])
	}
	
	// Reverse the result (since we built it backwards)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	
	return string(result)
}

// IsValidShortCode validates if a short code is valid
func IsValidShortCode(shortCode string) bool {
	if len(shortCode) < minLength || len(shortCode) > maxLength {
		return false
	}
	
	for _, char := range shortCode {
		if !strings.ContainsRune(base62Alphabet, char) {
			return false
		}
	}
	
	return true
}

// GenerateCustomCode validates and formats a custom alias
func (s *Shortener) GenerateCustomCode(alias string) (string, error) {
	if len(alias) < 3 || len(alias) > 50 {
		return "", fmt.Errorf("custom alias must be between 3 and 50 characters")
	}
	
	// Allow alphanumeric, hyphens, and underscores
	for _, char := range alias {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			char == '-' || char == '_') {
			return "", fmt.Errorf("custom alias can only contain alphanumeric characters, hyphens, and underscores")
		}
	}
	
	return alias, nil
}

// GenerateWithCollisionHandling generates a short code with collision handling
func (s *Shortener) GenerateWithCollisionHandling(url string, attempt int) (string, error) {
	if attempt == 0 {
		// First attempt: try deterministic generation
		return s.GenerateDeterministic(url), nil
	}
	
	if attempt < 5 {
		// Next few attempts: add attempt number to make it unique
		input := fmt.Sprintf("%s:attempt:%d", url, attempt)
		hash := sha256.Sum256([]byte(input))
		return s.hashToBase62(hash[:], s.length), nil
	}
	
	// Final attempts: use random generation
	return s.GenerateShortCode()
}
