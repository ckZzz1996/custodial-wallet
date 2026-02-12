package asset

import (
	"time"
)

// Asset 资产配置
type Asset struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Chain           string    `gorm:"type:varchar(20);not null;index" json:"chain"`
	Symbol          string    `gorm:"type:varchar(20);not null;index" json:"symbol"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	ContractAddress string    `gorm:"type:varchar(255)" json:"contract_address"`
	Decimals        int       `gorm:"default:18" json:"decimals"`
	Type            AssetType `gorm:"type:varchar(20);not null" json:"type"`
	IconURL         string    `gorm:"type:varchar(500)" json:"icon_url"`
	MinDeposit      string    `gorm:"type:decimal(36,18);default:0" json:"min_deposit"`
	MinWithdrawal   string    `gorm:"type:decimal(36,18);default:0" json:"min_withdrawal"`
	WithdrawalFee   string    `gorm:"type:decimal(36,18);default:0" json:"withdrawal_fee"`
	DepositEnabled  bool      `gorm:"default:true" json:"deposit_enabled"`
	WithdrawEnabled bool      `gorm:"default:true" json:"withdraw_enabled"`
	Status          int       `gorm:"default:1" json:"status"`
	SortOrder       int       `gorm:"default:0" json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AssetType 资产类型
type AssetType string

const (
	AssetTypeNative AssetType = "native" // 原生币
	AssetTypeERC20  AssetType = "erc20"  // ERC20
	AssetTypeTRC20  AssetType = "trc20"  // TRC20
	AssetTypeBEP20  AssetType = "bep20"  // BEP20
)

// AssetPrice 资产价格
type AssetPrice struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AssetID   uint      `gorm:"index;not null" json:"asset_id"`
	Symbol    string    `gorm:"type:varchar(20);index;not null" json:"symbol"`
	PriceUSD  string    `gorm:"type:decimal(36,18)" json:"price_usd"`
	PriceBTC  string    `gorm:"type:decimal(36,18)" json:"price_btc"`
	PriceETH  string    `gorm:"type:decimal(36,18)" json:"price_eth"`
	Change24h string    `gorm:"type:decimal(10,4)" json:"change_24h"`
	Volume24h string    `gorm:"type:decimal(36,18)" json:"volume_24h"`
	MarketCap string    `gorm:"type:decimal(36,18)" json:"market_cap"`
	Source    string    `gorm:"type:varchar(50)" json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserAsset 用户资产
type UserAsset struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	AssetID       uint      `gorm:"index;not null" json:"asset_id"`
	Chain         string    `gorm:"type:varchar(20);not null" json:"chain"`
	Symbol        string    `gorm:"type:varchar(20);not null" json:"symbol"`
	Available     string    `gorm:"type:decimal(36,18);default:0" json:"available"`
	Frozen        string    `gorm:"type:decimal(36,18);default:0" json:"frozen"`
	Pending       string    `gorm:"type:decimal(36,18);default:0" json:"pending"`
	TotalDeposit  string    `gorm:"type:decimal(36,18);default:0" json:"total_deposit"`
	TotalWithdraw string    `gorm:"type:decimal(36,18);default:0" json:"total_withdraw"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName 表名
func (Asset) TableName() string {
	return "assets"
}

func (AssetPrice) TableName() string {
	return "asset_prices"
}

func (UserAsset) TableName() string {
	return "user_assets"
}
