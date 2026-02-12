package riskcontrol

import (
	"time"
)

// RiskRule 风控规则
type RiskRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Type        RuleType  `gorm:"type:varchar(50);not null" json:"type"`
	Chain       string    `gorm:"type:varchar(20)" json:"chain"`
	Currency    string    `gorm:"type:varchar(20)" json:"currency"`
	Condition   string    `gorm:"type:text;not null" json:"condition"` // JSON condition
	Action      string    `gorm:"type:varchar(50);not null" json:"action"`
	RiskLevel   int       `gorm:"default:1" json:"risk_level"`
	Priority    int       `gorm:"default:0" json:"priority"`
	Status      int       `gorm:"default:1" json:"status"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RuleType 规则类型
type RuleType string

const (
	RuleTypeAmountLimit      RuleType = "amount_limit"
	RuleTypeFrequencyLimit   RuleType = "frequency_limit"
	RuleTypeAddressBlacklist RuleType = "address_blacklist"
	RuleTypeAddressWhitelist RuleType = "address_whitelist"
	RuleTypeGeoRestriction   RuleType = "geo_restriction"
	RuleTypeDeviceLimit      RuleType = "device_limit"
	RuleTypeKYCRequired      RuleType = "kyc_required"
	RuleTypeCustom           RuleType = "custom"
)

// Blacklist 黑名单
type Blacklist struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Type      string     `gorm:"type:varchar(50);not null;index" json:"type"` // address, user, ip, device
	Value     string     `gorm:"type:varchar(255);not null;index" json:"value"`
	Chain     string     `gorm:"type:varchar(20)" json:"chain"`
	Reason    string     `gorm:"type:text" json:"reason"`
	Source    string     `gorm:"type:varchar(100)" json:"source"`
	ExpiresAt *time.Time `json:"expires_at"`
	Status    int        `gorm:"default:1" json:"status"`
	CreatedBy uint       `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// RiskLog 风控日志
type RiskLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Action      string    `gorm:"type:varchar(50);not null" json:"action"`
	RuleID      uint      `gorm:"index" json:"rule_id"`
	RuleName    string    `gorm:"type:varchar(100)" json:"rule_name"`
	RiskLevel   int       `json:"risk_level"`
	Result      string    `gorm:"type:varchar(50)" json:"result"` // pass, block, review
	Details     string    `gorm:"type:text" json:"details"`       // JSON
	IP          string    `gorm:"type:varchar(45)" json:"ip"`
	UserAgent   string    `gorm:"type:varchar(500)" json:"user_agent"`
	RequestData string    `gorm:"type:text" json:"request_data"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserRiskProfile 用户风险画像
type UserRiskProfile struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            uint       `gorm:"uniqueIndex;not null" json:"user_id"`
	RiskScore         int        `gorm:"default:0" json:"risk_score"`
	TotalWithdrawals  string     `gorm:"type:decimal(36,18);default:0" json:"total_withdrawals"`
	TotalDeposits     string     `gorm:"type:decimal(36,18);default:0" json:"total_deposits"`
	WithdrawalCount   int        `gorm:"default:0" json:"withdrawal_count"`
	DepositCount      int        `gorm:"default:0" json:"deposit_count"`
	FailedWithdrawals int        `gorm:"default:0" json:"failed_withdrawals"`
	BlockedCount      int        `gorm:"default:0" json:"blocked_count"`
	LastWithdrawalAt  *time.Time `json:"last_withdrawal_at"`
	LastDepositAt     *time.Time `json:"last_deposit_at"`
	LastRiskCheckAt   *time.Time `json:"last_risk_check_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TableName 表名
func (RiskRule) TableName() string {
	return "risk_rules"
}

func (Blacklist) TableName() string {
	return "blacklists"
}

func (RiskLog) TableName() string {
	return "risk_logs"
}

func (UserRiskProfile) TableName() string {
	return "user_risk_profiles"
}
