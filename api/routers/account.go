package routers

import (
	"strconv"

	"custodial-wallet/internal/account"
	"custodial-wallet/pkg/httputil"

	"github.com/gin-gonic/gin"
)

// AccountHandler 账户处理器
type AccountHandler struct {
	service account.Service
}

// NewAccountHandler 创建账户处理器
func NewAccountHandler(service account.Service) *AccountHandler {
	return &AccountHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *AccountHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)

	auth := r.Group("")
	auth.Use(AuthMiddleware())
	{
		auth.GET("/profile", h.GetProfile)
		auth.PUT("/profile", h.UpdateProfile)
		auth.PUT("/password", h.ChangePassword)
		auth.POST("/2fa/enable", h.Enable2FA)
		auth.POST("/2fa/verify", h.Verify2FA)
		auth.GET("/login-history", h.GetLoginHistory)
		auth.POST("/api-keys", h.CreateAPIKey)
		auth.GET("/api-keys", h.ListAPIKeys)
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone"`
}

// Register 用户注册
// @Summary 用户注册
// @Tags Account
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册信息"
// @Success 200 {object} httputil.Response
// @Router /api/v1/register [post]
func (h *AccountHandler) Register(c *gin.Context) {
	var req account.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.Register(&req)
	if err != nil {
		if err == account.ErrUserExists {
			httputil.Error(c, httputil.ErrCodeUserExists, err.Error())
			return
		}
		httputil.InternalError(c, err.Error())
		return
	}

	httputil.Success(c, user)
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
	TwoFACode string `json:"two_fa_code"`
}

// Login 用户登录
// @Summary 用户登录
// @Tags Account
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录信息"
// @Success 200 {object} httputil.Response
// @Router /api/v1/login [post]
func (h *AccountHandler) Login(c *gin.Context) {
	var req account.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	resp, err := h.service.Login(&req, ip, userAgent)
	if err != nil {
		if err == account.ErrUserNotFound || err == account.ErrInvalidPassword {
			httputil.Error(c, httputil.ErrCodeInvalidPassword, "invalid email or password")
			return
		}
		httputil.InternalError(c, err.Error())
		return
	}

	httputil.Success(c, resp)
}

// GetProfile 获取用户资料
func (h *AccountHandler) GetProfile(c *gin.Context) {
	userID := GetUserID(c)
	user, err := h.service.GetUser(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, user)
}

// UpdateProfile 更新用户资料
func (h *AccountHandler) UpdateProfile(c *gin.Context) {
	userID := GetUserID(c)
	var req account.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.UpdateUser(userID, &req)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, user)
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword 修改密码
func (h *AccountHandler) ChangePassword(c *gin.Context) {
	userID := GetUserID(c)
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	if err := h.service.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		if err == account.ErrInvalidPassword {
			httputil.Error(c, httputil.ErrCodeInvalidPassword, "invalid old password")
			return
		}
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, nil)
}

// Enable2FA 启用两步验证
func (h *AccountHandler) Enable2FA(c *gin.Context) {
	userID := GetUserID(c)
	secret, err := h.service.Enable2FA(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, gin.H{"secret": secret})
}

// Verify2FARequest 验证2FA请求
type Verify2FARequest struct {
	Code string `json:"code" binding:"required"`
}

// Verify2FA 验证两步验证
func (h *AccountHandler) Verify2FA(c *gin.Context) {
	userID := GetUserID(c)
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	if !h.service.Verify2FA(userID, req.Code) {
		httputil.Error(c, 400, "invalid 2FA code")
		return
	}
	httputil.Success(c, nil)
}

// GetLoginHistory 获取登录历史
func (h *AccountHandler) GetLoginHistory(c *gin.Context) {
	userID := GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	history, err := h.service.ListLoginHistory(userID, limit)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, history)
}

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
}

// CreateAPIKey 创建API密钥
func (h *AccountHandler) CreateAPIKey(c *gin.Context) {
	userID := GetUserID(c)
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	apiKey, secret, err := h.service.GenerateAPIKey(userID, req.Name, req.Permissions)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, gin.H{
		"api_key": apiKey,
		"secret":  secret,
	})
}

// ListAPIKeys 列出API密钥
func (h *AccountHandler) ListAPIKeys(c *gin.Context) {
	userID := GetUserID(c)
	apiKeys, err := h.service.ListAPIKeys(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, apiKeys)
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) uint {
	userID, _ := c.Get("user_id")
	return userID.(uint)
}
