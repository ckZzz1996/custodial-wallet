package withdrawal

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Repository 提现仓储接口
type Repository interface {
	Create(w *Withdrawal) error
	GetByID(id uint) (*Withdrawal, error)
	GetByUUID(uuid string) (*Withdrawal, error)
	GetByTxHash(txHash string) (*Withdrawal, error)
	ListByUserID(userID uint, page, pageSize int) ([]*Withdrawal, int64, error)
	ListByStatus(status WithdrawalStatus, limit int) ([]*Withdrawal, error)
	ListPendingReview(limit int) ([]*Withdrawal, error)
	ListPendingConfirmation(chain string, limit int) ([]*Withdrawal, error)
	Update(w *Withdrawal) error
	UpdateStatus(id uint, status WithdrawalStatus, errorMsg string) error

	GetUserDailyWithdrawal(userID uint, chain, currency string) (string, error)
	GetUserMonthlyWithdrawal(userID uint, chain, currency string) (string, error)

	CreateLimit(limit *WithdrawalLimit) error
	GetLimit(userID uint, chain, currency string) (*WithdrawalLimit, error)
	GetGlobalLimit(chain, currency string) (*WithdrawalLimit, error)
	UpdateLimit(limit *WithdrawalLimit) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建提现仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create 创建提现
func (r *repository) Create(w *Withdrawal) error {
	return r.db.Create(w).Error
}

// GetByID 通过ID获取提现
func (r *repository) GetByID(id uint) (*Withdrawal, error) {
	var w Withdrawal
	if err := r.db.First(&w, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// GetByUUID 通过UUID获取提现
func (r *repository) GetByUUID(uuid string) (*Withdrawal, error) {
	var w Withdrawal
	if err := r.db.Where("uuid = ?", uuid).First(&w).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// GetByTxHash 通过交易哈希获取提现
func (r *repository) GetByTxHash(txHash string) (*Withdrawal, error) {
	var w Withdrawal
	if err := r.db.Where("tx_hash = ?", txHash).First(&w).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// ListByUserID 列出用户提现
func (r *repository) ListByUserID(userID uint, page, pageSize int) ([]*Withdrawal, int64, error) {
	var withdrawals []*Withdrawal
	var total int64

	r.db.Model(&Withdrawal{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * pageSize
	if err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&withdrawals).Error; err != nil {
		return nil, 0, err
	}

	return withdrawals, total, nil
}

// ListByStatus 根据状态列出提现
func (r *repository) ListByStatus(status WithdrawalStatus, limit int) ([]*Withdrawal, error) {
	var withdrawals []*Withdrawal
	if err := r.db.Where("status = ?", status).
		Order("created_at ASC").
		Limit(limit).
		Find(&withdrawals).Error; err != nil {
		return nil, err
	}
	return withdrawals, nil
}

// ListPendingReview 列出待审核的提现
func (r *repository) ListPendingReview(limit int) ([]*Withdrawal, error) {
	var withdrawals []*Withdrawal
	if err := r.db.Where("status IN ?", []WithdrawalStatus{
		WithdrawalStatusRiskReview,
		WithdrawalStatusManualReview,
	}).Order("created_at ASC").Limit(limit).Find(&withdrawals).Error; err != nil {
		return nil, err
	}
	return withdrawals, nil
}

// ListPendingConfirmation 列出待确认的提现
func (r *repository) ListPendingConfirmation(chain string, limit int) ([]*Withdrawal, error) {
	var withdrawals []*Withdrawal
	query := r.db.Where("status IN ?", []WithdrawalStatus{
		WithdrawalStatusBroadcast,
		WithdrawalStatusConfirming,
	})
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if err := query.Order("created_at ASC").Limit(limit).Find(&withdrawals).Error; err != nil {
		return nil, err
	}
	return withdrawals, nil
}

// Update 更新提现
func (r *repository) Update(w *Withdrawal) error {
	return r.db.Save(w).Error
}

// UpdateStatus 更新提现状态
func (r *repository) UpdateStatus(id uint, status WithdrawalStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	if status == WithdrawalStatusCompleted {
		now := time.Now()
		updates["completed_at"] = &now
	}
	return r.db.Model(&Withdrawal{}).Where("id = ?", id).Updates(updates).Error
}

// GetUserDailyWithdrawal 获取用户今日提现总额
func (r *repository) GetUserDailyWithdrawal(userID uint, chain, currency string) (string, error) {
	var total string
	today := time.Now().Format("2006-01-02")
	err := r.db.Model(&Withdrawal{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND chain = ? AND currency = ? AND DATE(created_at) = ? AND status NOT IN ?",
			userID, chain, currency, today, []WithdrawalStatus{
				WithdrawalStatusRejected,
				WithdrawalStatusCancelled,
				WithdrawalStatusFailed,
			}).
		Scan(&total).Error
	return total, err
}

// GetUserMonthlyWithdrawal 获取用户本月提现总额
func (r *repository) GetUserMonthlyWithdrawal(userID uint, chain, currency string) (string, error) {
	var total string
	startOfMonth := time.Now().Format("2006-01") + "-01"
	err := r.db.Model(&Withdrawal{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND chain = ? AND currency = ? AND created_at >= ? AND status NOT IN ?",
			userID, chain, currency, startOfMonth, []WithdrawalStatus{
				WithdrawalStatusRejected,
				WithdrawalStatusCancelled,
				WithdrawalStatusFailed,
			}).
		Scan(&total).Error
	return total, err
}

// CreateLimit 创建限额
func (r *repository) CreateLimit(limit *WithdrawalLimit) error {
	return r.db.Create(limit).Error
}

// GetLimit 获取用户限额
func (r *repository) GetLimit(userID uint, chain, currency string) (*WithdrawalLimit, error) {
	var limit WithdrawalLimit
	if err := r.db.Where("user_id = ? AND chain = ? AND currency = ?",
		userID, chain, currency).First(&limit).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &limit, nil
}

// GetGlobalLimit 获取全局限额
func (r *repository) GetGlobalLimit(chain, currency string) (*WithdrawalLimit, error) {
	var limit WithdrawalLimit
	if err := r.db.Where("user_id = 0 AND chain = ? AND currency = ?",
		chain, currency).First(&limit).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &limit, nil
}

// UpdateLimit 更新限额
func (r *repository) UpdateLimit(limit *WithdrawalLimit) error {
	return r.db.Save(limit).Error
}
