package auth

import (
    "fmt"

    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/models"
)

// AuthService handles authentication operations
type AuthService struct {
    userRepo     *database.UserRepository
    securityRepo *database.SecurityRepository
    minPasswordLength int
    requireSpecial    bool
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo *database.UserRepository, securityRepo *database.SecurityRepository, minPasswordLength int, requireSpecial bool) *AuthService {
    return &AuthService{
        userRepo:          userRepo,
        securityRepo:      securityRepo,
        minPasswordLength: minPasswordLength,
        requireSpecial:    requireSpecial,
    }
}

// Register registers a new user
func (s *AuthService) Register(username, password string) (*models.User, error) {
    // Validate username
    if err := ValidateUsername(username); err != nil {
        return nil, err
    }

    // Check if username already exists
    exists, err := s.userRepo.UsernameExists(username)
    if err != nil {
        return nil, fmt.Errorf("failed to check username: %w", err)
    }
    if exists {
        return nil, fmt.Errorf("username already exists")
    }

    // Validate password strength
    if err := ValidatePasswordStrength(password, s.minPasswordLength, s.requireSpecial); err != nil {
        return nil, err
    }

    // Generate salt
    salt, err := GenerateSalt()
    if err != nil {
        return nil, fmt.Errorf("failed to generate salt: %w", err)
    }

    // Hash password
    passwordHash := HashPassword(password, salt)

    // Create user
    user, err := s.userRepo.Create(username, passwordHash, salt)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    return user, nil
}

// Login authenticates a user and returns the user object
func (s *AuthService) Login(username, password, ipAddress string) (*models.User, error) {
    // Get user by username
    user, err := s.userRepo.GetByUsername(username)
    if err != nil {
        // Record failed login attempt with dummy user ID (0)
        s.securityRepo.RecordLoginAttempt(0, ipAddress, false, nil)
        return nil, fmt.Errorf("invalid username or password")
    }

    // Check if account is active
    if !user.IsActive {
        s.securityRepo.RecordLoginAttempt(user.UserID, ipAddress, false, nil)
        return nil, fmt.Errorf("account is inactive")
    }

    // Verify password
    if !VerifyPassword(password, user.PasswordSalt, user.PasswordHash) {
        s.securityRepo.RecordLoginAttempt(user.UserID, ipAddress, false, nil)
        return nil, fmt.Errorf("invalid username or password")
    }

    // At this point, password is correct - record successful login
    s.securityRepo.RecordLoginAttempt(user.UserID, ipAddress, true, nil)

    // Update last login time
    if err := s.userRepo.UpdateLastLogin(user.UserID); err != nil {
        // Log error but don't fail the login
        fmt.Printf("Warning: failed to update last login time: %v\n", err)
    }

    return user, nil
}

// ValidateUsername validates a username
func ValidateUsername(username string) error {
    if len(username) < 3 {
        return fmt.Errorf("username must be at least 3 characters long")
    }

    if len(username) > 50 {
        return fmt.Errorf("username must be at most 50 characters long")
    }

    // Check for valid characters (alphanumeric, underscore, hyphen)
    for _, char := range username {
        if !isValidUsernameChar(char) {
            return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
        }
    }

    return nil
}

// isValidUsernameChar checks if a character is valid in a username
func isValidUsernameChar(char rune) bool {
    return (char >= 'a' && char <= 'z') ||
        (char >= 'A' && char <= 'Z') ||
        (char >= '0' && char <= '9') ||
        char == '_' ||
        char == '-'
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(userID int64, oldPassword, newPassword string) error {
    // Get user
    user, err := s.userRepo.GetByID(userID)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    // Verify old password
    if !VerifyPassword(oldPassword, user.PasswordSalt, user.PasswordHash) {
        return fmt.Errorf("incorrect old password")
    }

    // Validate new password strength
    if err := ValidatePasswordStrength(newPassword, s.minPasswordLength, s.requireSpecial); err != nil {
        return err
    }

    // Generate new salt
    newSalt, err := GenerateSalt()
    if err != nil {
        return fmt.Errorf("failed to generate salt: %w", err)
    }

    // Hash new password
    newPasswordHash := HashPassword(newPassword, newSalt)

    // Update password in database
    // TODO: Implement UpdatePassword in UserRepository
    _ = newPasswordHash
    _ = newSalt

    return fmt.Errorf("password update not yet implemented")
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(userID int64) (*models.User, error) {
    return s.userRepo.GetByID(userID)
}

// GetUserByUsername retrieves a user by username
func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
    return s.userRepo.GetByUsername(username)
}
