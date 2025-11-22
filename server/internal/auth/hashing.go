package auth

import (
    "crypto/rand"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/hex"
    "fmt"
)

func HashSHA256(data string) string {
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

func HashSHA256Bytes(data []byte) []byte {
    hash := sha256.Sum256(data)
    return hash[:]
}

func GenerateSalt() (string, error) {
    salt := make([]byte, 16) 
    if _, err := rand.Read(salt); err != nil {
        return "", fmt.Errorf("failed to generate salt: %w", err)
    }
    return hex.EncodeToString(salt), nil
}

func HashPassword(password, salt string) string {
    
    combined := password + salt
    return HashSHA256(combined)
}

func VerifyPassword(password, salt, storedHash string) bool {
    
    computedHash := HashPassword(password, salt)

    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

func HashMessage(message string) string {
    return HashSHA256(message)
}

func VerifyMessageHash(message, expectedHash string) bool {
    computedHash := HashMessage(message)
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(expectedHash)) == 1
}

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

func isSpecialChar(char rune) bool {
    specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?/"
    for _, special := range specialChars {
        if char == special {
            return true
        }
    }
    return false
}
