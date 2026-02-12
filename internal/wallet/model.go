package wallet

import (
	"time"

	"gorm.io/gorm"
)

// Wallet 钱包模型
type Wallet struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UUID      string         `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Name      string         `gorm:"type:varchar(100)" json:"name"`
	Type      WalletType     `gorm:"type:smallint;default:1" json:"type"`
	Status    WalletStatus   `gorm:"type:smallint;default:1" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Addresses []Address `gorm:"foreignKey:WalletID" json:"addresses,omitempty"`
}

// WalletType 钱包类型
type WalletType int

const (
	WalletTypeHot  WalletType = 1 // 热钱包
	WalletTypeCold WalletType = 2 // 冷钱包
)

// WalletStatus 钱包状态
type WalletStatus int

const (
	WalletStatusInactive WalletStatus = 0
	WalletStatusActive   WalletStatus = 1
	WalletStatusFrozen   WalletStatus = 2
)

// Address 地址模型
type Address struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	UUID           string        `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	WalletID       uint          `gorm:"index;not null" json:"wallet_id"`
	UserID         uint          `gorm:"index;not null" json:"user_id"`
	Chain          Chain         `gorm:"type:varchar(20);index;not null" json:"chain"`
	Address        string        `gorm:"type:varchar(255);index;not null" json:"address"`
	Label          string        `gorm:"type:varchar(100)" json:"label"`
	DerivationPath string        `gorm:"type:varchar(100)" json:"derivation_path"`
	Type           AddressType   `gorm:"type:smallint;default:1" json:"type"`
	Status         AddressStatus `gorm:"type:smallint;default:1" json:"status"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Chain 区块链类型
type Chain string

const (
	ChainBitcoin  Chain = "bitcoin"
	ChainEthereum Chain = "ethereum"
	ChainTron     Chain = "tron"
	ChainBSC      Chain = "bsc"
	ChainPolygon  Chain = "polygon"
)

// AddressType 地址类型
type AddressType int

const (
	AddressTypeDeposit    AddressType = 1 // 充值地址
	AddressTypeWithdrawal AddressType = 2 // 提现地址
	AddressTypeInternal   AddressType = 3 // 内部地址
)

// AddressStatus 地址状态
type AddressStatus int

const (
	AddressStatusInactive AddressStatus = 0
	AddressStatusActive   AddressStatus = 1
	AddressStatusUsed     AddressStatus = 2
)

// Balance 余额模型
type Balance struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	WalletID     uint      `gorm:"index;not null" json:"wallet_id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	Chain        Chain     `gorm:"type:varchar(20);not null" json:"chain"`
	Currency     string    `gorm:"type:varchar(20);not null" json:"currency"`
	ContractAddr string    `gorm:"type:varchar(255)" json:"contract_address"`
	Available    string    `gorm:"type:decimal(36,18);default:0" json:"available"`
	Frozen       string    `gorm:"type:decimal(36,18);default:0" json:"frozen"`
	Pending      string    `gorm:"type:decimal(36,18);default:0" json:"pending"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AddressBook 地址簿
type AddressBook struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	Chain       Chain     `gorm:"type:varchar(20);not null" json:"chain"`
	Address     string    `gorm:"type:varchar(255);not null" json:"address"`
	Label       string    `gorm:"type:varchar(100)" json:"label"`
	Memo        string    `gorm:"type:varchar(200)" json:"memo"`
	IsWhitelist bool      `gorm:"default:false" json:"is_whitelist"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 表名
func (Wallet) TableName() string {
	return "wallets"
}

func (Address) TableName() string {
	return "addresses"
}

func (Balance) TableName() string {
	return "balances"
}

func (AddressBook) TableName() string {
	return "address_books"
}
