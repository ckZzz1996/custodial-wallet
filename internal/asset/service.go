package asset

import (
	"errors"

	"custodial-wallet/pkg/logger"

	"github.com/shopspring/decimal"
)

var (
	ErrAssetNotFound = errors.New("asset not found")
	ErrAssetDisabled = errors.New("asset is disabled")
)

// Service 资产服务接口
type Service interface {
	// 资产配置
	CreateAsset(asset *Asset) error
	GetAsset(chain, symbol string) (*Asset, error)
	GetAssetByContract(chain, contractAddress string) (*Asset, error)
	ListAssets(chain string) ([]*Asset, error)
	ListEnabledAssets() ([]*Asset, error)
	UpdateAsset(asset *Asset) error
	EnableAsset(assetID uint) error
	DisableAsset(assetID uint) error

	// 价格
	UpdatePrice(symbol, priceUSD string) error
	GetPrice(symbol string) (*AssetPrice, error)
	GetPrices(symbols []string) (map[string]*AssetPrice, error)

	// 用户资产
	GetUserAssets(userID uint) ([]*UserAssetDetail, error)
	GetUserAsset(userID uint, chain, symbol string) (*UserAssetDetail, error)
	GetUserTotalValue(userID uint) (string, error)
}

type service struct {
	repo Repository
}

// NewService 创建资产服务
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// UserAssetDetail 用户资产详情
type UserAssetDetail struct {
	Asset    *Asset      `json:"asset"`
	Balance  *UserAsset  `json:"balance"`
	Price    *AssetPrice `json:"price"`
	ValueUSD string      `json:"value_usd"`
}

// CreateAsset 创建资产
func (s *service) CreateAsset(asset *Asset) error {
	existing, _ := s.repo.GetAsset(asset.Chain, asset.Symbol)
	if existing != nil {
		return errors.New("asset already exists")
	}

	if err := s.repo.CreateAsset(asset); err != nil {
		return err
	}
	logger.Infof("Asset created: %s on %s", asset.Symbol, asset.Chain)
	return nil
}

// GetAsset 获取资产
func (s *service) GetAsset(chain, symbol string) (*Asset, error) {
	asset, err := s.repo.GetAsset(chain, symbol)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, ErrAssetNotFound
	}
	return asset, nil
}

// GetAssetByContract 通过合约地址获取资产
func (s *service) GetAssetByContract(chain, contractAddress string) (*Asset, error) {
	asset, err := s.repo.GetAssetByContract(chain, contractAddress)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, ErrAssetNotFound
	}
	return asset, nil
}

// ListAssets 列出资产
func (s *service) ListAssets(chain string) ([]*Asset, error) {
	return s.repo.ListAssets(chain, -1)
}

// ListEnabledAssets 列出启用的资产
func (s *service) ListEnabledAssets() ([]*Asset, error) {
	return s.repo.ListEnabledAssets()
}

// UpdateAsset 更新资产
func (s *service) UpdateAsset(asset *Asset) error {
	return s.repo.UpdateAsset(asset)
}

// EnableAsset 启用资产
func (s *service) EnableAsset(assetID uint) error {
	asset, err := s.repo.GetAssetByID(assetID)
	if err != nil {
		return err
	}
	if asset == nil {
		return ErrAssetNotFound
	}
	asset.Status = 1
	return s.repo.UpdateAsset(asset)
}

// DisableAsset 禁用资产
func (s *service) DisableAsset(assetID uint) error {
	asset, err := s.repo.GetAssetByID(assetID)
	if err != nil {
		return err
	}
	if asset == nil {
		return ErrAssetNotFound
	}
	asset.Status = 0
	return s.repo.UpdateAsset(asset)
}

// UpdatePrice 更新价格
func (s *service) UpdatePrice(symbol, priceUSD string) error {
	price := &AssetPrice{
		Symbol:   symbol,
		PriceUSD: priceUSD,
	}
	return s.repo.UpdatePrice(price)
}

// GetPrice 获取价格
func (s *service) GetPrice(symbol string) (*AssetPrice, error) {
	return s.repo.GetPrice(symbol)
}

// GetPrices 批量获取价格
func (s *service) GetPrices(symbols []string) (map[string]*AssetPrice, error) {
	prices, err := s.repo.ListPrices(symbols)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*AssetPrice)
	for _, p := range prices {
		result[p.Symbol] = p
	}
	return result, nil
}

// GetUserAssets 获取用户资产列表
func (s *service) GetUserAssets(userID uint) ([]*UserAssetDetail, error) {
	// 获取所有启用的资产
	assets, err := s.repo.ListEnabledAssets()
	if err != nil {
		return nil, err
	}

	// 获取用户资产
	userAssets, err := s.repo.ListUserAssets(userID)
	if err != nil {
		return nil, err
	}
	userAssetMap := make(map[string]*UserAsset)
	for _, ua := range userAssets {
		key := ua.Chain + "_" + ua.Symbol
		userAssetMap[key] = ua
	}

	// 获取价格
	symbols := make([]string, 0, len(assets))
	for _, a := range assets {
		symbols = append(symbols, a.Symbol)
	}
	prices, _ := s.GetPrices(symbols)

	// 组装结果
	result := make([]*UserAssetDetail, 0, len(assets))
	for _, asset := range assets {
		key := asset.Chain + "_" + asset.Symbol
		userAsset := userAssetMap[key]
		if userAsset == nil {
			userAsset = &UserAsset{
				UserID:    userID,
				Chain:     asset.Chain,
				Symbol:    asset.Symbol,
				Available: "0",
				Frozen:    "0",
				Pending:   "0",
			}
		}

		detail := &UserAssetDetail{
			Asset:   asset,
			Balance: userAsset,
			Price:   prices[asset.Symbol],
		}

		// 计算USD价值
		if detail.Price != nil && detail.Price.PriceUSD != "" {
			available, _ := decimal.NewFromString(userAsset.Available)
			priceUSD, _ := decimal.NewFromString(detail.Price.PriceUSD)
			detail.ValueUSD = available.Mul(priceUSD).String()
		}

		result = append(result, detail)
	}

	return result, nil
}

// GetUserAsset 获取用户单个资产
func (s *service) GetUserAsset(userID uint, chain, symbol string) (*UserAssetDetail, error) {
	asset, err := s.repo.GetAsset(chain, symbol)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, ErrAssetNotFound
	}

	userAsset, err := s.repo.GetUserAsset(userID, chain, symbol)
	if err != nil {
		return nil, err
	}
	if userAsset == nil {
		userAsset = &UserAsset{
			UserID:    userID,
			Chain:     chain,
			Symbol:    symbol,
			Available: "0",
			Frozen:    "0",
			Pending:   "0",
		}
	}

	price, _ := s.repo.GetPrice(symbol)

	detail := &UserAssetDetail{
		Asset:   asset,
		Balance: userAsset,
		Price:   price,
	}

	if price != nil && price.PriceUSD != "" {
		available, _ := decimal.NewFromString(userAsset.Available)
		priceUSD, _ := decimal.NewFromString(price.PriceUSD)
		detail.ValueUSD = available.Mul(priceUSD).String()
	}

	return detail, nil
}

// GetUserTotalValue 获取用户总资产价值
func (s *service) GetUserTotalValue(userID uint) (string, error) {
	assets, err := s.GetUserAssets(userID)
	if err != nil {
		return "0", err
	}

	total := decimal.Zero
	for _, a := range assets {
		if a.ValueUSD != "" {
			value, _ := decimal.NewFromString(a.ValueUSD)
			total = total.Add(value)
		}
	}

	return total.String(), nil
}
