package notification

import (
	"time"
)

// Notification 通知记录
type Notification struct {
	ID         uint             `gorm:"primaryKey" json:"id"`
	UserID     uint             `gorm:"index;not null" json:"user_id"`
	Type       NotificationType `gorm:"type:varchar(50);not null" json:"type"`
	Channel    Channel          `gorm:"type:varchar(20);not null" json:"channel"`
	Title      string           `gorm:"type:varchar(200)" json:"title"`
	Content    string           `gorm:"type:text;not null" json:"content"`
	Data       string           `gorm:"type:text" json:"data"`   // JSON
	Status     int              `gorm:"default:0" json:"status"` // 0=pending, 1=sent, 2=failed, 3=read
	SendAt     *time.Time       `json:"send_at"`
	ReadAt     *time.Time       `json:"read_at"`
	ErrorMsg   string           `gorm:"type:text" json:"error_msg"`
	RetryCount int              `gorm:"default:0" json:"retry_count"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// NotificationType 通知类型
type NotificationType string

const (
	NotificationTypeDeposit       NotificationType = "deposit"
	NotificationTypeWithdrawal    NotificationType = "withdrawal"
	NotificationTypeLogin         NotificationType = "login"
	NotificationTypeSecurityAlert NotificationType = "security_alert"
	NotificationTypeSystemNotice  NotificationType = "system_notice"
	NotificationTypeKYCStatus     NotificationType = "kyc_status"
)

// Channel 通知渠道
type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelSMS     Channel = "sms"
	ChannelWebhook Channel = "webhook"
	ChannelInApp   Channel = "in_app"
	ChannelPush    Channel = "push"
)

// NotificationTemplate 通知模板
type NotificationTemplate struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	Type      NotificationType `gorm:"type:varchar(50);uniqueIndex:idx_type_channel;not null" json:"type"`
	Channel   Channel          `gorm:"type:varchar(20);uniqueIndex:idx_type_channel;not null" json:"channel"`
	Title     string           `gorm:"type:varchar(200)" json:"title"`
	Content   string           `gorm:"type:text;not null" json:"content"`
	Variables string           `gorm:"type:text" json:"variables"` // JSON array of variable names
	Status    int              `gorm:"default:1" json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// UserNotificationSetting 用户通知设置
type UserNotificationSetting struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	UserID    uint             `gorm:"uniqueIndex:idx_user_type;not null" json:"user_id"`
	Type      NotificationType `gorm:"type:varchar(50);uniqueIndex:idx_user_type;not null" json:"type"`
	Email     bool             `gorm:"default:true" json:"email"`
	SMS       bool             `gorm:"default:false" json:"sms"`
	InApp     bool             `gorm:"default:true" json:"in_app"`
	Push      bool             `gorm:"default:true" json:"push"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	URL       string    `gorm:"type:varchar(500);not null" json:"url"`
	Secret    string    `gorm:"type:varchar(255)" json:"secret"`
	Events    string    `gorm:"type:text" json:"events"`  // JSON array
	Headers   string    `gorm:"type:text" json:"headers"` // JSON object
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (Notification) TableName() string {
	return "notifications"
}

func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

func (UserNotificationSetting) TableName() string {
	return "user_notification_settings"
}

func (WebhookConfig) TableName() string {
	return "webhook_configs"
}
