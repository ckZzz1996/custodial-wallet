package routers

import (
	"net/http"
	"time"

	"custodial-wallet/internal/account"
	"custodial-wallet/internal/asset"
	"custodial-wallet/internal/deposit"
	"custodial-wallet/internal/wallet"
	"custodial-wallet/internal/withdrawal"

	"github.com/gin-gonic/gin"
)

// Services 服务集合
type Services struct {
	Account    account.Service
	Wallet     wallet.Service
	Deposit    deposit.Service
	Withdrawal withdrawal.Service
	Asset      asset.Service
}

// SetupRouter 设置路由
func SetupRouter(svc *Services) *gin.Engine {
	router := gin.New()
	router.Use(LoggerMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(CORSMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API v1
	apiV1 := router.Group("/api/v1")
	{
		// Public routes
		accountHandler := NewAccountHandler(svc.Account)
		apiV1.POST("/register", accountHandler.Register)
		apiV1.POST("/login", accountHandler.Login)

		// Protected routes
		protected := apiV1.Group("")
		protected.Use(AuthMiddleware())
		{
			// Account
			protected.GET("/profile", accountHandler.GetProfile)
			protected.PUT("/profile", accountHandler.UpdateProfile)
			protected.PUT("/password", accountHandler.ChangePassword)
			protected.POST("/2fa/enable", accountHandler.Enable2FA)
			protected.GET("/login-history", accountHandler.GetLoginHistory)
			protected.POST("/api-keys", accountHandler.CreateAPIKey)
			protected.GET("/api-keys", accountHandler.ListAPIKeys)

			// Wallet
			walletHandler := NewWalletHandler(svc.Wallet)
			walletHandler.Register(protected)

			// Deposit
			depositHandler := NewDepositHandler(svc.Deposit)
			depositHandler.Register(protected)

			// Withdrawal
			withdrawalHandler := NewWithdrawalHandler(svc.Withdrawal)
			withdrawalHandler.Register(protected)

			// Asset
			assetHandler := NewAssetHandler(svc.Asset)
			assetHandler.Register(protected)
		}
	}

	return router
}
