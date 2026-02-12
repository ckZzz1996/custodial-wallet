package routers

import (
	"strconv"

	"custodial-wallet/internal/wallet"
	"custodial-wallet/pkg/httputil"

	"github.com/gin-gonic/gin"
)

// WalletHandler 钱包处理器
type WalletHandler struct {
	service wallet.Service
}

// NewWalletHandler 创建钱包处理器
func NewWalletHandler(service wallet.Service) *WalletHandler {
	return &WalletHandler{service: service}
}

// Register 注册路由
func (h *WalletHandler) Register(r *gin.RouterGroup) {
	r.POST("/wallets", h.CreateWallet)
	r.GET("/wallets", h.ListWallets)
	r.GET("/wallets/:id", h.GetWallet)
	r.PUT("/wallets/:id", h.UpdateWallet)
	r.DELETE("/wallets/:id", h.DeleteWallet)

	r.POST("/wallets/:id/addresses", h.GenerateAddress)
	r.GET("/wallets/:id/addresses", h.ListAddresses)

	r.GET("/deposit-address", h.GetDepositAddress)
	r.GET("/balances", h.ListBalances)
	r.GET("/balances/:chain/:currency", h.GetBalance)

	r.POST("/address-book", h.AddToAddressBook)
	r.GET("/address-book", h.ListAddressBook)
	r.DELETE("/address-book/:id", h.RemoveFromAddressBook)
}

// CreateWalletRequest 创建钱包请求
type CreateWalletRequest struct {
	Name string            `json:"name" binding:"required"`
	Type wallet.WalletType `json:"type"`
}

// CreateWallet 创建钱包
func (h *WalletHandler) CreateWallet(c *gin.Context) {
	userID := GetUserID(c)
	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	if req.Type == 0 {
		req.Type = wallet.WalletTypeHot
	}

	w, err := h.service.CreateWallet(userID, req.Name, req.Type)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}

	httputil.Success(c, w)
}

// ListWallets 列出钱包
func (h *WalletHandler) ListWallets(c *gin.Context) {
	userID := GetUserID(c)
	wallets, err := h.service.ListWallets(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, wallets)
}

// GetWallet 获取钱包
func (h *WalletHandler) GetWallet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	w, err := h.service.GetWallet(uint(id))
	if err != nil {
		if err == wallet.ErrWalletNotFound {
			httputil.NotFound(c, "wallet not found")
			return
		}
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, w)
}

// UpdateWalletRequest 更新钱包请求
type UpdateWalletRequest struct {
	Name string `json:"name" binding:"required"`
}

// UpdateWallet 更新钱包
func (h *WalletHandler) UpdateWallet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req UpdateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	w, err := h.service.UpdateWallet(uint(id), req.Name)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, w)
}

// DeleteWallet 删除钱包
func (h *WalletHandler) DeleteWallet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.service.DeleteWallet(uint(id)); err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, nil)
}

// GenerateAddressRequest 生成地址请求
type GenerateAddressRequest struct {
	Chain string `json:"chain" binding:"required"`
	Label string `json:"label"`
}

// GenerateAddress 生成地址
func (h *WalletHandler) GenerateAddress(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req GenerateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	addr, err := h.service.GenerateAddress(uint(id), wallet.Chain(req.Chain), req.Label)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, addr)
}

// ListAddresses 列出地址
func (h *WalletHandler) ListAddresses(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	addresses, err := h.service.ListAddresses(uint(id))
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, addresses)
}

// GetDepositAddress 获取充值地址
func (h *WalletHandler) GetDepositAddress(c *gin.Context) {
	userID := GetUserID(c)
	chain := c.Query("chain")
	if chain == "" {
		httputil.BadRequest(c, "chain is required")
		return
	}

	addr, err := h.service.GetDepositAddress(userID, wallet.Chain(chain))
	if err != nil {
		if err == wallet.ErrAddressNotFound {
			httputil.NotFound(c, "no deposit address found, please generate one first")
			return
		}
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, addr)
}

// ListBalances 列出余额
func (h *WalletHandler) ListBalances(c *gin.Context) {
	userID := GetUserID(c)
	balances, err := h.service.ListBalances(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, balances)
}

// GetBalance 获取余额
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID := GetUserID(c)
	chain := c.Param("chain")
	currency := c.Param("currency")

	balance, err := h.service.GetBalance(userID, wallet.Chain(chain), currency)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, balance)
}

// AddToAddressBookRequest 添加到地址簿请求
type AddToAddressBookRequest struct {
	Chain       string `json:"chain" binding:"required"`
	Address     string `json:"address" binding:"required"`
	Label       string `json:"label"`
	IsWhitelist bool   `json:"is_whitelist"`
}

// AddToAddressBook 添加到地址簿
func (h *WalletHandler) AddToAddressBook(c *gin.Context) {
	userID := GetUserID(c)
	var req AddToAddressBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}

	entry, err := h.service.AddToAddressBook(userID, wallet.Chain(req.Chain), req.Address, req.Label, req.IsWhitelist)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, entry)
}

// ListAddressBook 列出地址簿
func (h *WalletHandler) ListAddressBook(c *gin.Context) {
	userID := GetUserID(c)
	entries, err := h.service.ListAddressBook(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, entries)
}

// RemoveFromAddressBook 从地址簿删除
func (h *WalletHandler) RemoveFromAddressBook(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.service.RemoveFromAddressBook(uint(id)); err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, nil)
}
