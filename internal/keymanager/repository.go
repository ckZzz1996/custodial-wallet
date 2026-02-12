package keymanager

import (
	"errors"

	"gorm.io/gorm"
)

// Repository 密钥仓储接口
type Repository interface {
	CreateKey(key *EncryptedKey) error
	GetKeyByID(id uint) (*EncryptedKey, error)
	GetKeyByAddress(chain, address string) (*EncryptedKey, error)
	GetMasterKey(userID uint, chain string) (*EncryptedKey, error)
	ListKeysByUserID(userID uint, chain string) ([]*EncryptedKey, error)
	UpdateKey(key *EncryptedKey) error
	DeleteKey(id uint) error
	GetNextDerivationIndex(userID uint, chain string) (int, error)

	CreateSignatureRequest(req *SignatureRequest) error
	GetSignatureRequestByID(id uint) (*SignatureRequest, error)
	GetSignatureRequestByRequestID(requestID string) (*SignatureRequest, error)
	ListSignatureRequestsByUserID(userID uint, limit int) ([]*SignatureRequest, error)
	UpdateSignatureRequest(req *SignatureRequest) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建密钥仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateKey 创建密钥
func (r *repository) CreateKey(key *EncryptedKey) error {
	return r.db.Create(key).Error
}

// GetKeyByID 通过ID获取密钥
func (r *repository) GetKeyByID(id uint) (*EncryptedKey, error) {
	var key EncryptedKey
	if err := r.db.First(&key, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

// GetKeyByAddress 通过地址获取密钥
func (r *repository) GetKeyByAddress(chain, address string) (*EncryptedKey, error) {
	var key EncryptedKey
	if err := r.db.Where("chain = ? AND address = ?", chain, address).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

// GetMasterKey 获取主密钥
func (r *repository) GetMasterKey(userID uint, chain string) (*EncryptedKey, error) {
	var key EncryptedKey
	if err := r.db.Where("user_id = ? AND chain = ? AND key_type = ?",
		userID, chain, "master").First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

// ListKeysByUserID 列出用户密钥
func (r *repository) ListKeysByUserID(userID uint, chain string) ([]*EncryptedKey, error) {
	var keys []*EncryptedKey
	query := r.db.Where("user_id = ?", userID)
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if err := query.Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// UpdateKey 更新密钥
func (r *repository) UpdateKey(key *EncryptedKey) error {
	return r.db.Save(key).Error
}

// DeleteKey 删除密钥
func (r *repository) DeleteKey(id uint) error {
	return r.db.Delete(&EncryptedKey{}, id).Error
}

// GetNextDerivationIndex 获取下一个派生索引
func (r *repository) GetNextDerivationIndex(userID uint, chain string) (int, error) {
	var count int64
	if err := r.db.Model(&EncryptedKey{}).
		Where("user_id = ? AND chain = ? AND key_type = ?", userID, chain, "derived").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// CreateSignatureRequest 创建签名请求
func (r *repository) CreateSignatureRequest(req *SignatureRequest) error {
	return r.db.Create(req).Error
}

// GetSignatureRequestByID 通过ID获取签名请求
func (r *repository) GetSignatureRequestByID(id uint) (*SignatureRequest, error) {
	var req SignatureRequest
	if err := r.db.First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

// GetSignatureRequestByRequestID 通过请求ID获取签名请求
func (r *repository) GetSignatureRequestByRequestID(requestID string) (*SignatureRequest, error) {
	var req SignatureRequest
	if err := r.db.Where("request_id = ?", requestID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

// ListSignatureRequestsByUserID 列出用户签名请求
func (r *repository) ListSignatureRequestsByUserID(userID uint, limit int) ([]*SignatureRequest, error) {
	var reqs []*SignatureRequest
	if err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&reqs).Error; err != nil {
		return nil, err
	}
	return reqs, nil
}

// UpdateSignatureRequest 更新签名请求
func (r *repository) UpdateSignatureRequest(req *SignatureRequest) error {
	return r.db.Save(req).Error
}
