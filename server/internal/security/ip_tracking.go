package security

import (
    "fmt"
    "log"

    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/models"
)

type IPTrackingService struct {
    securityRepo   *database.SecurityRepository
    maxSuspicion   int
    enableTracking bool
}

func NewIPTrackingService(securityRepo *database.SecurityRepository, maxSuspicion int, enableTracking bool) *IPTrackingService {
    return &IPTrackingService{
        securityRepo:   securityRepo,
        maxSuspicion:   maxSuspicion,
        enableTracking: enableTracking,
    }
}

func (s *IPTrackingService) CheckIPAndTrack(userID int64, currentIP string) error {
    if !s.enableTracking {
        return nil
    }

    status, err := s.securityRepo.GetSecurityStatus(userID)
    if err != nil {
        return fmt.Errorf("failed to get security status: %w", err)
    }

    if status.AccountLocked {
        reason := "Account locked"
        if status.LockReason != nil {
            reason = *status.LockReason
        }
        return fmt.Errorf("account is locked: %s", reason)
    }

    if status.LastKnownIP == nil {
        if err := s.securityRepo.UpdateLastKnownIP(userID, currentIP); err != nil {
            log.Printf("Warning: failed to update last known IP: %v", err)
        }
        return nil
    }

    if *status.LastKnownIP != currentIP {
        log.Printf("IP change detected for user %d: %s -> %s", userID, *status.LastKnownIP, currentIP)

        newCount, err := s.securityRepo.IncrementSuspicionCount(userID)
        if err != nil {
            return fmt.Errorf("failed to increment suspicion count: %w", err)
        }

        log.Printf("IP suspicion count for user %d: %d/%d", userID, newCount, s.maxSuspicion)

        if newCount > s.maxSuspicion {
            
            reason := fmt.Sprintf("Too many IP address changes (%d)", newCount)
            if err := s.securityRepo.LockAccount(userID, reason, nil); err != nil {
                return fmt.Errorf("failed to lock account: %w", err)
            }

            log.Printf("Account locked for user %d due to IP suspicion", userID)
            return fmt.Errorf("account locked due to suspicious activity: too many IP address changes")
        }

        if err := s.securityRepo.UpdateLastKnownIP(userID, currentIP); err != nil {
            log.Printf("Warning: failed to update last known IP: %v", err)
        }
    }

    return nil
}

func (s *IPTrackingService) UnlockAccount(userID int64) error {
    if err := s.securityRepo.UnlockAccount(userID); err != nil {
        return fmt.Errorf("failed to unlock account: %w", err)
    }

    log.Printf("Account unlocked for user %d", userID)
    return nil
}

func (s *IPTrackingService) ResetSuspicionCount(userID int64) error {
    if err := s.securityRepo.ResetSuspicionCount(userID); err != nil {
        return fmt.Errorf("failed to reset suspicion count: %w", err)
    }

    log.Printf("IP suspicion count reset for user %d", userID)
    return nil
}

func (s *IPTrackingService) GetSecurityStatus(userID int64) (*models.UserSecurityStatus, error) {
    return s.securityRepo.GetSecurityStatus(userID)
}

func (s *IPTrackingService) GetLoginHistory(userID int64, limit int) ([]*models.UserIPTracking, error) {
    return s.securityRepo.GetLoginHistory(userID, limit)
}

func (s *IPTrackingService) IsAccountLocked(userID int64) (bool, error) {
    return s.securityRepo.IsAccountLocked(userID)
}

func (s *IPTrackingService) ManualLock(userID int64, reason string, adminID int64) error {
    if err := s.securityRepo.LockAccount(userID, reason, &adminID); err != nil {
        return fmt.Errorf("failed to lock account: %w", err)
    }

    log.Printf("Account manually locked for user %d by admin %d: %s", userID, adminID, reason)
    return nil
}
