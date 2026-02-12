package audit

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Repository 审计仓储接口
type Repository interface {
	Create(log *AuditLog) error
	GetByID(id uint) (*AuditLog, error)
	List(filter *ListFilter) ([]*AuditLog, int64, error)
	ListByUserID(userID uint, page, pageSize int) ([]*AuditLog, int64, error)
	ListByModule(module string, page, pageSize int) ([]*AuditLog, int64, error)
	CountByAction(module, action string, startTime, endTime time.Time) (int64, error)
}

// ListFilter 列表过滤条件
type ListFilter struct {
	UserID    uint
	AdminID   uint
	Module    string
	Action    string
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建审计仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create 创建审计日志
func (r *repository) Create(log *AuditLog) error {
	return r.db.Create(log).Error
}

// GetByID 获取审计日志
func (r *repository) GetByID(id uint) (*AuditLog, error) {
	var log AuditLog
	if err := r.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// List 列出审计日志
func (r *repository) List(filter *ListFilter) ([]*AuditLog, int64, error) {
	var logs []*AuditLog
	var total int64

	query := r.db.Model(&AuditLog{})

	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.AdminID > 0 {
		query = query.Where("admin_id = ?", filter.AdminID)
	}
	if filter.Module != "" {
		query = query.Where("module = ?", filter.Module)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	query.Count(&total)

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListByUserID 列出用户审计日志
func (r *repository) ListByUserID(userID uint, page, pageSize int) ([]*AuditLog, int64, error) {
	return r.List(&ListFilter{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListByModule 列出模块审计日志
func (r *repository) ListByModule(module string, page, pageSize int) ([]*AuditLog, int64, error) {
	return r.List(&ListFilter{
		Module:   module,
		Page:     page,
		PageSize: pageSize,
	})
}

// CountByAction 统计操作次数
func (r *repository) CountByAction(module, action string, startTime, endTime time.Time) (int64, error) {
	var count int64
	query := r.db.Model(&AuditLog{}).Where("created_at BETWEEN ? AND ?", startTime, endTime)
	if module != "" {
		query = query.Where("module = ?", module)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Service 审计服务接口
type Service interface {
	Log(entry *LogEntry) error
	LogUserAction(userID uint, module, action, resourceID, description string) error
	LogAdminAction(adminID uint, module, action, resourceID, description string, oldValue, newValue interface{}) error
	GetLog(id uint) (*AuditLog, error)
	ListLogs(filter *ListFilter) ([]*AuditLog, int64, error)
	GetUserLogs(userID uint, page, pageSize int) ([]*AuditLog, int64, error)
	ExportLogs(filter *ListFilter) ([]byte, error)
}

type service struct {
	repo Repository
}

// NewService 创建审计服务
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// LogEntry 日志条目
type LogEntry struct {
	UserID      uint
	AdminID     uint
	Module      string
	Action      string
	ResourceID  string
	Description string
	OldValue    interface{}
	NewValue    interface{}
	IP          string
	UserAgent   string
	Status      int
	ErrorMsg    string
}

// Log 记录日志
func (s *service) Log(entry *LogEntry) error {
	var oldValueStr, newValueStr string

	if entry.OldValue != nil {
		if data, err := json.Marshal(entry.OldValue); err == nil {
			oldValueStr = string(data)
		}
	}
	if entry.NewValue != nil {
		if data, err := json.Marshal(entry.NewValue); err == nil {
			newValueStr = string(data)
		}
	}

	log := &AuditLog{
		UserID:      entry.UserID,
		AdminID:     entry.AdminID,
		Module:      entry.Module,
		Action:      entry.Action,
		ResourceID:  entry.ResourceID,
		Description: entry.Description,
		OldValue:    oldValueStr,
		NewValue:    newValueStr,
		IP:          entry.IP,
		UserAgent:   entry.UserAgent,
		Status:      entry.Status,
		ErrorMsg:    entry.ErrorMsg,
	}

	return s.repo.Create(log)
}

// LogUserAction 记录用户操作
func (s *service) LogUserAction(userID uint, module, action, resourceID, description string) error {
	return s.Log(&LogEntry{
		UserID:      userID,
		Module:      module,
		Action:      action,
		ResourceID:  resourceID,
		Description: description,
		Status:      1,
	})
}

// LogAdminAction 记录管理员操作
func (s *service) LogAdminAction(adminID uint, module, action, resourceID, description string, oldValue, newValue interface{}) error {
	return s.Log(&LogEntry{
		AdminID:     adminID,
		Module:      module,
		Action:      action,
		ResourceID:  resourceID,
		Description: description,
		OldValue:    oldValue,
		NewValue:    newValue,
		Status:      1,
	})
}

// GetLog 获取日志
func (s *service) GetLog(id uint) (*AuditLog, error) {
	return s.repo.GetByID(id)
}

// ListLogs 列出日志
func (s *service) ListLogs(filter *ListFilter) ([]*AuditLog, int64, error) {
	return s.repo.List(filter)
}

// GetUserLogs 获取用户日志
func (s *service) GetUserLogs(userID uint, page, pageSize int) ([]*AuditLog, int64, error) {
	return s.repo.ListByUserID(userID, page, pageSize)
}

// ExportLogs 导出日志
func (s *service) ExportLogs(filter *ListFilter) ([]byte, error) {
	logs, _, err := s.repo.List(filter)
	if err != nil {
		return nil, err
	}
	return json.Marshal(logs)
}
