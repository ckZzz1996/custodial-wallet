package account

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UUID         string         `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Phone        string         `gorm:"type:varchar(20);index" json:"phone"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	Status       UserStatus     `gorm:"type:smallint;default:1" json:"status"`
	KYCStatus    KYCStatus      `gorm:"type:smallint;default:0" json:"kyc_status"`
	KYCLevel     int            `gorm:"default:0" json:"kyc_level"`
	TwoFAEnabled bool           `gorm:"default:false" json:"two_fa_enabled"`
	TwoFASecret  string         `gorm:"type:varchar(255)" json:"-"`
	LastLoginAt  *time.Time     `json:"last_login_at"`
	LastLoginIP  string         `gorm:"type:varchar(45)" json:"last_login_ip"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// UserStatus 用户状态
type UserStatus int

const (
	UserStatusInactive UserStatus = 0
	UserStatusActive   UserStatus = 1
	UserStatusFrozen   UserStatus = 2
	UserStatusBanned   UserStatus = 3
)

// KYCStatus KYC状态
type KYCStatus int

const (
	KYCStatusNone     KYCStatus = 0
	KYCStatusPending  KYCStatus = 1
	KYCStatusApproved KYCStatus = 2
	KYCStatusRejected KYCStatus = 3
)

// UserProfile 用户资料
type UserProfile struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"uniqueIndex;not null" json:"user_id"`
	FirstName   string     `gorm:"type:varchar(100)" json:"first_name"`
	LastName    string     `gorm:"type:varchar(100)" json:"last_name"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Country     string     `gorm:"type:varchar(2)" json:"country"`
	Address     string     `gorm:"type:varchar(500)" json:"address"`
	City        string     `gorm:"type:varchar(100)" json:"city"`
	PostalCode  string     `gorm:"type:varchar(20)" json:"postal_code"`
	IDType      string     `gorm:"type:varchar(50)" json:"id_type"`
	IDNumber    string     `gorm:"type:varchar(100)" json:"id_number"`
	IDFrontURL  string     `gorm:"type:varchar(500)" json:"id_front_url"`
	IDBackURL   string     `gorm:"type:varchar(500)" json:"id_back_url"`
	SelfieURL   string     `gorm:"type:varchar(500)" json:"selfie_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// APIKey API密钥
type APIKey struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	Name        string     `gorm:"type:varchar(100);not null" json:"name"`
	Key         string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"key"`
	Secret      string     `gorm:"type:varchar(255);not null" json:"-"`
	Permissions string     `gorm:"type:text" json:"permissions"`  // JSON array
	IPWhitelist string     `gorm:"type:text" json:"ip_whitelist"` // JSON array
	Status      int        `gorm:"default:1" json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// LoginHistory 登录历史
type LoginHistory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	IP        string    `gorm:"type:varchar(45);not null" json:"ip"`
	UserAgent string    `gorm:"type:varchar(500)" json:"user_agent"`
	Device    string    `gorm:"type:varchar(100)" json:"device"`
	Location  string    `gorm:"type:varchar(200)" json:"location"`
	Status    int       `gorm:"default:1" json:"status"` // 1=success, 0=failed
	CreatedAt time.Time `json:"created_at"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

func (UserProfile) TableName() string {
	return "user_profiles"
}

func (APIKey) TableName() string {
	return "api_keys"
}

func (LoginHistory) TableName() string {
	return "login_histories"
}
