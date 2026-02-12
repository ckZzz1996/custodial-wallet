package wallet

import (
	"errors"

	"custodial-wallet/internal/keymanager"
	"custodial-wallet/pkg/logger"

	"github.com/google/uuid"
)

var (
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrAddressNotFound     = errors.New("address not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// Service 钱包服务接口
type Service interface {
	CreateWallet(userID uint, name string, walletType WalletType) (*Wallet, error)
	GetWallet(walletID uint) (*Wallet, error)
	GetWalletByUUID(uuid string) (*Wallet, error)
	ListWallets(userID uint) ([]*Wallet, error)
	UpdateWallet(walletID uint, name string) (*Wallet, error)
	DeleteWallet(walletID uint) error

	GenerateAddress(walletID uint, chain Chain, label string) (*Address, error)
	GetAddress(addressID uint) (*Address, error)
	GetAddressByAddress(chain Chain, address string) (*Address, error)
	ListAddresses(walletID uint) ([]*Address, error)
	GetDepositAddress(userID uint, chain Chain) (*Address, error)

	GetBalance(userID uint, chain Chain, currency string) (*Balance, error)
	ListBalances(userID uint) ([]*Balance, error)

	AddToAddressBook(userID uint, chain Chain, address, label string, isWhitelist bool) (*AddressBook, error)
	ListAddressBook(userID uint) ([]*AddressBook, error)
	RemoveFromAddressBook(id uint) error
	IsAddressWhitelisted(userID uint, chain Chain, address string) (bool, error)
}

type service struct {
	repo       Repository
	keyManager keymanager.Service
}

// NewService 创建钱包服务
func NewService(repo Repository, keyManager keymanager.Service) Service {
	return &service{
		repo:       repo,
		keyManager: keyManager,
	}
}

// CreateWallet 创建钱包
func (s *service) CreateWallet(userID uint, name string, walletType WalletType) (*Wallet, error) {
	wallet := &Wallet{
		UUID:   uuid.New().String(),
		UserID: userID,
		Name:   name,
		Type:   walletType,
		Status: WalletStatusActive,
	}

	if err := s.repo.CreateWallet(wallet); err != nil {
		return nil, err
	}

	logger.Infof("Wallet created: %s for user %d", wallet.UUID, userID)
	return wallet, nil
}

// GetWallet 获取钱包
func (s *service) GetWallet(walletID uint) (*Wallet, error) {
	wallet, err := s.repo.GetWalletByID(walletID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}
	return wallet, nil
}

// GetWalletByUUID 通过UUID获取钱包
func (s *service) GetWalletByUUID(uuid string) (*Wallet, error) {
	wallet, err := s.repo.GetWalletByUUID(uuid)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}
	return wallet, nil
}

// ListWallets 列出用户钱包
func (s *service) ListWallets(userID uint) ([]*Wallet, error) {
	return s.repo.ListWalletsByUserID(userID)
}

// UpdateWallet 更新钱包
func (s *service) UpdateWallet(walletID uint, name string) (*Wallet, error) {
	wallet, err := s.repo.GetWalletByID(walletID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	wallet.Name = name
	if err := s.repo.UpdateWallet(wallet); err != nil {
		return nil, err
	}

	return wallet, nil
}

// DeleteWallet 删除钱包
func (s *service) DeleteWallet(walletID uint) error {
	return s.repo.DeleteWallet(walletID)
}

// GenerateAddress 生成地址
func (s *service) GenerateAddress(walletID uint, chain Chain, label string) (*Address, error) {
	wallet, err := s.repo.GetWalletByID(walletID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	// 使用密钥管理器生成地址
	addr, derivationPath, err := s.keyManager.GenerateAddress(wallet.UserID, string(chain))
	if err != nil {
		return nil, err
	}

	address := &Address{
		UUID:           uuid.New().String(),
		WalletID:       walletID,
		UserID:         wallet.UserID,
		Chain:          chain,
		Address:        addr,
		Label:          label,
		DerivationPath: derivationPath,
		Type:           AddressTypeDeposit,
		Status:         AddressStatusActive,
	}

	if err := s.repo.CreateAddress(address); err != nil {
		return nil, err
	}

	// 初始化余额记录
	currency := s.getDefaultCurrency(chain)
	balance, _ := s.repo.GetBalance(wallet.UserID, chain, currency)
	if balance == nil {
		balance = &Balance{
			WalletID:  walletID,
			UserID:    wallet.UserID,
			Chain:     chain,
			Currency:  currency,
			Available: "0",
			Frozen:    "0",
			Pending:   "0",
		}
		_ = s.repo.CreateBalance(balance)
	}

	logger.Infof("Address generated: %s on %s for wallet %d", addr, chain, walletID)
	return address, nil
}

func (s *service) getDefaultCurrency(chain Chain) string {
	switch chain {
	case ChainBitcoin:
		return "BTC"
	case ChainEthereum:
		return "ETH"
	case ChainTron:
		return "TRX"
	case ChainBSC:
		return "BNB"
	case ChainPolygon:
		return "MATIC"
	default:
		return "UNKNOWN"
	}
}

// GetAddress 获取地址
func (s *service) GetAddress(addressID uint) (*Address, error) {
	address, err := s.repo.GetAddressByID(addressID)
	if err != nil {
		return nil, err
	}
	if address == nil {
		return nil, ErrAddressNotFound
	}
	return address, nil
}

// GetAddressByAddress 通过地址字符串获取
func (s *service) GetAddressByAddress(chain Chain, address string) (*Address, error) {
	return s.repo.GetAddressByAddress(chain, address)
}

// ListAddresses 列出地址
func (s *service) ListAddresses(walletID uint) ([]*Address, error) {
	return s.repo.ListAddressesByWalletID(walletID)
}

// GetDepositAddress 获取充值地址
func (s *service) GetDepositAddress(userID uint, chain Chain) (*Address, error) {
	address, err := s.repo.GetAvailableDepositAddress(userID, chain)
	if err != nil {
		return nil, err
	}
	if address == nil {
		return nil, ErrAddressNotFound
	}
	return address, nil
}

// GetBalance 获取余额
func (s *service) GetBalance(userID uint, chain Chain, currency string) (*Balance, error) {
	balance, err := s.repo.GetBalance(userID, chain, currency)
	if err != nil {
		return nil, err
	}
	if balance == nil {
		return &Balance{
			UserID:    userID,
			Chain:     chain,
			Currency:  currency,
			Available: "0",
			Frozen:    "0",
			Pending:   "0",
		}, nil
	}
	return balance, nil
}

// ListBalances 列出余额
func (s *service) ListBalances(userID uint) ([]*Balance, error) {
	return s.repo.ListBalancesByUserID(userID)
}

// AddToAddressBook 添加到地址簿
func (s *service) AddToAddressBook(userID uint, chain Chain, address, label string, isWhitelist bool) (*AddressBook, error) {
	entry := &AddressBook{
		UserID:      userID,
		Chain:       chain,
		Address:     address,
		Label:       label,
		IsWhitelist: isWhitelist,
	}

	if err := s.repo.CreateAddressBook(entry); err != nil {
		return nil, err
	}

	return entry, nil
}

// ListAddressBook 列出地址簿
func (s *service) ListAddressBook(userID uint) ([]*AddressBook, error) {
	return s.repo.ListAddressBookByUserID(userID)
}

// RemoveFromAddressBook 从地址簿删除
func (s *service) RemoveFromAddressBook(id uint) error {
	return s.repo.DeleteAddressBook(id)
}

// IsAddressWhitelisted 检查地址是否在白名单
func (s *service) IsAddressWhitelisted(userID uint, chain Chain, address string) (bool, error) {
	return s.repo.IsWhitelisted(userID, chain, address)
}
