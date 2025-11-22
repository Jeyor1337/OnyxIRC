-- OnyxIRC Database Schema
-- MySQL Database Schema for IRC Server
-- Character Set: utf8mb4 (supports full Unicode including emoji)

CREATE DATABASE IF NOT EXISTS onyxirc CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE onyxirc;

-- =======================
-- USERS TABLE
-- =======================
CREATE TABLE users (
    user_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash CHAR(64) NOT NULL COMMENT 'SHA-256 hash (64 hex chars)',
    password_salt CHAR(32) NOT NULL COMMENT 'Random salt for password hashing',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    last_login_time TIMESTAMP NULL,
    INDEX idx_username (username),
    INDEX idx_active_users (is_active, last_login_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- USER IP TRACKING TABLE
-- =======================
CREATE TABLE user_ip_tracking (
    tracking_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    ip_address VARCHAR(45) NOT NULL COMMENT 'Supports both IPv4 and IPv6',
    login_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_successful BOOLEAN DEFAULT FALSE,
    user_agent VARCHAR(255) NULL COMMENT 'Client information',
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    INDEX idx_user_login (user_id, login_timestamp DESC),
    INDEX idx_ip_address (ip_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- USER SECURITY STATUS TABLE
-- =======================
CREATE TABLE user_security_status (
    user_id BIGINT PRIMARY KEY,
    last_known_ip VARCHAR(45) NULL,
    ip_suspicion_count INT DEFAULT 0,
    account_locked BOOLEAN DEFAULT FALSE,
    lock_reason VARCHAR(255) NULL,
    locked_at TIMESTAMP NULL,
    locked_by BIGINT NULL COMMENT 'Admin user_id who locked the account',
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (locked_by) REFERENCES users(user_id) ON DELETE SET NULL,
    INDEX idx_locked_accounts (account_locked, locked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- CHANNELS TABLE
-- =======================
CREATE TABLE channels (
    channel_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    channel_name VARCHAR(100) NOT NULL UNIQUE,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    topic TEXT NULL,
    is_private BOOLEAN DEFAULT FALSE,
    max_members INT DEFAULT 1000,
    FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE RESTRICT,
    INDEX idx_channel_name (channel_name),
    INDEX idx_private_channels (is_private)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- CHANNEL MEMBERS TABLE
-- =======================
CREATE TABLE channel_members (
    membership_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    role ENUM('member', 'moderator', 'owner') DEFAULT 'member',
    is_muted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (channel_id) REFERENCES channels(channel_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    UNIQUE KEY unique_membership (channel_id, user_id),
    INDEX idx_channel_users (channel_id, role),
    INDEX idx_user_channels (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- MESSAGES TABLE
-- =======================
CREATE TABLE messages (
    message_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    message_content TEXT NOT NULL COMMENT 'Encrypted message content',
    message_hash CHAR(64) NULL COMMENT 'SHA-256 hash for integrity verification',
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (channel_id) REFERENCES channels(channel_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    INDEX idx_channel_messages (channel_id, sent_at DESC),
    INDEX idx_user_messages (user_id, sent_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- DIRECT MESSAGES TABLE
-- =======================
CREATE TABLE direct_messages (
    dm_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    sender_id BIGINT NOT NULL,
    recipient_id BIGINT NOT NULL,
    message_content TEXT NOT NULL COMMENT 'Encrypted message content',
    message_hash CHAR(64) NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_read BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (sender_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id) REFERENCES users(user_id) ON DELETE CASCADE,
    INDEX idx_recipient_messages (recipient_id, is_read, sent_at DESC),
    INDEX idx_sender_messages (sender_id, sent_at DESC),
    INDEX idx_conversation (sender_id, recipient_id, sent_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- SERVER CONFIG TABLE
-- =======================
CREATE TABLE server_config (
    config_key VARCHAR(100) PRIMARY KEY,
    config_value TEXT NOT NULL,
    description TEXT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by BIGINT NULL,
    FOREIGN KEY (updated_by) REFERENCES users(user_id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- ADMIN ACTION LOG TABLE
-- =======================
CREATE TABLE admin_action_log (
    log_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    admin_id BIGINT NOT NULL,
    action_type VARCHAR(50) NOT NULL COMMENT 'kick, ban, unlock, makeadmin, etc.',
    target_user_id BIGINT NULL,
    target_channel_id BIGINT NULL,
    action_details TEXT NULL COMMENT 'JSON or text description',
    performed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (admin_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (target_user_id) REFERENCES users(user_id) ON DELETE SET NULL,
    FOREIGN KEY (target_channel_id) REFERENCES channels(channel_id) ON DELETE SET NULL,
    INDEX idx_admin_actions (admin_id, performed_at DESC),
    INDEX idx_target_user (target_user_id, performed_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- USER BANS TABLE
-- =======================
CREATE TABLE user_bans (
    ban_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    banned_by BIGINT NOT NULL,
    reason TEXT NULL,
    banned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL COMMENT 'NULL for permanent ban',
    is_active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (banned_by) REFERENCES users(user_id) ON DELETE RESTRICT,
    INDEX idx_active_bans (user_id, is_active, expires_at),
    INDEX idx_ban_expiry (expires_at, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- SESSION TOKENS TABLE
-- =======================
CREATE TABLE session_tokens (
    token_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_hash CHAR(64) NOT NULL COMMENT 'SHA-256 hash of session token',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    ip_address VARCHAR(45) NOT NULL,
    is_valid BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    INDEX idx_user_sessions (user_id, is_valid, expires_at),
    INDEX idx_token_lookup (token_hash, is_valid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =======================
-- INITIAL DATA / SEED
-- =======================

-- Insert default server configuration
INSERT INTO server_config (config_key, config_value, description) VALUES
('server.version', '1.0.0', 'Server version'),
('security.max_ip_suspicion', '3', 'Maximum IP suspicion count before account lock'),
('security.session_timeout_seconds', '3600', 'Session timeout in seconds'),
('security.password_min_length', '8', 'Minimum password length'),
('server.max_connections', '1000', 'Maximum concurrent connections'),
('server.message_history_limit', '1000', 'Maximum messages to keep per channel');

-- =======================
-- TRIGGERS
-- =======================

-- Trigger: Auto-create security status entry when user is created
DELIMITER //
CREATE TRIGGER after_user_insert
AFTER INSERT ON users
FOR EACH ROW
BEGIN
    INSERT INTO user_security_status (user_id, ip_suspicion_count, account_locked)
    VALUES (NEW.user_id, 0, FALSE);
END//
DELIMITER ;

-- Trigger: Clean up expired sessions periodically (called manually or via event)
DELIMITER //
CREATE PROCEDURE cleanup_expired_sessions()
BEGIN
    UPDATE session_tokens
    SET is_valid = FALSE
    WHERE expires_at < NOW() AND is_valid = TRUE;
END//
DELIMITER ;

-- Trigger: Auto-expire bans
DELIMITER //
CREATE PROCEDURE cleanup_expired_bans()
BEGIN
    UPDATE user_bans
    SET is_active = FALSE
    WHERE expires_at IS NOT NULL
      AND expires_at < NOW()
      AND is_active = TRUE;
END//
DELIMITER ;

-- =======================
-- EVENT SCHEDULER (Optional - requires MySQL Event Scheduler)
-- =======================
-- Enable event scheduler: SET GLOBAL event_scheduler = ON;

-- Event: Clean expired sessions every hour
-- CREATE EVENT IF NOT EXISTS cleanup_sessions_hourly
-- ON SCHEDULE EVERY 1 HOUR
-- DO CALL cleanup_expired_sessions();

-- Event: Clean expired bans every hour
-- CREATE EVENT IF NOT EXISTS cleanup_bans_hourly
-- ON SCHEDULE EVERY 1 HOUR
-- DO CALL cleanup_expired_bans();
