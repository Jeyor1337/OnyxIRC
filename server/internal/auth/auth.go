package auth

import (
    "fmt"

    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/models"
)

type AuthService struct {
    userRepo     *database.UserRepository
    securityRepo *database.SecurityRepository
    minPasswordLength int
    requireSpecial    bool
}

func NewAuthService(userRepo *database.UserRepository, securityRepo *database.SecurityRepository, minPasswordLength int, requireSpecial bool) *AuthService {
    return &AuthService{
        userRepo:          userRepo,
        securityRepo:      securityRepo,
        minPasswordLength: minPasswordLength,
        requireSpecial:    requireSpecial,
    }
}

func (s *AuthService) Register(username, password string) (*models.User, error) {
    
    if err := ValidateUsername(username); err != nil {
        return nil, err
    }

    exists, err := s.userRepo.UsernameExists(username)
    if err != nil {
        return nil, fmt.Errorf("failed to check username: %w", err)
    }
    if exists {
        return nil, fmt.Errorf("username already exists")
    }

    if err := ValidatePasswordStrength(password, s.minPasswordLength, s.requireSpecial); err != nil {
        return nil, err
    }

    salt, err := GenerateSalt()
    if err != nil {
        return nil, fmt.Errorf("failed to generate salt: %w", err)
    }

    passwordHash := HashPassword(password, salt)

    user, err := s.userRepo.Create(username, passwordHash, salt)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    return user, nil
}

func (s *AuthService) Login(username, password, ipAddress string) (*models.User, error) {
    
    user, err := s.userRepo.GetByUsername(username)
    if err != nil {
        
        s.securityRepo.RecordLoginAttempt(0, ipAddress, false, nil)
        return nil, fmt.Errorf("invalid username or password")
    }

    if !user.IsActive {
        s.securityRepo.RecordLoginAttempt(user.UserID, ipAddress, false, nil)
        return nil, fmt.Errorf("account is inactive")
    }

    if !VerifyPassword(password, user.PasswordSalt, user.PasswordHash) {
        s.securityRepo.RecordLoginAttempt(user.UserID, ipAddress, false, nil)
        return nil, fmt.Errorf("invalid username or password")
    }

    s.securityRepo.RecordLoginAttempt(user.UserID, ipAddress, true, nil)

    if err := s.userRepo.UpdateLastLogin(user.UserID); err != nil {
        
        fmt.Printf("Warning: failed to update last login time: %v\n", err)
    }

    return user, nil
}

func ValidateUsername(username string) error {
    if len(username) < 3 {
        return fmt.Errorf("username must be at least 3 characters long")
    }

    if len(username) > 50 {
        return fmt.Errorf("username must be at most 50 characters long")
    }

    for _, char := range username {
        if !isValidUsernameChar(char) {
            return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
        }
    }

    return nil
}

func isValidUsernameChar(char rune) bool {
    return (char >= 'a' && char <= 'z') ||
        (char >= 'A' && char <= 'Z') ||
        (char >= '0' && char <= '9') ||
        char == '_' ||
        char == '-'
}

func (s *AuthService) ChangePassword(userID int64, oldPassword, newPassword string) error {
    
    user, err := s.userRepo.GetByID(userID)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if !VerifyPassword(oldPassword, user.PasswordSalt, user.PasswordHash) {
        return fmt.Errorf("incorrect old password")
    }

    if err := ValidatePasswordStrength(newPassword, s.minPasswordLength, s.requireSpecial); err != nil {
        return err
    }

    newSalt, err := GenerateSalt()
    if err != nil {
        return fmt.Errorf("failed to generate salt: %w", err)
    }

    newPasswordHash := HashPassword(newPassword, newSalt)

    _ = newPasswordHash
    _ = newSalt

    return fmt.Errorf("password update not yet implemented")
}

func (s *AuthService) GetUserByID(userID int64) (*models.User, error) {
    return s.userRepo.GetByID(userID)
}

func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
    return s.userRepo.GetByUsername(username)
}
