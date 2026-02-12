package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"custodial-wallet/internal/blockchain"
	"custodial-wallet/internal/blockchain/ethereum"
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
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志
	logger.Init(cfg.App.Env)
	defer logger.Sync()

	logger.Info("Starting worker...")

	// 初始化数据库
	if err := database.Init(cfg.Database); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 初始化Redis
	if err := cache.Init(cfg.Redis); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer cache.Close()

	// 初始化区块链客户端
	blockchains := initBlockchains(cfg)

	// 初始化服务
	services := initServices(cfg, blockchains)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动后台任务
	go runDepositScanner(ctx, services.deposit)
	go runWithdrawalProcessor(ctx, services.withdrawal)
	go runConfirmationChecker(ctx, services.deposit, services.withdrawal, blockchains)
	go runNotificationProcessor(ctx, services.notification)

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")
	cancel()

	// 等待任务完成
	time.Sleep(5 * time.Second)
	logger.Info("Worker exited")
}

func initBlockchains(cfg *config.Config) map[string]blockchain.Chain {
	chains := make(map[string]blockchain.Chain)

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

	return chains
}

type workerServices struct {
	deposit      deposit.Service
	withdrawal   withdrawal.Service
	transaction  transaction.Service
	notification notification.Service
}

func initServices(cfg *config.Config, blockchains map[string]blockchain.Chain) *workerServices {
	db := database.GetDB()

	walletRepo := wallet.NewRepository(db)
	keyManagerRepo := keymanager.NewRepository(db)
	depositRepo := deposit.NewRepository(db)
	withdrawalRepo := withdrawal.NewRepository(db)
	transactionRepo := transaction.NewRepository(db)
	riskControlRepo := riskcontrol.NewRepository(db)
	notificationRepo := notification.NewRepository(db)

	keyManagerSvc := keymanager.NewService(keyManagerRepo, cfg.JWT.Secret)
	riskControlSvc := riskcontrol.NewService(riskControlRepo)

	return &workerServices{
		deposit:      deposit.NewService(depositRepo, walletRepo, keyManagerSvc, blockchains),
		withdrawal:   withdrawal.NewService(withdrawalRepo, walletRepo, keyManagerSvc, riskControlSvc, blockchains),
		transaction:  transaction.NewService(transactionRepo, keyManagerSvc, blockchains),
		notification: notification.NewService(notificationRepo),
	}
}

// runDepositScanner 运行充值扫描
func runDepositScanner(ctx context.Context, svc deposit.Service) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 扫描各链的充值
			chains := []string{"ethereum", "bitcoin", "tron"}
			for _, chain := range chains {
				if err := svc.ScanDeposits(chain); err != nil {
					logger.Errorf("Failed to scan deposits for %s: %v", chain, err)
				}
			}
		}
	}
}

// runWithdrawalProcessor 运行提现处理
func runWithdrawalProcessor(ctx context.Context, svc withdrawal.Service) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.ProcessApprovedWithdrawals(); err != nil {
				logger.Errorf("Failed to process withdrawals: %v", err)
			}
		}
	}
}

// runConfirmationChecker 运行确认检查
func runConfirmationChecker(ctx context.Context, depositSvc deposit.Service, withdrawalSvc withdrawal.Service, blockchains map[string]blockchain.Chain) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for chain := range blockchains {
				// 检查充值确认
				if err := depositSvc.CheckConfirmations(chain); err != nil {
					logger.Errorf("Failed to check deposit confirmations for %s: %v", chain, err)
				}

				// 处理入账
				if err := depositSvc.ProcessCredits(); err != nil {
					logger.Errorf("Failed to process credits: %v", err)
				}

				// 检查提现确认
				if err := withdrawalSvc.CheckConfirmations(chain); err != nil {
					logger.Errorf("Failed to check withdrawal confirmations for %s: %v", chain, err)
				}
			}
		}
	}
}

// runNotificationProcessor 运行通知处理
func runNotificationProcessor(ctx context.Context, svc notification.Service) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.ProcessPendingNotifications(); err != nil {
				logger.Errorf("Failed to process notifications: %v", err)
			}
		}
	}
}
