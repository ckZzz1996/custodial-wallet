package transaction

import (
	"time"

	"gorm.io/gorm"
)

// Transaction 交易模型
type Transaction struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UUID            string         `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	WalletID        uint           `gorm:"index" json:"wallet_id"`
	Chain           string         `gorm:"type:varchar(20);index;not null" json:"chain"`
	TxHash          string         `gorm:"type:varchar(255);index" json:"tx_hash"`
	FromAddress     string         `gorm:"type:varchar(255);index" json:"from_address"`
	ToAddress       string         `gorm:"type:varchar(255);index" json:"to_address"`
	Currency        string         `gorm:"type:varchar(20);not null" json:"currency"`
	ContractAddress string         `gorm:"type:varchar(255)" json:"contract_address"`
	Amount          string         `gorm:"type:decimal(36,18);not null" json:"amount"`
	Fee             string         `gorm:"type:decimal(36,18)" json:"fee"`
	GasPrice        string         `gorm:"type:decimal(36,18)" json:"gas_price"`
	GasLimit        uint64         `gorm:"default:0" json:"gas_limit"`
	GasUsed         uint64         `gorm:"default:0" json:"gas_used"`
	Nonce           uint64         `gorm:"default:0" json:"nonce"`
	Type            TxType         `gorm:"type:smallint;not null" json:"type"`
	Status          TxStatus       `gorm:"type:smallint;default:0;index" json:"status"`
	Confirmations   int            `gorm:"default:0" json:"confirmations"`
	BlockNumber     uint64         `gorm:"default:0" json:"block_number"`
	BlockHash       string         `gorm:"type:varchar(255)" json:"block_hash"`
	RawTx           string         `gorm:"type:text" json:"-"`
	SignedTx        string         `gorm:"type:text" json:"-"`
	ErrorMsg        string         `gorm:"type:text" json:"error_msg"`
	Memo            string         `gorm:"type:varchar(500)" json:"memo"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	ConfirmedAt     *time.Time     `json:"confirmed_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TxType 交易类型
type TxType int

const (
	TxTypeDeposit    TxType = 1 // 充值
	TxTypeWithdrawal TxType = 2 // 提现
	TxTypeInternal   TxType = 3 // 内部转账
	TxTypeSweep      TxType = 4 // 归集
	TxTypeFee        TxType = 5 // 手续费
)

// TxStatus 交易状态
type TxStatus int

const (
	TxStatusPending    TxStatus = 0 // 待处理
	TxStatusSigned     TxStatus = 1 // 已签名
	TxStatusBroadcast  TxStatus = 2 // 已广播
	TxStatusConfirming TxStatus = 3 // 确认中
	TxStatusConfirmed  TxStatus = 4 // 已确认
	TxStatusFailed     TxStatus = 5 // 失败
	TxStatusCancelled  TxStatus = 6 // 已取消
)

// TableName 表名
func (Transaction) TableName() string {
	return "transactions"
}
