package routers

import (
	"strconv"

	"custodial-wallet/internal/deposit"
	"custodial-wallet/internal/withdrawal"
	"custodial-wallet/pkg/httputil"

	"github.com/gin-gonic/gin"
)

// DepositHandler 充值处理器
type DepositHandler struct {
	service deposit.Service
}

// NewDepositHandler 创建充值处理器
func NewDepositHandler(service deposit.Service) *DepositHandler {
	return &DepositHandler{service: service}
}

// Register 注册路由
func (h *DepositHandler) Register(r *gin.RouterGroup) {
	r.GET("/deposits", h.ListDeposits)
	r.GET("/deposits/:id", h.GetDeposit)
	r.GET("/deposit-addresses", h.ListDepositAddresses)
	r.POST("/deposit-addresses", h.AllocateDepositAddress)
}

// ListDeposits 列出充值记录
func (h *DepositHandler) ListDeposits(c *gin.Context) {
	userID := GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	deposits, total, err := h.service.ListDeposits(userID, page, pageSize)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}

	httputil.SuccessWithPage(c, total, page, pageSize, deposits)
}

// GetDeposit 获取充值记录
func (h *DepositHandler) GetDeposit(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	d, err := h.service.GetDeposit(uint(id))
	if err != nil {
		if err == deposit.ErrDepositNotFound {
			httputil.NotFound(c, "deposit not found")
			return
		}
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, d)
}

// ListDepositAddresses 列出充值地址
func (h *DepositHandler) ListDepositAddresses(c *gin.Context) {
	userID := GetUserID(c)
	addresses, err := h.service.ListDepositAddresses(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, addresses)
}

// AllocateDepositAddressRequest 分配充值地址请求
type AllocateDepositAddressRequest struct {
	Chain string `json:"chain" binding:"required"`
}

// AllocateDepositAddress 分配充值地址
func (h *DepositHandler) AllocateDepositAddress(c *gin.Context) {
	userID := GetUserID(c)
	var req AllocateDepositAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	addr, err := h.service.AllocateDepositAddress(userID, req.Chain)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, addr)
}

// WithdrawalHandler 提现处理器
type WithdrawalHandler struct {
	service withdrawal.Service
}

// NewWithdrawalHandler 创建提现处理器
func NewWithdrawalHandler(service withdrawal.Service) *WithdrawalHandler {
	return &WithdrawalHandler{service: service}
}

// Register 注册路由
func (h *WithdrawalHandler) Register(r *gin.RouterGroup) {
	r.POST("/withdrawals", h.CreateWithdrawal)
	r.GET("/withdrawals", h.ListWithdrawals)
	r.GET("/withdrawals/:id", h.GetWithdrawal)
	r.POST("/withdrawals/:id/cancel", h.CancelWithdrawal)
}

// CreateWithdrawalRequest 创建提现请求
type CreateWithdrawalRequest struct {
	Chain           string `json:"chain" binding:"required"`
	ToAddress       string `json:"to_address" binding:"required"`
	Currency        string `json:"currency" binding:"required"`
	Amount          string `json:"amount" binding:"required"`
	ContractAddress string `json:"contract_address"`
	Memo            string `json:"memo"`
}

// CreateWithdrawal 创建提现
func (h *WithdrawalHandler) CreateWithdrawal(c *gin.Context) {
	userID := GetUserID(c)
	var req withdrawal.CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	req.UserID = userID

	w, err := h.service.CreateWithdrawal(&req)
	if err != nil {
		switch err {
		case withdrawal.ErrInsufficientBalance:
			httputil.Error(c, httputil.ErrCodeInsufficientFund, err.Error())
		case withdrawal.ErrExceedDailyLimit, withdrawal.ErrExceedSingleLimit:
			httputil.Error(c, httputil.ErrCodeWithdrawalFailed, err.Error())
		case withdrawal.ErrBelowMinAmount:
			httputil.BadRequest(c, err.Error())
		default:
			httputil.InternalError(c, err.Error())
		}
		return
	}

	httputil.Success(c, w)
}

// ListWithdrawals 列出提现记录
func (h *WithdrawalHandler) ListWithdrawals(c *gin.Context) {
	userID := GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	withdrawals, total, err := h.service.ListWithdrawals(userID, page, pageSize)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}

	httputil.SuccessWithPage(c, total, page, pageSize, withdrawals)
}

// GetWithdrawal 获取提现记录
func (h *WithdrawalHandler) GetWithdrawal(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	w, err := h.service.GetWithdrawal(uint(id))
	if err != nil {
		if err == withdrawal.ErrWithdrawalNotFound {
			httputil.NotFound(c, "withdrawal not found")
			return
		}
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, w)
}

// CancelWithdrawal 取消提现
func (h *WithdrawalHandler) CancelWithdrawal(c *gin.Context) {
	userID := GetUserID(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.service.CancelWithdrawal(uint(id), userID); err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, nil)
}
