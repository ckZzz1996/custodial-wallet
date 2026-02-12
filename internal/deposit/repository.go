package deposit

import (
	"errors"

	"gorm.io/gorm"
)

// Repository 充值仓储接口
type Repository interface {
	CreateDeposit(deposit *Deposit) error
	GetDepositByID(id uint) (*Deposit, error)
	GetDepositByTxHash(txHash string) (*Deposit, error)
	ListDepositsByUserID(userID uint, page, pageSize int) ([]*Deposit, int64, error)
	ListPendingDeposits(chain string, limit int) ([]*Deposit, error)
	ListUnconfirmedDeposits(chain string, limit int) ([]*Deposit, error)
	UpdateDeposit(deposit *Deposit) error
	UpdateDepositStatus(id uint, status DepositStatus) error
	CreditDeposit(id uint) error

	CreateDepositAddress(addr *DepositAddress) error
	GetDepositAddress(chain, address string) (*DepositAddress, error)
	GetUserDepositAddress(userID uint, chain string) (*DepositAddress, error)
	ListDepositAddresses(userID uint) ([]*DepositAddress, error)

	// 以下为扫描相关
	ListAllDepositAddresses(chain string) ([]*DepositAddress, error)
	GetLastScannedBlock(chain string) (uint64, error)
	SetLastScannedBlock(chain string, block uint64) error

	CreateSweepTask(task *SweepTask) error
	GetSweepTask(id uint) (*SweepTask, error)
	ListPendingSweepTasks(chain string, limit int) ([]*SweepTask, error)
	UpdateSweepTask(task *SweepTask) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建充值仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateDeposit 创建充值记录
func (r *repository) CreateDeposit(deposit *Deposit) error {
	return r.db.Create(deposit).Error
}

// GetDepositByID 通过ID获取充值
func (r *repository) GetDepositByID(id uint) (*Deposit, error) {
	var deposit Deposit
	if err := r.db.First(&deposit, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &deposit, nil
}

// GetDepositByTxHash 通过交易哈希获取充值
func (r *repository) GetDepositByTxHash(txHash string) (*Deposit, error) {
	var deposit Deposit
	if err := r.db.Where("tx_hash = ?", txHash).First(&deposit).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &deposit, nil
}

// ListDepositsByUserID 列出用户充值记录
func (r *repository) ListDepositsByUserID(userID uint, page, pageSize int) ([]*Deposit, int64, error) {
	var deposits []*Deposit
	var total int64

	r.db.Model(&Deposit{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * pageSize
	if err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&deposits).Error; err != nil {
		return nil, 0, err
	}

	return deposits, total, nil
}

// ListPendingDeposits 列出待处理充值
func (r *repository) ListPendingDeposits(chain string, limit int) ([]*Deposit, error) {
	var deposits []*Deposit
	query := r.db.Where("status IN ?", []DepositStatus{DepositStatusPending, DepositStatusConfirming})
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if err := query.Order("created_at ASC").Limit(limit).Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// ListUnconfirmedDeposits 列出未确认的充值
func (r *repository) ListUnconfirmedDeposits(chain string, limit int) ([]*Deposit, error) {
	var deposits []*Deposit
	query := r.db.Where("status = ? AND credited = ?", DepositStatusConfirmed, false)
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if err := query.Order("created_at ASC").Limit(limit).Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// UpdateDeposit 更新充值记录
func (r *repository) UpdateDeposit(deposit *Deposit) error {
	return r.db.Save(deposit).Error
}

// UpdateDepositStatus 更新充值状态
func (r *repository) UpdateDepositStatus(id uint, status DepositStatus) error {
	return r.db.Model(&Deposit{}).Where("id = ?", id).Update("status", status).Error
}

// CreditDeposit 入账充值
func (r *repository) CreditDeposit(id uint) error {
	return r.db.Model(&Deposit{}).Where("id = ?", id).Updates(map[string]interface{}{
		"credited":    true,
		"credited_at": gorm.Expr("NOW()"),
		"status":      DepositStatusCredited,
	}).Error
}

// CreateDepositAddress 创建充值地址
func (r *repository) CreateDepositAddress(addr *DepositAddress) error {
	return r.db.Create(addr).Error
}

// GetDepositAddress 获取充值地址
func (r *repository) GetDepositAddress(chain, address string) (*DepositAddress, error) {
	var addr DepositAddress
	if err := r.db.Where("chain = ? AND address = ?", chain, address).First(&addr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addr, nil
}

// GetUserDepositAddress 获取用户充值地址
func (r *repository) GetUserDepositAddress(userID uint, chain string) (*DepositAddress, error) {
	var addr DepositAddress
	if err := r.db.Where("user_id = ? AND chain = ?", userID, chain).First(&addr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addr, nil
}

// ListDepositAddresses 列出用户充值地址
func (r *repository) ListDepositAddresses(userID uint) ([]*DepositAddress, error) {
	var addrs []*DepositAddress
	if err := r.db.Where("user_id = ?", userID).Find(&addrs).Error; err != nil {
		return nil, err
	}
	return addrs, nil
}

// ListAllDepositAddresses 列出某链上所有充值地址
func (r *repository) ListAllDepositAddresses(chain string) ([]*DepositAddress, error) {
	var addrs []*DepositAddress
	if err := r.db.Where("chain = ?", chain).Find(&addrs).Error; err != nil {
		return nil, err
	}
	return addrs, nil
}

// GetLastScannedBlock 获取最后已扫描的区块号
func (r *repository) GetLastScannedBlock(chain string) (uint64, error) {
	var s ScanProgress
	if err := r.db.Where("chain = ?", chain).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return s.LastScanned, nil
}

// SetLastScannedBlock 设置最后已扫描的区块号
func (r *repository) SetLastScannedBlock(chain string, block uint64) error {
	var s ScanProgress
	if err := r.db.Where("chain = ?", chain).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s = ScanProgress{Chain: chain, LastScanned: block}
			return r.db.Create(&s).Error
		}
		return err
	}
	return r.db.Model(&s).Update("last_scanned", block).Error
}

// CreateSweepTask 创建归集任务
func (r *repository) CreateSweepTask(task *SweepTask) error {
	return r.db.Create(task).Error
}

// GetSweepTask 获取归集任务
func (r *repository) GetSweepTask(id uint) (*SweepTask, error) {
	var task SweepTask
	if err := r.db.First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// ListPendingSweepTasks 列出待处理归集任务
func (r *repository) ListPendingSweepTasks(chain string, limit int) ([]*SweepTask, error) {
	var tasks []*SweepTask
	query := r.db.Where("status = ?", 0)
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if err := query.Order("created_at ASC").Limit(limit).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateSweepTask 更新归集任务
func (r *repository) UpdateSweepTask(task *SweepTask) error {
	return r.db.Save(task).Error
}
