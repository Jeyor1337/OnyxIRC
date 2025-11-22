package config

import (
    "fmt"
    "os"
    "time"

    "gopkg.in/yaml.v3"
)

// Config represents the complete server configuration
type Config struct {
    Server     ServerConfig     `yaml:"server"`
    Database   DatabaseConfig   `yaml:"database"`
    Security   SecurityConfig   `yaml:"security"`
    ThreadPool ThreadPoolConfig `yaml:"threadpool"`
    Logging    LoggingConfig    `yaml:"logging"`
    Features   FeaturesConfig   `yaml:"features"`
}

// ServerConfig contains main server settings
type ServerConfig struct {
    Host           string        `yaml:"host"`
    Port           int           `yaml:"port"`
    MaxConnections int           `yaml:"max_connections"`
    ReadTimeout    time.Duration `yaml:"read_timeout"`
    WriteTimeout   time.Duration `yaml:"write_timeout"`
    ServerName     string        `yaml:"server_name"`
    MOTD           string        `yaml:"motd"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
    Host            string        `yaml:"host"`
    Port            int           `yaml:"port"`
    Name            string        `yaml:"name"`
    User            string        `yaml:"user"`
    Password        string        `yaml:"password"`
    MaxOpenConns    int           `yaml:"max_open_conns"`
    MaxIdleConns    int           `yaml:"max_idle_conns"`
    ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
    RSAKeySize             int    `yaml:"rsa_key_size"`
    RSAPrivateKeyPath      string `yaml:"rsa_private_key_path"`
    RSAPublicKeyPath       string `yaml:"rsa_public_key_path"`
    AESKeySize             int    `yaml:"aes_key_size"`
    AESMode                string `yaml:"aes_mode"`
    SessionTimeout         int    `yaml:"session_timeout"`
    MaxIPSuspicion         int    `yaml:"max_ip_suspicion"`
    EnableIPTracking       bool   `yaml:"enable_ip_tracking"`
    PasswordMinLength      int    `yaml:"password_min_length"`
    PasswordRequireSpecial bool   `yaml:"password_require_special"`
    MaxLoginAttempts       int    `yaml:"max_login_attempts"`
    LoginAttemptWindow     int    `yaml:"login_attempt_window"`
}

// ThreadPoolConfig contains thread pool settings
type ThreadPoolConfig struct {
    WorkerCount       int           `yaml:"worker_count"`
    QueueSize         int           `yaml:"queue_size"`
    MaxWorkers        int           `yaml:"max_workers"`
    WorkerIdleTimeout time.Duration `yaml:"worker_idle_timeout"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
    Level         string `yaml:"level"`
    Output        string `yaml:"output"`
    MaxSizeMB     int    `yaml:"max_size_mb"`
    MaxBackups    int    `yaml:"max_backups"`
    MaxAgeDays    int    `yaml:"max_age_days"`
    Compress      bool   `yaml:"compress"`
    ConsoleOutput bool   `yaml:"console_output"`
}

// FeaturesConfig contains feature flags
type FeaturesConfig struct {
    EnableMessageHistory  bool `yaml:"enable_message_history"`
    MaxMessageHistory     int  `yaml:"max_message_history"`
    EnableDirectMessages  bool `yaml:"enable_direct_messages"`
    EnableFileTransfer    bool `yaml:"enable_file_transfer"`
    MaxChannelNameLength  int  `yaml:"max_channel_name_length"`
    MaxChannelsPerUser    int  `yaml:"max_channels_per_user"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }

    // Expand environment variables in sensitive fields
    cfg.Database.Password = os.ExpandEnv(cfg.Database.Password)

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        return fmt.Errorf("invalid server port: %d", c.Server.Port)
    }

    if c.Database.Name == "" {
        return fmt.Errorf("database name is required")
    }

    if c.Security.RSAKeySize != 2048 && c.Security.RSAKeySize != 4096 {
        return fmt.Errorf("RSA key size must be 2048 or 4096")
    }

    if c.Security.AESKeySize != 256 {
        return fmt.Errorf("AES key size must be 256")
    }

    if c.Security.MaxIPSuspicion < 1 {
        return fmt.Errorf("max IP suspicion must be at least 1")
    }

    return nil
}
