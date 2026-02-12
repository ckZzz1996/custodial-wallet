package audit

import (
	"time"
)

// AuditLog 审计日志
type AuditLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	AdminID     uint      `gorm:"index" json:"admin_id"`
	Module      string    `gorm:"type:varchar(50);index;not null" json:"module"`
	Action      string    `gorm:"type:varchar(50);index;not null" json:"action"`
	ResourceID  string    `gorm:"type:varchar(100)" json:"resource_id"`
	Description string    `gorm:"type:text" json:"description"`
	OldValue    string    `gorm:"type:text" json:"old_value"`
	NewValue    string    `gorm:"type:text" json:"new_value"`
	IP          string    `gorm:"type:varchar(45)" json:"ip"`
	UserAgent   string    `gorm:"type:varchar(500)" json:"user_agent"`
	Status      int       `gorm:"default:1" json:"status"` // 1=success, 0=failed
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

// Module 模块常量
const (
	ModuleAccount     = "account"
	ModuleWallet      = "wallet"
	ModuleTransaction = "transaction"
	ModuleDeposit     = "deposit"
	ModuleWithdrawal  = "withdrawal"
	ModuleAsset       = "asset"
	ModuleRisk        = "risk"
	ModuleAdmin       = "admin"
	ModuleSystem      = "system"
)

// Action 操作常量
const (
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionApprove  = "approve"
	ActionReject   = "reject"
	ActionLogin    = "login"
	ActionLogout   = "logout"
	ActionExport   = "export"
	ActionTransfer = "transfer"
)

// TableName 表名
func (AuditLog) TableName() string {
	return "audit_logs"
}
