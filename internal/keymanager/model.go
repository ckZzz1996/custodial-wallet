package keymanager

import (
	"time"

	"gorm.io/gorm"
)

// EncryptedKey 加密密钥模型
type EncryptedKey struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	Chain          string         `gorm:"type:varchar(20);not null" json:"chain"`
	PublicKey      string         `gorm:"type:text;not null" json:"public_key"`
	EncryptedPriv  string         `gorm:"type:text;not null" json:"-"`
	KeyType        string         `gorm:"type:varchar(20);not null" json:"key_type"` // master, derived
	DerivationPath string         `gorm:"type:varchar(100)" json:"derivation_path"`
	Address        string         `gorm:"type:varchar(255);index" json:"address"`
	Status         int            `gorm:"default:1" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// SignatureRequest 签名请求模型
type SignatureRequest struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	RequestID   string     `gorm:"type:varchar(36);uniqueIndex;not null" json:"request_id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	KeyID       uint       `gorm:"index;not null" json:"key_id"`
	Chain       string     `gorm:"type:varchar(20);not null" json:"chain"`
	TxHash      string     `gorm:"type:varchar(255)" json:"tx_hash"`
	RawTx       string     `gorm:"type:text;not null" json:"raw_tx"`
	SignedTx    string     `gorm:"type:text" json:"signed_tx"`
	Status      SignStatus `gorm:"type:smallint;default:0" json:"status"`
	ErrorMsg    string     `gorm:"type:text" json:"error_msg"`
	RequestedAt time.Time  `json:"requested_at"`
	SignedAt    *time.Time `json:"signed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// SignStatus 签名状态
type SignStatus int

const (
	SignStatusPending  SignStatus = 0
	SignStatusSigned   SignStatus = 1
	SignStatusFailed   SignStatus = 2
	SignStatusRejected SignStatus = 3
)

// TableName 表名
func (EncryptedKey) TableName() string {
	return "encrypted_keys"
}

func (SignatureRequest) TableName() string {
	return "signature_requests"
}
