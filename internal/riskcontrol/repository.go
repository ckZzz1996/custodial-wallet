package riskcontrol

import (
	"errors"

	"gorm.io/gorm"
)

// Repository 风控仓储接口
type Repository interface {
	// Rules
	CreateRule(rule *RiskRule) error
	GetRuleByID(id uint) (*RiskRule, error)
	ListRules(ruleType RuleType, status int) ([]*RiskRule, error)
	ListActiveRules() ([]*RiskRule, error)
	UpdateRule(rule *RiskRule) error
	DeleteRule(id uint) error

	// Blacklist
	CreateBlacklist(bl *Blacklist) error
	GetBlacklistByID(id uint) (*Blacklist, error)
	CheckBlacklist(blType, value, chain string) (bool, error)
	ListBlacklist(blType string, page, pageSize int) ([]*Blacklist, int64, error)
	UpdateBlacklist(bl *Blacklist) error
	DeleteBlacklist(id uint) error

	// Risk Log
	CreateRiskLog(log *RiskLog) error
	ListRiskLogsByUserID(userID uint, limit int) ([]*RiskLog, error)
	ListRiskLogs(page, pageSize int) ([]*RiskLog, int64, error)

	// User Risk Profile
	CreateUserRiskProfile(profile *UserRiskProfile) error
	GetUserRiskProfile(userID uint) (*UserRiskProfile, error)
	UpdateUserRiskProfile(profile *UserRiskProfile) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建风控仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateRule 创建规则
func (r *repository) CreateRule(rule *RiskRule) error {
	return r.db.Create(rule).Error
}

// GetRuleByID 获取规则
func (r *repository) GetRuleByID(id uint) (*RiskRule, error) {
	var rule RiskRule
	if err := r.db.First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// ListRules 列出规则
func (r *repository) ListRules(ruleType RuleType, status int) ([]*RiskRule, error) {
	var rules []*RiskRule
	query := r.db.Model(&RiskRule{})
	if ruleType != "" {
		query = query.Where("type = ?", ruleType)
	}
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("priority DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ListActiveRules 列出活跃规则
func (r *repository) ListActiveRules() ([]*RiskRule, error) {
	var rules []*RiskRule
	if err := r.db.Where("status = ?", 1).Order("priority DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// UpdateRule 更新规则
func (r *repository) UpdateRule(rule *RiskRule) error {
	return r.db.Save(rule).Error
}

// DeleteRule 删除规则
func (r *repository) DeleteRule(id uint) error {
	return r.db.Delete(&RiskRule{}, id).Error
}

// CreateBlacklist 创建黑名单
func (r *repository) CreateBlacklist(bl *Blacklist) error {
	return r.db.Create(bl).Error
}

// GetBlacklistByID 获取黑名单
func (r *repository) GetBlacklistByID(id uint) (*Blacklist, error) {
	var bl Blacklist
	if err := r.db.First(&bl, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bl, nil
}

// CheckBlacklist 检查是否在黑名单
func (r *repository) CheckBlacklist(blType, value, chain string) (bool, error) {
	var count int64
	query := r.db.Model(&Blacklist{}).Where("type = ? AND value = ? AND status = 1", blType, value)
	if chain != "" {
		query = query.Where("(chain = ? OR chain = '')", chain)
	}
	query = query.Where("(expires_at IS NULL OR expires_at > NOW())")
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListBlacklist 列出黑名单
func (r *repository) ListBlacklist(blType string, page, pageSize int) ([]*Blacklist, int64, error) {
	var items []*Blacklist
	var total int64

	query := r.db.Model(&Blacklist{})
	if blType != "" {
		query = query.Where("type = ?", blType)
	}
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// UpdateBlacklist 更新黑名单
func (r *repository) UpdateBlacklist(bl *Blacklist) error {
	return r.db.Save(bl).Error
}

// DeleteBlacklist 删除黑名单
func (r *repository) DeleteBlacklist(id uint) error {
	return r.db.Delete(&Blacklist{}, id).Error
}

// CreateRiskLog 创建风控日志
func (r *repository) CreateRiskLog(log *RiskLog) error {
	return r.db.Create(log).Error
}

// ListRiskLogsByUserID 列出用户风控日志
func (r *repository) ListRiskLogsByUserID(userID uint, limit int) ([]*RiskLog, error) {
	var logs []*RiskLog
	if err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// ListRiskLogs 列出风控日志
func (r *repository) ListRiskLogs(page, pageSize int) ([]*RiskLog, int64, error) {
	var logs []*RiskLog
	var total int64

	r.db.Model(&RiskLog{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := r.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// CreateUserRiskProfile 创建用户风险画像
func (r *repository) CreateUserRiskProfile(profile *UserRiskProfile) error {
	return r.db.Create(profile).Error
}

// GetUserRiskProfile 获取用户风险画像
func (r *repository) GetUserRiskProfile(userID uint) (*UserRiskProfile, error) {
	var profile UserRiskProfile
	if err := r.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// UpdateUserRiskProfile 更新用户风险画像
func (r *repository) UpdateUserRiskProfile(profile *UserRiskProfile) error {
	return r.db.Save(profile).Error
}
