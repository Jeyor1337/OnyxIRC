package auth

import (
    "crypto/rand"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/hex"
    "fmt"
)

// HashSHA256 computes the SHA-256 hash of data and returns it as a hex string
func HashSHA256(data string) string {
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

// HashSHA256Bytes computes the SHA-256 hash of byte data
func HashSHA256Bytes(data []byte) []byte {
    hash := sha256.Sum256(data)
    return hash[:]
}

// GenerateSalt generates a random salt for password hashing
func GenerateSalt() (string, error) {
    salt := make([]byte, 16) // 16 bytes = 128 bits
    if _, err := rand.Read(salt); err != nil {
        return "", fmt.Errorf("failed to generate salt: %w", err)
    }
    return hex.EncodeToString(salt), nil
}

// HashPassword hashes a password with a salt using SHA-256
// Returns the hash as a hex string
func HashPassword(password, salt string) string {
    // Combine password and salt
    combined := password + salt
    return HashSHA256(combined)
}

// VerifyPassword verifies a password against a stored hash
// Uses constant-time comparison to prevent timing attacks
func VerifyPassword(password, salt, storedHash string) bool {
    // Hash the provided password with the salt
    computedHash := HashPassword(password, salt)

    // Use constant-time comparison to prevent timing attacks
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

// HashMessage computes a SHA-256 hash of a message for integrity verification
func HashMessage(message string) string {
    return HashSHA256(message)
}

// VerifyMessageHash verifies a message against its hash
func VerifyMessageHash(message, expectedHash string) bool {
    computedHash := HashMessage(message)
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(expectedHash)) == 1
}

// ValidatePasswordStrength validates password strength requirements
func ValidatePasswordStrength(password string, minLength int, requireSpecial bool) error {
    if len(password) < minLength {
        return fmt.Errorf("password must be at least %d characters long", minLength)
    }

    if requireSpecial {
        hasSpecial := false
        hasDigit := false
        hasUpper := false
        hasLower := false

        for _, char := range password {
            switch {
            case char >= 'A' && char <= 'Z':
                hasUpper = true
            case char >= 'a' && char <= 'z':
                hasLower = true
            case char >= '0' && char <= '9':
                hasDigit = true
            case isSpecialChar(char):
                hasSpecial = true
            }
        }

        if !hasSpecial {
            return fmt.Errorf("password must contain at least one special character")
        }
        if !hasDigit {
            return fmt.Errorf("password must contain at least one digit")
        }
        if !hasUpper {
            return fmt.Errorf("password must contain at least one uppercase letter")
        }
        if !hasLower {
            return fmt.Errorf("password must contain at least one lowercase letter")
        }
    }

    return nil
}

// isSpecialChar checks if a character is a special character
func isSpecialChar(char rune) bool {
    specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?/"
    for _, special := range specialChars {
        if char == special {
            return true
        }
    }
    return false
}
