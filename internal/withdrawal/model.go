package withdrawal

import (
	"time"

	"gorm.io/gorm"
)

// Withdrawal 提现记录
type Withdrawal struct {
	ID              uint             `gorm:"primaryKey" json:"id"`
	UUID            string           `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	UserID          uint             `gorm:"index;not null" json:"user_id"`
	WalletID        uint             `gorm:"index" json:"wallet_id"`
	Chain           string           `gorm:"type:varchar(20);index;not null" json:"chain"`
	TxHash          string           `gorm:"type:varchar(255);index" json:"tx_hash"`
	FromAddress     string           `gorm:"type:varchar(255)" json:"from_address"`
	ToAddress       string           `gorm:"type:varchar(255);not null" json:"to_address"`
	Currency        string           `gorm:"type:varchar(20);not null" json:"currency"`
	ContractAddress string           `gorm:"type:varchar(255)" json:"contract_address"`
	Amount          string           `gorm:"type:decimal(36,18);not null" json:"amount"`
	Fee             string           `gorm:"type:decimal(36,18)" json:"fee"`
	Status          WithdrawalStatus `gorm:"type:smallint;default:0;index" json:"status"`
	RiskLevel       int              `gorm:"default:0" json:"risk_level"`
	RiskReview      bool             `gorm:"default:false" json:"risk_review"`
	ManualReview    bool             `gorm:"default:false" json:"manual_review"`
	ReviewedBy      uint             `gorm:"default:0" json:"reviewed_by"`
	ReviewedAt      *time.Time       `json:"reviewed_at"`
	ReviewNote      string           `gorm:"type:text" json:"review_note"`
	Confirmations   int              `gorm:"default:0" json:"confirmations"`
	BlockNumber     uint64           `gorm:"default:0" json:"block_number"`
	Memo            string           `gorm:"type:varchar(500)" json:"memo"`
	ErrorMsg        string           `gorm:"type:text" json:"error_msg"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CompletedAt     *time.Time       `json:"completed_at"`
	DeletedAt       gorm.DeletedAt   `gorm:"index" json:"-"`
}

// WithdrawalStatus 提现状态
type WithdrawalStatus int

const (
	WithdrawalStatusPending      WithdrawalStatus = 0  // 待处理
	WithdrawalStatusRiskReview   WithdrawalStatus = 1  // 风控审核
	WithdrawalStatusManualReview WithdrawalStatus = 2  // 人工审核
	WithdrawalStatusApproved     WithdrawalStatus = 3  // 已通过
	WithdrawalStatusProcessing   WithdrawalStatus = 4  // 处理中
	WithdrawalStatusBroadcast    WithdrawalStatus = 5  // 已广播
	WithdrawalStatusConfirming   WithdrawalStatus = 6  // 确认中
	WithdrawalStatusCompleted    WithdrawalStatus = 7  // 已完成
	WithdrawalStatusFailed       WithdrawalStatus = 8  // 失败
	WithdrawalStatusRejected     WithdrawalStatus = 9  // 已拒绝
	WithdrawalStatusCancelled    WithdrawalStatus = 10 // 已取消
)

// WithdrawalLimit 提现限额
type WithdrawalLimit struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index" json:"user_id"` // 0 = 全局
	Chain         string    `gorm:"type:varchar(20)" json:"chain"`
	Currency      string    `gorm:"type:varchar(20)" json:"currency"`
	MinAmount     string    `gorm:"type:decimal(36,18);default:0" json:"min_amount"`
	MaxAmount     string    `gorm:"type:decimal(36,18)" json:"max_amount"`
	DailyLimit    string    `gorm:"type:decimal(36,18)" json:"daily_limit"`
	MonthlyLimit  string    `gorm:"type:decimal(36,18)" json:"monthly_limit"`
	RequireReview string    `gorm:"type:decimal(36,18)" json:"require_review"` // 超过此金额需要人工审核
	Status        int       `gorm:"default:1" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName 表名
func (Withdrawal) TableName() string {
	return "withdrawals"
}

func (WithdrawalLimit) TableName() string {
	return "withdrawal_limits"
}
