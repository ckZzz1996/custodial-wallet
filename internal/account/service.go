package account

import (
	"encoding/json"
	"errors"
	"time"

	"custodial-wallet/pkg/crypto"
	"custodial-wallet/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserInactive    = errors.New("user is inactive")
	ErrInvalidToken    = errors.New("invalid token")
)

// Service 账户服务接口
type Service interface {
	Register(req *RegisterRequest) (*User, error)
	Login(req *LoginRequest, ip, userAgent string) (*LoginResponse, error)
	GetUser(userID uint) (*User, error)
	GetUserByUUID(uuid string) (*User, error)
	UpdateUser(userID uint, req *UpdateUserRequest) (*User, error)
	ChangePassword(userID uint, oldPassword, newPassword string) error
	UpdateKYCStatus(userID uint, status KYCStatus, level int) error
	Enable2FA(userID uint) (string, error)
	Verify2FA(userID uint, code string) bool
	GenerateAPIKey(userID uint, name string, permissions []string) (*APIKey, string, error)
	ValidateAPIKey(key, secret string) (*User, error)
	ListLoginHistory(userID uint, limit int) ([]*LoginHistory, error)
	ListAPIKeys(userID uint) ([]*APIKey, error)
}

type service struct {
	repo      Repository
	jwtSecret []byte
	jwtExpiry time.Duration
}

// NewService 创建账户服务
func NewService(repo Repository, jwtSecret string, jwtExpiry time.Duration) Service {
	return &service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
	TwoFACode string `json:"two_fa_code"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      *User  `json:"user"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Phone string `json:"phone"`
}

// Register 用户注册
func (s *service) Register(req *RegisterRequest) (*User, error) {
	// 检查邮箱是否已存在
	existingUser, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 密码加密
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &User{
		UUID:         uuid.New().String(),
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: passwordHash,
		Status:       UserStatusActive,
		KYCStatus:    KYCStatusNone,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	logger.Infof("User registered: %s", user.Email)
	return user, nil
}

// Login 用户登录
func (s *service) Login(req *LoginRequest, ip, userAgent string) (*LoginResponse, error) {
	// 获取用户
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 检查用户状态
	if user.Status != UserStatusActive {
		s.recordLoginHistory(user.ID, ip, userAgent, 0)
		return nil, ErrUserInactive
	}

	// 验证密码
	if !crypto.CheckPassword(req.Password, user.PasswordHash) {
		s.recordLoginHistory(user.ID, ip, userAgent, 0)
		return nil, ErrInvalidPassword
	}

	// 验证2FA（如果启用）
	if user.TwoFAEnabled {
		if req.TwoFACode == "" || !s.Verify2FA(user.ID, req.TwoFACode) {
			s.recordLoginHistory(user.ID, ip, userAgent, 0)
			return nil, errors.New("invalid 2FA code")
		}
	}

	// 生成JWT
	expiresAt := time.Now().Add(s.jwtExpiry)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"uuid":    user.UUID,
		"email":   user.Email,
		"exp":     expiresAt.Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	// 更新最后登录信息
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = ip
	_ = s.repo.UpdateUser(user)

	// 记录登录历史
	s.recordLoginHistory(user.ID, ip, userAgent, 1)

	logger.Infof("User logged in: %s from %s", user.Email, ip)
	return &LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
		User:      user,
	}, nil
}

func (s *service) recordLoginHistory(userID uint, ip, userAgent string, status int) {
	history := &LoginHistory{
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		Status:    status,
	}
	_ = s.repo.CreateLoginHistory(history)
}

// GetUser 获取用户
func (s *service) GetUser(userID uint) (*User, error) {
	return s.repo.GetUserByID(userID)
}

// GetUserByUUID 通过UUID获取用户
func (s *service) GetUserByUUID(uuid string) (*User, error) {
	return s.repo.GetUserByUUID(uuid)
}

// UpdateUser 更新用户
func (s *service) UpdateUser(userID uint, req *UpdateUserRequest) (*User, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if req.Phone != "" {
		user.Phone = req.Phone
	}

	if err := s.repo.UpdateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword 修改密码
func (s *service) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if !crypto.CheckPassword(oldPassword, user.PasswordHash) {
		return ErrInvalidPassword
	}

	newHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = newHash
	return s.repo.UpdateUser(user)
}

// UpdateKYCStatus 更新KYC状态
func (s *service) UpdateKYCStatus(userID uint, status KYCStatus, level int) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.KYCStatus = status
	user.KYCLevel = level
	return s.repo.UpdateUser(user)
}

// Enable2FA 启用两步验证
func (s *service) Enable2FA(userID uint) (string, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}

	// 使用 TOTP 生成密钥
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "CustodialWallet", AccountName: user.Email})
	if err != nil {
		return "", err
	}

	user.TwoFASecret = key.Secret()
	user.TwoFAEnabled = true
	if err := s.repo.UpdateUser(user); err != nil {
		return "", err
	}

	return key.Secret(), nil
}

// Verify2FA 验证两步验证码
func (s *service) Verify2FA(userID uint, code string) bool {
	user, err := s.repo.GetUserByID(userID)
	if err != nil || user == nil || user.TwoFASecret == "" {
		return false
	}
	valid := totp.Validate(code, user.TwoFASecret)
	return valid
}

// GenerateAPIKey 生成API密钥
func (s *service) GenerateAPIKey(userID uint, name string, permissions []string) (*APIKey, string, error) {
	key, err := crypto.GenerateRandomHex(16)
	if err != nil {
		return nil, "", err
	}

	secretRaw, err := crypto.GenerateRandomHex(32)
	if err != nil {
		return nil, "", err
	}

	secretHash, err := crypto.HashPassword(secretRaw)
	if err != nil {
		return nil, "", err
	}

	permJSON, _ := json.Marshal(permissions)

	apiKey := &APIKey{
		UserID:      userID,
		Name:        name,
		Key:         key,
		Secret:      secretHash,
		Permissions: string(permJSON),
		Status:      1,
	}

	if err := s.repo.CreateAPIKey(apiKey); err != nil {
		return nil, "", err
	}

	return apiKey, secretRaw, nil
}

// ValidateAPIKey 验证API密钥
func (s *service) ValidateAPIKey(key, secret string) (*User, error) {
	apiKey, err := s.repo.GetAPIKeyByKey(key)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, errors.New("api key not found")
	}

	if !crypto.CheckPassword(secret, apiKey.Secret) {
		return nil, errors.New("invalid api secret")
	}

	return s.repo.GetUserByID(apiKey.UserID)
}

// ListLoginHistory 获取登录历史
func (s *service) ListLoginHistory(userID uint, limit int) ([]*LoginHistory, error) {
	return s.repo.ListLoginHistoriesByUserID(userID, limit)
}

// ListAPIKeys 列出用户API密钥
func (s *service) ListAPIKeys(userID uint) ([]*APIKey, error) {
	return s.repo.ListAPIKeysByUserID(userID)
}
