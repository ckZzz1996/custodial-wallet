package wallet

import (
	"errors"

	"gorm.io/gorm"
)

// Repository 钱包仓储接口
type Repository interface {
	// Wallet
	CreateWallet(wallet *Wallet) error
	GetWalletByID(id uint) (*Wallet, error)
	GetWalletByUUID(uuid string) (*Wallet, error)
	ListWalletsByUserID(userID uint) ([]*Wallet, error)
	UpdateWallet(wallet *Wallet) error
	DeleteWallet(id uint) error

	// Address
	CreateAddress(address *Address) error
	GetAddressByID(id uint) (*Address, error)
	GetAddressByAddress(chain Chain, addr string) (*Address, error)
	ListAddressesByWalletID(walletID uint) ([]*Address, error)
	ListAddressesByUserID(userID uint, chain Chain) ([]*Address, error)
	GetAvailableDepositAddress(userID uint, chain Chain) (*Address, error)
	UpdateAddress(address *Address) error

	// Balance
	CreateBalance(balance *Balance) error
	GetBalance(userID uint, chain Chain, currency string) (*Balance, error)
	ListBalancesByUserID(userID uint) ([]*Balance, error)
	UpdateBalance(balance *Balance) error
	IncrementBalance(userID uint, chain Chain, currency string, amount string) error
	DecrementBalance(userID uint, chain Chain, currency string, amount string) error
	FreezeBalance(userID uint, chain Chain, currency string, amount string) error
	UnfreezeBalance(userID uint, chain Chain, currency string, amount string) error

	// AddressBook
	CreateAddressBook(entry *AddressBook) error
	GetAddressBookByID(id uint) (*AddressBook, error)
	ListAddressBookByUserID(userID uint) ([]*AddressBook, error)
	UpdateAddressBook(entry *AddressBook) error
	DeleteAddressBook(id uint) error
	IsWhitelisted(userID uint, chain Chain, address string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建钱包仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateWallet 创建钱包
func (r *repository) CreateWallet(wallet *Wallet) error {
	return r.db.Create(wallet).Error
}

// GetWalletByID 通过ID获取钱包
func (r *repository) GetWalletByID(id uint) (*Wallet, error) {
	var wallet Wallet
	if err := r.db.Preload("Addresses").First(&wallet, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

// GetWalletByUUID 通过UUID获取钱包
func (r *repository) GetWalletByUUID(uuid string) (*Wallet, error) {
	var wallet Wallet
	if err := r.db.Preload("Addresses").Where("uuid = ?", uuid).First(&wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

// ListWalletsByUserID 获取用户的钱包列表
func (r *repository) ListWalletsByUserID(userID uint) ([]*Wallet, error) {
	var wallets []*Wallet
	if err := r.db.Preload("Addresses").Where("user_id = ?", userID).Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

// UpdateWallet 更新钱包
func (r *repository) UpdateWallet(wallet *Wallet) error {
	return r.db.Save(wallet).Error
}

// DeleteWallet 删除钱包
func (r *repository) DeleteWallet(id uint) error {
	return r.db.Delete(&Wallet{}, id).Error
}

// CreateAddress 创建地址
func (r *repository) CreateAddress(address *Address) error {
	return r.db.Create(address).Error
}

// GetAddressByID 通过ID获取地址
func (r *repository) GetAddressByID(id uint) (*Address, error) {
	var address Address
	if err := r.db.First(&address, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &address, nil
}

// GetAddressByAddress 通过地址获取
func (r *repository) GetAddressByAddress(chain Chain, addr string) (*Address, error) {
	var address Address
	if err := r.db.Where("chain = ? AND address = ?", chain, addr).First(&address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &address, nil
}

// ListAddressesByWalletID 获取钱包的地址列表
func (r *repository) ListAddressesByWalletID(walletID uint) ([]*Address, error) {
	var addresses []*Address
	if err := r.db.Where("wallet_id = ?", walletID).Find(&addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}

// ListAddressesByUserID 获取用户的地址列表
func (r *repository) ListAddressesByUserID(userID uint, chain Chain) ([]*Address, error) {
	var addresses []*Address
	query := r.db.Where("user_id = ?", userID)
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if err := query.Find(&addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}

// GetAvailableDepositAddress 获取可用的充值地址
func (r *repository) GetAvailableDepositAddress(userID uint, chain Chain) (*Address, error) {
	var address Address
	if err := r.db.Where("user_id = ? AND chain = ? AND type = ? AND status = ?",
		userID, chain, AddressTypeDeposit, AddressStatusActive).First(&address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &address, nil
}

// UpdateAddress 更新地址
func (r *repository) UpdateAddress(address *Address) error {
	return r.db.Save(address).Error
}

// CreateBalance 创建余额
func (r *repository) CreateBalance(balance *Balance) error {
	return r.db.Create(balance).Error
}

// GetBalance 获取余额
func (r *repository) GetBalance(userID uint, chain Chain, currency string) (*Balance, error) {
	var balance Balance
	if err := r.db.Where("user_id = ? AND chain = ? AND currency = ?",
		userID, chain, currency).First(&balance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &balance, nil
}

// ListBalancesByUserID 获取用户余额列表
func (r *repository) ListBalancesByUserID(userID uint) ([]*Balance, error) {
	var balances []*Balance
	if err := r.db.Where("user_id = ?", userID).Find(&balances).Error; err != nil {
		return nil, err
	}
	return balances, nil
}

// UpdateBalance 更新余额
func (r *repository) UpdateBalance(balance *Balance) error {
	return r.db.Save(balance).Error
}

// IncrementBalance 增加余额
func (r *repository) IncrementBalance(userID uint, chain Chain, currency string, amount string) error {
	return r.db.Model(&Balance{}).
		Where("user_id = ? AND chain = ? AND currency = ?", userID, chain, currency).
		Update("available", gorm.Expr("available + ?", amount)).Error
}

// DecrementBalance 减少余额
func (r *repository) DecrementBalance(userID uint, chain Chain, currency string, amount string) error {
	return r.db.Model(&Balance{}).
		Where("user_id = ? AND chain = ? AND currency = ?", userID, chain, currency).
		Where("available >= ?", amount).
		Update("available", gorm.Expr("available - ?", amount)).Error
}

// FreezeBalance 冻结余额
func (r *repository) FreezeBalance(userID uint, chain Chain, currency string, amount string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Balance{}).
			Where("user_id = ? AND chain = ? AND currency = ?", userID, chain, currency).
			Where("available >= ?", amount).
			Update("available", gorm.Expr("available - ?", amount)).Error; err != nil {
			return err
		}
		return tx.Model(&Balance{}).
			Where("user_id = ? AND chain = ? AND currency = ?", userID, chain, currency).
			Update("frozen", gorm.Expr("frozen + ?", amount)).Error
	})
}

// UnfreezeBalance 解冻余额
func (r *repository) UnfreezeBalance(userID uint, chain Chain, currency string, amount string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Balance{}).
			Where("user_id = ? AND chain = ? AND currency = ?", userID, chain, currency).
			Where("frozen >= ?", amount).
			Update("frozen", gorm.Expr("frozen - ?", amount)).Error; err != nil {
			return err
		}
		return tx.Model(&Balance{}).
			Where("user_id = ? AND chain = ? AND currency = ?", userID, chain, currency).
			Update("available", gorm.Expr("available + ?", amount)).Error
	})
}

// CreateAddressBook 创建地址簿条目
func (r *repository) CreateAddressBook(entry *AddressBook) error {
	return r.db.Create(entry).Error
}

// GetAddressBookByID 获取地址簿条目
func (r *repository) GetAddressBookByID(id uint) (*AddressBook, error) {
	var entry AddressBook
	if err := r.db.First(&entry, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

// ListAddressBookByUserID 获取用户地址簿
func (r *repository) ListAddressBookByUserID(userID uint) ([]*AddressBook, error) {
	var entries []*AddressBook
	if err := r.db.Where("user_id = ?", userID).Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// UpdateAddressBook 更新地址簿条目
func (r *repository) UpdateAddressBook(entry *AddressBook) error {
	return r.db.Save(entry).Error
}

// DeleteAddressBook 删除地址簿条目
func (r *repository) DeleteAddressBook(id uint) error {
	return r.db.Delete(&AddressBook{}, id).Error
}

// IsWhitelisted 检查地址是否在白名单
func (r *repository) IsWhitelisted(userID uint, chain Chain, address string) (bool, error) {
	var count int64
	if err := r.db.Model(&AddressBook{}).
		Where("user_id = ? AND chain = ? AND address = ? AND is_whitelist = ?",
			userID, chain, address, true).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
