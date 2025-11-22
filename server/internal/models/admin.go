package models

import "time"

type AdminActionLog struct {
    LogID           int64      `json:"log_id"`
    AdminID         int64      `json:"admin_id"`
    ActionType      string     `json:"action_type"`
    TargetUserID    *int64     `json:"target_user_id,omitempty"`
    TargetChannelID *int64     `json:"target_channel_id,omitempty"`
    ActionDetails   *string    `json:"action_details,omitempty"`
    PerformedAt     time.Time  `json:"performed_at"`
}

type ServerConfig struct {
    ConfigKey   string     `json:"config_key"`
    ConfigValue string     `json:"config_value"`
    Description *string    `json:"description,omitempty"`
    UpdatedAt   time.Time  `json:"updated_at"`
    UpdatedBy   *int64     `json:"updated_by,omitempty"`
}
