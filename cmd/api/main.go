package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "custodial-wallet/api/grpc"
	"custodial-wallet/api/routers"
	"custodial-wallet/internal/account"
	"custodial-wallet/internal/asset"
	"custodial-wallet/internal/audit"
	"custodial-wallet/internal/blockchain"
	"custodial-wallet/internal/blockchain/bitcoin"
	"custodial-wallet/internal/blockchain/ethereum"
	"custodial-wallet/internal/blockchain/tron"
	"custodial-wallet/internal/deposit"
	"custodial-wallet/internal/keymanager"
	"custodial-wallet/internal/notification"
	"custodial-wallet/internal/riskcontrol"
	"custodial-wallet/internal/transaction"
	"custodial-wallet/internal/wallet"
	"custodial-wallet/internal/withdrawal"
	"custodial-wallet/pkg/cache"
	"custodial-wallet/pkg/config"
	"custodial-wallet/pkg/database"
	"custodial-wallet/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志
	logger.Init(cfg.App.Env)
	defer logger.Sync()

	logger.Infof("Starting %s v%s", cfg.App.Name, cfg.App.Version)

	// 初始化数据库
	if err := database.Init(cfg.Database); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 自动迁移
	if err := autoMigrate(); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化Redis
	if err := cache.Init(cfg.Redis); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer cache.Close()

	// 初始化区块链客户端
	blockchains := initBlockchains(cfg)

	// 初始化服务
	services := initServices(cfg, blockchains)

	// 设置JWT密钥
	routers.SetJWTSecret(cfg.JWT.Secret)
	grpcserver.SetJWTSecret(cfg.JWT.Secret)

	// 初始化Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// HTTP服务器 (Gin)
	httpRouter := routers.SetupRouter(&routers.Services{
		Account:    services.account,
		Wallet:     services.wallet,
		Deposit:    services.deposit,
		Withdrawal: services.withdrawal,
		Asset:      services.asset,
	})
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      httpRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// gRPC服务器
	grpcPort := fmt.Sprintf("%d", cfg.App.Port+1) // gRPC端口 = HTTP端口 + 1
	grpcSrv, err := grpcserver.NewServer(
		&grpcserver.ServerConfig{Port: grpcPort},
		&grpcserver.Services{
			Account:    services.account,
			Wallet:     services.wallet,
			Deposit:    services.deposit,
			Withdrawal: services.withdrawal,
			Asset:      services.asset,
		},
	)
	if err != nil {
		logger.Fatalf("Failed to create gRPC server: %v", err)
	}

	// 启动HTTP服务器
	go func() {
		logger.Infof("HTTP server (Gin) listening on port %d", cfg.App.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 启动gRPC服务器
	go func() {
		logger.Infof("gRPC server listening on port %s", grpcPort)
		if err := grpcSrv.Start(); err != nil {
			logger.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Errorf("HTTP server forced to shutdown: %v", err)
	}

	// 关闭gRPC服务器
	grpcSrv.Stop()

	logger.Info("Servers exited")
}

func autoMigrate() error {
	db := database.GetDB()
	return db.AutoMigrate(
		// Account
		&account.User{},
		&account.UserProfile{},
		&account.APIKey{},
		&account.LoginHistory{},
		// Wallet
		&wallet.Wallet{},
		&wallet.Address{},
		&wallet.Balance{},
		&wallet.AddressBook{},
		// KeyManager
		&keymanager.EncryptedKey{},
		&keymanager.SignatureRequest{},
		// Transaction
		&transaction.Transaction{},
		// Deposit
		&deposit.Deposit{},
		&deposit.DepositAddress{},
		&deposit.SweepTask{},
		&deposit.ScanProgress{},
		// Withdrawal
		&withdrawal.Withdrawal{},
		&withdrawal.WithdrawalLimit{},
		// Asset
		&asset.Asset{},
		&asset.AssetPrice{},
		&asset.UserAsset{},
		// RiskControl
		&riskcontrol.RiskRule{},
		&riskcontrol.Blacklist{},
		&riskcontrol.RiskLog{},
		&riskcontrol.UserRiskProfile{},
		// Audit
		&audit.AuditLog{},
		// Notification
		&notification.Notification{},
		&notification.NotificationTemplate{},
		&notification.UserNotificationSetting{},
		&notification.WebhookConfig{},
	)
}

func initBlockchains(cfg *config.Config) map[string]blockchain.Chain {
	chains := make(map[string]blockchain.Chain)

	// Ethereum
	ethClient, err := ethereum.NewClient(
		cfg.Blockchain.Ethereum.RPCURL,
		cfg.Blockchain.Ethereum.ChainID,
		cfg.Blockchain.Ethereum.Confirmations,
	)
	if err != nil {
		logger.Warnf("Failed to initialize Ethereum client: %v", err)
	} else {
		chains["ethereum"] = ethClient
	}

	// BSC (EVM compatible)
	bscClient, err := ethereum.NewClientWithName(
		cfg.Blockchain.BSC.RPCURL,
		cfg.Blockchain.BSC.ChainID,
		cfg.Blockchain.BSC.Confirmations,
		"bsc",
	)
	if err != nil {
		logger.Warnf("Failed to initialize BSC client: %v", err)
	} else {
		chains["bsc"] = bscClient
	}

	// Polygon (EVM compatible)
	polygonClient, err := ethereum.NewClientWithName(
		cfg.Blockchain.Polygon.RPCURL,
		cfg.Blockchain.Polygon.ChainID,
		cfg.Blockchain.Polygon.Confirmations,
		"polygon",
	)
	if err != nil {
		logger.Warnf("Failed to initialize Polygon client: %v", err)
	} else {
		chains["polygon"] = polygonClient
	}

	// Bitcoin
	btcClient, err := bitcoin.NewClient(
		cfg.Blockchain.Bitcoin.RPCURL,
		cfg.Blockchain.Bitcoin.RPCUser,
		cfg.Blockchain.Bitcoin.RPCPassword,
		cfg.Blockchain.Bitcoin.Network,
		cfg.Blockchain.Bitcoin.Confirmations,
	)
	if err != nil {
		logger.Warnf("Failed to initialize Bitcoin client: %v", err)
	} else {
		chains["bitcoin"] = btcClient
	}

	// Tron
	tronClient, err := tron.NewClient(
		cfg.Blockchain.Tron.RPCURL,
		cfg.Blockchain.Tron.APIKey,
		cfg.Blockchain.Tron.Network,
		cfg.Blockchain.Tron.Confirmations,
	)
	if err != nil {
		logger.Warnf("Failed to initialize Tron client: %v", err)
	} else {
		chains["tron"] = tronClient
	}

	return chains
}

type services struct {
	account      account.Service
	wallet       wallet.Service
	keyManager   keymanager.Service
	transaction  transaction.Service
	deposit      deposit.Service
	withdrawal   withdrawal.Service
	asset        asset.Service
	riskControl  riskcontrol.Service
	audit        audit.Service
	notification notification.Service
}

func initServices(cfg *config.Config, blockchains map[string]blockchain.Chain) *services {
	db := database.GetDB()

	// Repositories
	accountRepo := account.NewRepository(db)
	walletRepo := wallet.NewRepository(db)
	keyManagerRepo := keymanager.NewRepository(db)
	transactionRepo := transaction.NewRepository(db)
	depositRepo := deposit.NewRepository(db)
	withdrawalRepo := withdrawal.NewRepository(db)
	assetRepo := asset.NewRepository(db)
	riskControlRepo := riskcontrol.NewRepository(db)
	auditRepo := audit.NewRepository(db)
	notificationRepo := notification.NewRepository(db)

	// Services
	keyManagerSvc := keymanager.NewService(keyManagerRepo, cfg.JWT.Secret)
	riskControlSvc := riskcontrol.NewService(riskControlRepo)

	return &services{
		account:      account.NewService(accountRepo, cfg.JWT.Secret, cfg.JWT.ExpireTime),
		wallet:       wallet.NewService(walletRepo, keyManagerSvc),
		keyManager:   keyManagerSvc,
		transaction:  transaction.NewService(transactionRepo, keyManagerSvc, blockchains),
		deposit:      deposit.NewService(depositRepo, walletRepo, keyManagerSvc, blockchains),
		withdrawal:   withdrawal.NewService(withdrawalRepo, walletRepo, keyManagerSvc, riskControlSvc, blockchains),
		asset:        asset.NewService(assetRepo),
		riskControl:  riskControlSvc,
		audit:        audit.NewService(auditRepo),
		notification: notification.NewService(notificationRepo),
	}
}
