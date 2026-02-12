package asset

import (
	"errors"

	"gorm.io/gorm"
)

// Repository 资产仓储接口
type Repository interface {
	// Asset
	CreateAsset(asset *Asset) error
	GetAssetByID(id uint) (*Asset, error)
	GetAsset(chain, symbol string) (*Asset, error)
	GetAssetByContract(chain, contractAddress string) (*Asset, error)
	ListAssets(chain string, status int) ([]*Asset, error)
	ListEnabledAssets() ([]*Asset, error)
	UpdateAsset(asset *Asset) error
	DeleteAsset(id uint) error

	// Price
	UpdatePrice(price *AssetPrice) error
	GetPrice(symbol string) (*AssetPrice, error)
	ListPrices(symbols []string) ([]*AssetPrice, error)

	// User Asset
	CreateUserAsset(ua *UserAsset) error
	GetUserAsset(userID uint, chain, symbol string) (*UserAsset, error)
	ListUserAssets(userID uint) ([]*UserAsset, error)
	UpdateUserAsset(ua *UserAsset) error
	IncrementUserAsset(userID uint, chain, symbol, field string, amount string) error
	DecrementUserAsset(userID uint, chain, symbol, field string, amount string) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建资产仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateAsset 创建资产
func (r *repository) CreateAsset(asset *Asset) error {
	return r.db.Create(asset).Error
}

// GetAssetByID 获取资产
func (r *repository) GetAssetByID(id uint) (*Asset, error) {
	var asset Asset
	if err := r.db.First(&asset, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &asset, nil
}

// GetAsset 获取资产
func (r *repository) GetAsset(chain, symbol string) (*Asset, error) {
	var asset Asset
	if err := r.db.Where("chain = ? AND symbol = ?", chain, symbol).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &asset, nil
}

// GetAssetByContract 通过合约地址获取资产
func (r *repository) GetAssetByContract(chain, contractAddress string) (*Asset, error) {
	var asset Asset
	if err := r.db.Where("chain = ? AND contract_address = ?", chain, contractAddress).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &asset, nil
}

// ListAssets 列出资产
func (r *repository) ListAssets(chain string, status int) ([]*Asset, error) {
	var assets []*Asset
	query := r.db.Model(&Asset{})
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("sort_order ASC, id ASC").Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}

// ListEnabledAssets 列出启用的资产
func (r *repository) ListEnabledAssets() ([]*Asset, error) {
	return r.ListAssets("", 1)
}

// UpdateAsset 更新资产
func (r *repository) UpdateAsset(asset *Asset) error {
	return r.db.Save(asset).Error
}

// DeleteAsset 删除资产
func (r *repository) DeleteAsset(id uint) error {
	return r.db.Delete(&Asset{}, id).Error
}

// UpdatePrice 更新价格
func (r *repository) UpdatePrice(price *AssetPrice) error {
	var existing AssetPrice
	err := r.db.Where("symbol = ?", price.Symbol).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(price).Error
	}
	if err != nil {
		return err
	}
	price.ID = existing.ID
	return r.db.Save(price).Error
}

// GetPrice 获取价格
func (r *repository) GetPrice(symbol string) (*AssetPrice, error) {
	var price AssetPrice
	if err := r.db.Where("symbol = ?", symbol).First(&price).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &price, nil
}

// ListPrices 列出价格
func (r *repository) ListPrices(symbols []string) ([]*AssetPrice, error) {
	var prices []*AssetPrice
	if err := r.db.Where("symbol IN ?", symbols).Find(&prices).Error; err != nil {
		return nil, err
	}
	return prices, nil
}

// CreateUserAsset 创建用户资产
func (r *repository) CreateUserAsset(ua *UserAsset) error {
	return r.db.Create(ua).Error
}

// GetUserAsset 获取用户资产
func (r *repository) GetUserAsset(userID uint, chain, symbol string) (*UserAsset, error) {
	var ua UserAsset
	if err := r.db.Where("user_id = ? AND chain = ? AND symbol = ?",
		userID, chain, symbol).First(&ua).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ua, nil
}

// ListUserAssets 列出用户资产
func (r *repository) ListUserAssets(userID uint) ([]*UserAsset, error) {
	var assets []*UserAsset
	if err := r.db.Where("user_id = ?", userID).Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}

// UpdateUserAsset 更新用户资产
func (r *repository) UpdateUserAsset(ua *UserAsset) error {
	return r.db.Save(ua).Error
}

// IncrementUserAsset 增加用户资产
func (r *repository) IncrementUserAsset(userID uint, chain, symbol, field string, amount string) error {
	return r.db.Model(&UserAsset{}).
		Where("user_id = ? AND chain = ? AND symbol = ?", userID, chain, symbol).
		Update(field, gorm.Expr(field+" + ?", amount)).Error
}

// DecrementUserAsset 减少用户资产
func (r *repository) DecrementUserAsset(userID uint, chain, symbol, field string, amount string) error {
	return r.db.Model(&UserAsset{}).
		Where("user_id = ? AND chain = ? AND symbol = ?", userID, chain, symbol).
		Where(field+" >= ?", amount).
		Update(field, gorm.Expr(field+" - ?", amount)).Error
}
