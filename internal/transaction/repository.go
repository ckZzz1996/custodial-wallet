package transaction

import (
	"errors"

	"gorm.io/gorm"
)

// Repository 交易仓储接口
type Repository interface {
	Create(tx *Transaction) error
	GetByID(id uint) (*Transaction, error)
	GetByUUID(uuid string) (*Transaction, error)
	GetByTxHash(chain, txHash string) (*Transaction, error)
	ListByUserID(userID uint, page, pageSize int) ([]*Transaction, int64, error)
	ListByStatus(status TxStatus, limit int) ([]*Transaction, error)
	ListPendingConfirmation(chain string, limit int) ([]*Transaction, error)
	Update(tx *Transaction) error
	UpdateStatus(id uint, status TxStatus, errorMsg string) error
	UpdateConfirmations(id uint, confirmations int, blockNumber uint64, blockHash string) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建交易仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create 创建交易
func (r *repository) Create(tx *Transaction) error {
	return r.db.Create(tx).Error
}

// GetByID 通过ID获取交易
func (r *repository) GetByID(id uint) (*Transaction, error) {
	var tx Transaction
	if err := r.db.First(&tx, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

// GetByUUID 通过UUID获取交易
func (r *repository) GetByUUID(uuid string) (*Transaction, error) {
	var tx Transaction
	if err := r.db.Where("uuid = ?", uuid).First(&tx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

// GetByTxHash 通过交易哈希获取交易
func (r *repository) GetByTxHash(chain, txHash string) (*Transaction, error) {
	var tx Transaction
	if err := r.db.Where("chain = ? AND tx_hash = ?", chain, txHash).First(&tx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

// ListByUserID 列出用户交易
func (r *repository) ListByUserID(userID uint, page, pageSize int) ([]*Transaction, int64, error) {
	var txs []*Transaction
	var total int64

	r.db.Model(&Transaction{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * pageSize
	if err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&txs).Error; err != nil {
		return nil, 0, err
	}

	return txs, total, nil
}

// ListByStatus 根据状态列出交易
func (r *repository) ListByStatus(status TxStatus, limit int) ([]*Transaction, error) {
	var txs []*Transaction
	if err := r.db.Where("status = ?", status).
		Order("created_at ASC").
		Limit(limit).
		Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// ListPendingConfirmation 列出待确认的交易
func (r *repository) ListPendingConfirmation(chain string, limit int) ([]*Transaction, error) {
	var txs []*Transaction
	query := r.db.Where("status IN ?", []TxStatus{TxStatusBroadcast, TxStatusConfirming})
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if err := query.Order("created_at ASC").Limit(limit).Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// Update 更新交易
func (r *repository) Update(tx *Transaction) error {
	return r.db.Save(tx).Error
}

// UpdateStatus 更新交易状态
func (r *repository) UpdateStatus(id uint, status TxStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	return r.db.Model(&Transaction{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateConfirmations 更新确认数
func (r *repository) UpdateConfirmations(id uint, confirmations int, blockNumber uint64, blockHash string) error {
	return r.db.Model(&Transaction{}).Where("id = ?", id).Updates(map[string]interface{}{
		"confirmations": confirmations,
		"block_number":  blockNumber,
		"block_hash":    blockHash,
	}).Error
}
