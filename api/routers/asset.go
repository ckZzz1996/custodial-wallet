package routers

import (
	"custodial-wallet/internal/asset"
	"custodial-wallet/pkg/httputil"

	"github.com/gin-gonic/gin"
)

// AssetHandler 资产处理器
type AssetHandler struct {
	service asset.Service
}

// NewAssetHandler 创建资产处理器
func NewAssetHandler(service asset.Service) *AssetHandler {
	return &AssetHandler{service: service}
}

// Register 注册路由
func (h *AssetHandler) Register(r *gin.RouterGroup) {
	r.GET("/assets", h.ListAssets)
	r.GET("/assets/user", h.GetUserAssets)
	r.GET("/assets/price/:symbol", h.GetAssetPrice)
	r.GET("/assets/total-value", h.GetTotalValue)
}

// ListAssets 列出资产
func (h *AssetHandler) ListAssets(c *gin.Context) {
	chain := c.Query("chain")
	assets, err := h.service.ListAssets(chain)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, assets)
}

// GetUserAssets 获取用户资产
func (h *AssetHandler) GetUserAssets(c *gin.Context) {
	userID := GetUserID(c)
	assets, err := h.service.GetUserAssets(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, assets)
}

// GetAssetPrice 获取资产价格
func (h *AssetHandler) GetAssetPrice(c *gin.Context) {
	symbol := c.Param("symbol")
	price, err := h.service.GetPrice(symbol)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	if price == nil {
		httputil.NotFound(c, "price not found")
		return
	}
	httputil.Success(c, price)
}

// GetTotalValue 获取用户总资产价值
func (h *AssetHandler) GetTotalValue(c *gin.Context) {
	userID := GetUserID(c)
	totalValue, err := h.service.GetUserTotalValue(userID)
	if err != nil {
		httputil.InternalError(c, err.Error())
		return
	}
	httputil.Success(c, gin.H{
		"total_value_usd": totalValue,
	})
}
