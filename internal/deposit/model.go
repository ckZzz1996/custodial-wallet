package deposit

import (
	"time"

	"gorm.io/gorm"
)

// Deposit 充值记录
type Deposit struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UUID            string         `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	WalletID        uint           `gorm:"index" json:"wallet_id"`
	AddressID       uint           `gorm:"index" json:"address_id"`
	Chain           string         `gorm:"type:varchar(20);index;not null" json:"chain"`
	TxHash          string         `gorm:"type:varchar(255);uniqueIndex" json:"tx_hash"`
	FromAddress     string         `gorm:"type:varchar(255)" json:"from_address"`
	ToAddress       string         `gorm:"type:varchar(255);index" json:"to_address"`
	Currency        string         `gorm:"type:varchar(20);not null" json:"currency"`
	ContractAddress string         `gorm:"type:varchar(255)" json:"contract_address"`
	Amount          string         `gorm:"type:decimal(36,18);not null" json:"amount"`
	Fee             string         `gorm:"type:decimal(36,18)" json:"fee"`
	Status          DepositStatus  `gorm:"type:smallint;default:0;index" json:"status"`
	Confirmations   int            `gorm:"default:0" json:"confirmations"`
	BlockNumber     uint64         `gorm:"default:0" json:"block_number"`
	BlockHash       string         `gorm:"type:varchar(255)" json:"block_hash"`
	Credited        bool           `gorm:"default:false" json:"credited"`
	CreditedAt      *time.Time     `json:"credited_at"`
	Swept           bool           `gorm:"default:false" json:"swept"`
	SweepTxHash     string         `gorm:"type:varchar(255)" json:"sweep_tx_hash"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// DepositStatus 充值状态
type DepositStatus int

const (
	DepositStatusPending    DepositStatus = 0 // 待确认
	DepositStatusConfirming DepositStatus = 1 // 确认中
	DepositStatusConfirmed  DepositStatus = 2 // 已确认
	DepositStatusCredited   DepositStatus = 3 // 已入账
	DepositStatusFailed     DepositStatus = 4 // 失败
)

// DepositAddress 充值地址分配
type DepositAddress struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	Chain      string     `gorm:"type:varchar(20);index;not null" json:"chain"`
	Address    string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"address"`
	Label      string     `gorm:"type:varchar(100)" json:"label"`
	Status     int        `gorm:"default:1" json:"status"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// SweepTask 归集任务
type SweepTask struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Chain       string    `gorm:"type:varchar(20);index;not null" json:"chain"`
	FromAddress string    `gorm:"type:varchar(255);not null" json:"from_address"`
	ToAddress   string    `gorm:"type:varchar(255);not null" json:"to_address"`
	Currency    string    `gorm:"type:varchar(20);not null" json:"currency"`
	Amount      string    `gorm:"type:decimal(36,18);not null" json:"amount"`
	TxHash      string    `gorm:"type:varchar(255)" json:"tx_hash"`
	Status      int       `gorm:"default:0" json:"status"` // 0=pending, 1=success, 2=failed
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ScanProgress 记录每条链的最后已扫描区块
type ScanProgress struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Chain       string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"chain"`
	LastScanned uint64    `gorm:"default:0" json:"last_scanned"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 表名
func (Deposit) TableName() string {
	return "deposits"
}

func (DepositAddress) TableName() string {
	return "deposit_addresses"
}

func (SweepTask) TableName() string {
	return "sweep_tasks"
}

func (ScanProgress) TableName() string {
	return "scan_progress"
}
