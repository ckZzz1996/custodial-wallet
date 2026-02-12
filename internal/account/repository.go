package account

import (
	"errors"

	"gorm.io/gorm"
)

// Repository 账户仓储接口
type Repository interface {
	CreateUser(user *User) error
	GetUserByID(id uint) (*User, error)
	GetUserByUUID(uuid string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id uint) error
	ListUsers(page, pageSize int) ([]*User, int64, error)

	CreateProfile(profile *UserProfile) error
	GetProfileByUserID(userID uint) (*UserProfile, error)
	UpdateProfile(profile *UserProfile) error

	CreateAPIKey(apiKey *APIKey) error
	GetAPIKeyByKey(key string) (*APIKey, error)
	ListAPIKeysByUserID(userID uint) ([]*APIKey, error)
	UpdateAPIKey(apiKey *APIKey) error
	DeleteAPIKey(id uint) error

	CreateLoginHistory(history *LoginHistory) error
	ListLoginHistoriesByUserID(userID uint, limit int) ([]*LoginHistory, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建账户仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateUser 创建用户
func (r *repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

// GetUserByID 通过ID获取用户
func (r *repository) GetUserByID(id uint) (*User, error) {
	var user User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUUID 通过UUID获取用户
func (r *repository) GetUserByUUID(uuid string) (*User, error) {
	var user User
	if err := r.db.Where("uuid = ?", uuid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 通过邮箱获取用户
func (r *repository) GetUserByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户
func (r *repository) UpdateUser(user *User) error {
	return r.db.Save(user).Error
}

// DeleteUser 删除用户（软删除）
func (r *repository) DeleteUser(id uint) error {
	return r.db.Delete(&User{}, id).Error
}

// ListUsers 列出用户
func (r *repository) ListUsers(page, pageSize int) ([]*User, int64, error) {
	var users []*User
	var total int64

	r.db.Model(&User{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := r.db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// CreateProfile 创建用户资料
func (r *repository) CreateProfile(profile *UserProfile) error {
	return r.db.Create(profile).Error
}

// GetProfileByUserID 通过用户ID获取资料
func (r *repository) GetProfileByUserID(userID uint) (*UserProfile, error) {
	var profile UserProfile
	if err := r.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// UpdateProfile 更新用户资料
func (r *repository) UpdateProfile(profile *UserProfile) error {
	return r.db.Save(profile).Error
}

// CreateAPIKey 创建API密钥
func (r *repository) CreateAPIKey(apiKey *APIKey) error {
	return r.db.Create(apiKey).Error
}

// GetAPIKeyByKey 通过Key获取API密钥
func (r *repository) GetAPIKeyByKey(key string) (*APIKey, error) {
	var apiKey APIKey
	if err := r.db.Where("key = ?", key).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

// ListAPIKeysByUserID 列出用户的API密钥
func (r *repository) ListAPIKeysByUserID(userID uint) ([]*APIKey, error) {
	var apiKeys []*APIKey
	if err := r.db.Where("user_id = ?", userID).Find(&apiKeys).Error; err != nil {
		return nil, err
	}
	return apiKeys, nil
}

// UpdateAPIKey 更新API密钥
func (r *repository) UpdateAPIKey(apiKey *APIKey) error {
	return r.db.Save(apiKey).Error
}

// DeleteAPIKey 删除API密钥
func (r *repository) DeleteAPIKey(id uint) error {
	return r.db.Delete(&APIKey{}, id).Error
}

// CreateLoginHistory 创建登录历史
func (r *repository) CreateLoginHistory(history *LoginHistory) error {
	return r.db.Create(history).Error
}

// ListLoginHistoriesByUserID 列出用户登录历史
func (r *repository) ListLoginHistoriesByUserID(userID uint, limit int) ([]*LoginHistory, error) {
	var histories []*LoginHistory
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}
