package deposit

import (
	"errors"
	"math/big"
	"strings"

	"custodial-wallet/internal/blockchain"
	"custodial-wallet/internal/wallet"
	"custodial-wallet/pkg/logger"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrDepositNotFound = errors.New("deposit not found")
	ErrAddressNotFound = errors.New("address not found")
)

// Service 充值服务接口
type Service interface {
	// 充值地址管理
	AllocateDepositAddress(userID uint, chain string) (*DepositAddress, error)
	GetDepositAddress(userID uint, chain string) (*DepositAddress, error)
	ListDepositAddresses(userID uint) ([]*DepositAddress, error)

	// 充值记录
	GetDeposit(depositID uint) (*Deposit, error)
	GetDepositByTxHash(txHash string) (*Deposit, error)
	ListDeposits(userID uint, page, pageSize int) ([]*Deposit, int64, error)

	// 充值处理
	ProcessDeposit(chain, txHash, fromAddress, toAddress, currency, amount string, blockNumber uint64) error
	ConfirmDeposit(depositID uint) error
	CreditDeposit(depositID uint) error

	// 链上监控
	ScanDeposits(chain string) error
	CheckConfirmations(chain string) error
	ProcessCredits() error

	// 归集
	CreateSweepTask(chain, fromAddress, toAddress, currency, amount string) (*SweepTask, error)
	ProcessSweepTasks(chain string) error
}

type service struct {
	repo                  Repository
	walletRepo            wallet.Repository
	blockchains           map[string]blockchain.Chain
	confirmationsRequired map[string]int
}

// NewService 创建充值服务
func NewService(
	repo Repository,
	walletRepo wallet.Repository,
	blockchains map[string]blockchain.Chain,
) Service {
	confirmations := make(map[string]int)
	for name, chain := range blockchains {
		confirmations[name] = chain.GetRequiredConfirmations()
	}

	return &service{
		repo:                  repo,
		walletRepo:            walletRepo,
		blockchains:           blockchains,
		confirmationsRequired: confirmations,
	}
}

// AllocateDepositAddress 分配充值地址
func (s *service) AllocateDepositAddress(userID uint, chain string) (*DepositAddress, error) {
	// 检查是否已有地址
	existing, err := s.repo.GetUserDepositAddress(userID, chain)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// 从钱包模块获取新地址
	addresses, err := s.walletRepo.ListAddressesByUserID(userID, wallet.Chain(chain))
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, errors.New("no address available, please generate one first")
	}

	// 使用第一个可用地址作为充值地址
	addr := &DepositAddress{
		UserID:  userID,
		Chain:   chain,
		Address: addresses[0].Address,
		Status:  1,
	}

	if err := s.repo.CreateDepositAddress(addr); err != nil {
		return nil, err
	}

	logger.Infof("Deposit address allocated: %s on %s for user %d", addr.Address, chain, userID)
	return addr, nil
}

// GetDepositAddress 获取充值地址
func (s *service) GetDepositAddress(userID uint, chain string) (*DepositAddress, error) {
	addr, err := s.repo.GetUserDepositAddress(userID, chain)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		return nil, ErrAddressNotFound
	}
	return addr, nil
}

// ListDepositAddresses 列出充值地址
func (s *service) ListDepositAddresses(userID uint) ([]*DepositAddress, error) {
	return s.repo.ListDepositAddresses(userID)
}

// GetDeposit 获取充值记录
func (s *service) GetDeposit(depositID uint) (*Deposit, error) {
	deposit, err := s.repo.GetDepositByID(depositID)
	if err != nil {
		return nil, err
	}
	if deposit == nil {
		return nil, ErrDepositNotFound
	}
	return deposit, nil
}

// GetDepositByTxHash 通过交易哈希获取充值
func (s *service) GetDepositByTxHash(txHash string) (*Deposit, error) {
	return s.repo.GetDepositByTxHash(txHash)
}

// ListDeposits 列出充值记录
func (s *service) ListDeposits(userID uint, page, pageSize int) ([]*Deposit, int64, error) {
	return s.repo.ListDepositsByUserID(userID, page, pageSize)
}

// ProcessDeposit 处理充值
func (s *service) ProcessDeposit(chain, txHash, fromAddress, toAddress, currency, amount string, blockNumber uint64) error {
	// 检查是否已存在
	existing, _ := s.repo.GetDepositByTxHash(txHash)
	if existing != nil {
		return nil // 已处理
	}

	// 查找充值地址归属
	depositAddr, err := s.repo.GetDepositAddress(chain, toAddress)
	if err != nil {
		return err
	}
	if depositAddr == nil {
		return nil // 不是我们的地址
	}

	// 获取用户钱包信息
	wallets, err := s.walletRepo.ListWalletsByUserID(depositAddr.UserID)
	if err != nil {
		return err
	}

	var walletID uint
	if len(wallets) > 0 {
		walletID = wallets[0].ID
	}

	// 创建充值记录
	deposit := &Deposit{
		UUID:        uuid.New().String(),
		UserID:      depositAddr.UserID,
		WalletID:    walletID,
		Chain:       chain,
		TxHash:      txHash,
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Currency:    currency,
		Amount:      amount,
		Status:      DepositStatusPending,
		BlockNumber: blockNumber,
	}

	if err := s.repo.CreateDeposit(deposit); err != nil {
		return err
	}

	logger.Infof("Deposit detected: %s, %s %s to %s", txHash, amount, currency, toAddress)
	return nil
}

// ConfirmDeposit 确认充值
func (s *service) ConfirmDeposit(depositID uint) error {
	deposit, err := s.repo.GetDepositByID(depositID)
	if err != nil {
		return err
	}
	if deposit == nil {
		return ErrDepositNotFound
	}

	deposit.Status = DepositStatusConfirmed
	return s.repo.UpdateDeposit(deposit)
}

// CreditDeposit 入账充值
func (s *service) CreditDeposit(depositID uint) error {
	deposit, err := s.repo.GetDepositByID(depositID)
	if err != nil {
		return err
	}
	if deposit == nil {
		return ErrDepositNotFound
	}

	if deposit.Credited {
		return nil // 已入账
	}

	// 增加用户余额
	amount, _ := decimal.NewFromString(deposit.Amount)
	if err := s.walletRepo.IncrementBalance(
		deposit.UserID,
		wallet.Chain(deposit.Chain),
		deposit.Currency,
		amount.String(),
	); err != nil {
		return err
	}

	// 更新充值状态
	if err := s.repo.CreditDeposit(depositID); err != nil {
		return err
	}

	logger.Infof("Deposit credited: %s, %s %s for user %d",
		deposit.TxHash, deposit.Amount, deposit.Currency, deposit.UserID)
	return nil
}

// ScanDeposits 扫描链上充值（支持ETH主币和ERC20 Transfer事件）
func (s *service) ScanDeposits(chainName string) error {
	chain, ok := s.blockchains[chainName]
	if !ok {
		return errors.New("unsupported chain")
	}

	// 获取上次已扫描区块号
	lastScanned, err := s.repo.GetLastScannedBlock(chainName)
	if err != nil {
		return err
	}

	// 获取当前最新区块号
	latestBlock, err := chain.GetBlockNumber()
	if err != nil {
		return err
	}

	// 限制每次最多扫描的区块数，防止首次启动时压力过大
	const maxBlocks = 200
	if latestBlock > lastScanned+maxBlocks {
		latestBlock = lastScanned + maxBlocks
	}

	// 读取本链所有充值地址并归一化
	addrs, err := s.repo.ListAllDepositAddresses(chainName)
	if err != nil {
		return err
	}
	addrMap := make(map[string]struct{}, len(addrs))
	for _, a := range addrs {
		addrMap[strings.ToLower(a.Address)] = struct{}{}
	}

	// 尝试断言链实现是否支持 GetBlock / GetLogs（以太坊客户端提供）
	type blockGetter interface {
		GetBlock(uint64) (*blockchain.Block, error)
	}
	type logGetter interface {
		GetLogs(uint64, uint64, []string) ([]types.Log, error)
	}

	bg, hasBlock := chain.(blockGetter)
	lg, hasLogs := chain.(logGetter)

	transferTopic := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	for blk := lastScanned + 1; blk <= latestBlock; blk++ {
		if hasBlock {
			block, err := bg.GetBlock(blk)
			if err != nil {
				logger.Errorf("failed to fetch block %d for %s: %v", blk, chainName, err)
				// 不更新 lastScanned，让下次继续尝试
				continue
			}

			// 遍历区块内交易（主币转账）
			for _, txHash := range block.Transactions {
				if txHash == "" {
					continue
				}
				// 获取交易详情
				txInfo, err := chain.GetTransaction(txHash)
				if err != nil {
					logger.Debugf("GetTransaction %s err: %v", txHash, err)
					continue
				}
				if txInfo == nil {
					continue
				}
				if txInfo.To == "" || txInfo.Amount == "" {
					continue
				}
				if _, exists := addrMap[strings.ToLower(txInfo.To)]; exists {
					// 发现充值（ETH）
					_ = s.ProcessDeposit(chainName, txInfo.TxHash, txInfo.From, txInfo.To, "ETH", txInfo.Amount, txInfo.BlockNumber)
				}
			}
		}

		// 如果支持日志查询，扫描 ERC20 Transfer 事件
		if hasLogs && lg != nil {
			logs, err := lg.GetLogs(blk, blk, nil)
			if err != nil {
				// 只记录日志错误，不终止扫描
				logger.Debugf("GetLogs for block %d returned error: %v", blk, err)
			} else {
				for _, lgEntry := range logs {
					// 检查是否为 Transfer topic
					if len(lgEntry.Topics) == 0 || lgEntry.Topics[0] != transferTopic {
						continue
					}

					if len(lgEntry.Topics) < 3 {
						continue
					}

					// topics[1]=from, topics[2]=to
					from := common.HexToAddress(lgEntry.Topics[1].Hex()).Hex()
					to := common.HexToAddress(lgEntry.Topics[2].Hex()).Hex()

					if _, ok := addrMap[strings.ToLower(to)]; !ok {
						continue
					}

					// amount in data (big-endian)
					amount := new(big.Int).SetBytes(lgEntry.Data).String()
					contract := lgEntry.Address.Hex()
					_ = s.ProcessDeposit(chainName, lgEntry.TxHash.Hex(), from, to, contract, amount, blk)
				}
			}
		}

		// 更新最后扫描区块（即使部分tx失败也推进）
		if err := s.repo.SetLastScannedBlock(chainName, blk); err != nil {
			logger.Errorf("failed to set last scanned block for %s to %d: %v", chainName, blk, err)
		}
	}

	logger.Infof("Scanned deposits for chain %s blocks %d..%d", chainName, lastScanned+1, latestBlock)
	return nil
}

// CheckConfirmations 检查确认数
func (s *service) CheckConfirmations(chainName string) error {
	chain, ok := s.blockchains[chainName]
	if !ok {
		return errors.New("unsupported chain")
	}

	requiredConfirmations := s.confirmationsRequired[chainName]

	// 获取待确认的充值
	deposits, err := s.repo.ListPendingDeposits(chainName, 100)
	if err != nil {
		return err
	}

	currentBlock, err := chain.GetBlockNumber()
	if err != nil {
		return err
	}

	for _, deposit := range deposits {
		if deposit.BlockNumber == 0 {
			// 获取交易信息
			txInfo, err := chain.GetTransaction(deposit.TxHash)
			if err != nil {
				continue
			}
			if txInfo != nil && txInfo.BlockNumber > 0 {
				deposit.BlockNumber = txInfo.BlockNumber
				deposit.BlockHash = txInfo.BlockHash
			}
		}

		if deposit.BlockNumber > 0 {
			confirmations := int(currentBlock - deposit.BlockNumber + 1)
			deposit.Confirmations = confirmations

			if confirmations >= requiredConfirmations {
				deposit.Status = DepositStatusConfirmed
				logger.Infof("Deposit confirmed: %s with %d confirmations", deposit.TxHash, confirmations)
			} else {
				deposit.Status = DepositStatusConfirming
			}

			_ = s.repo.UpdateDeposit(deposit)
		}
	}

	return nil
}

// ProcessCredits 处理入账
func (s *service) ProcessCredits() error {
	// 获取已确认但未入账的充值
	deposits, err := s.repo.ListUnconfirmedDeposits("", 100)
	if err != nil {
		return err
	}

	for _, deposit := range deposits {
		if err := s.CreditDeposit(deposit.ID); err != nil {
			logger.Errorf("Failed to credit deposit %d: %v", deposit.ID, err)
		}
	}

	return nil
}

// CreateSweepTask 创建归集任务
func (s *service) CreateSweepTask(chain, fromAddress, toAddress, currency, amount string) (*SweepTask, error) {
	task := &SweepTask{
		Chain:       chain,
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Currency:    currency,
		Amount:      amount,
		Status:      0,
	}

	if err := s.repo.CreateSweepTask(task); err != nil {
		return nil, err
	}

	logger.Infof("Sweep task created: %s %s from %s to %s", amount, currency, fromAddress, toAddress)
	return task, nil
}

// ProcessSweepTasks 处理归集任务
func (s *service) ProcessSweepTasks(chainName string) error {
	tasks, err := s.repo.ListPendingSweepTasks(chainName, 50)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		// TODO: 实现归集交易
		logger.Infof("Processing sweep task %d", task.ID)
	}

	return nil
}
