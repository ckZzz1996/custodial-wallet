package transaction

import (
	"errors"
	"time"

	"custodial-wallet/internal/blockchain"
	"custodial-wallet/internal/keymanager"
	"custodial-wallet/pkg/logger"

	"github.com/google/uuid"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidTransaction  = errors.New("invalid transaction")
	ErrBroadcastFailed     = errors.New("broadcast failed")
)

// Service 交易服务接口
type Service interface {
	CreateTransaction(req *CreateTxRequest) (*Transaction, error)
	GetTransaction(txID uint) (*Transaction, error)
	GetTransactionByUUID(uuid string) (*Transaction, error)
	GetTransactionByHash(chain, txHash string) (*Transaction, error)
	ListTransactions(userID uint, page, pageSize int) ([]*Transaction, int64, error)
	SignTransaction(txID uint) (*Transaction, error)
	BroadcastTransaction(txID uint) (*Transaction, error)
	UpdateTransactionStatus(txID uint, status TxStatus, errorMsg string) error
	ProcessPendingTransactions() error
	CheckConfirmations(chain string) error
}

type service struct {
	repo        Repository
	keyManager  keymanager.Service
	blockchains map[string]blockchain.Chain
}

// NewService 创建交易服务
func NewService(repo Repository, keyManager keymanager.Service, blockchains map[string]blockchain.Chain) Service {
	return &service{
		repo:        repo,
		keyManager:  keyManager,
		blockchains: blockchains,
	}
}

// CreateTxRequest 创建交易请求
type CreateTxRequest struct {
	UserID          uint   `json:"-"`
	WalletID        uint   `json:"wallet_id"`
	Chain           string `json:"chain" binding:"required"`
	FromAddress     string `json:"from_address" binding:"required"`
	ToAddress       string `json:"to_address" binding:"required"`
	Currency        string `json:"currency" binding:"required"`
	Amount          string `json:"amount" binding:"required"`
	ContractAddress string `json:"contract_address"`
	Memo            string `json:"memo"`
	Type            TxType `json:"type"`
}

// CreateTransaction 创建交易
func (s *service) CreateTransaction(req *CreateTxRequest) (*Transaction, error) {
	chain, ok := s.blockchains[req.Chain]
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	// 估算手续费
	fee, err := chain.EstimateFee(req.FromAddress, req.ToAddress, req.Amount)
	if err != nil {
		logger.Warnf("Failed to estimate fee: %v", err)
	}

	tx := &Transaction{
		UUID:            uuid.New().String(),
		UserID:          req.UserID,
		WalletID:        req.WalletID,
		Chain:           req.Chain,
		FromAddress:     req.FromAddress,
		ToAddress:       req.ToAddress,
		Currency:        req.Currency,
		ContractAddress: req.ContractAddress,
		Amount:          req.Amount,
		Fee:             fee,
		Type:            req.Type,
		Status:          TxStatusPending,
		Memo:            req.Memo,
	}

	if err := s.repo.Create(tx); err != nil {
		return nil, err
	}

	logger.Infof("Transaction created: %s", tx.UUID)
	return tx, nil
}

// GetTransaction 获取交易
func (s *service) GetTransaction(txID uint) (*Transaction, error) {
	tx, err := s.repo.GetByID(txID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ErrTransactionNotFound
	}
	return tx, nil
}

// GetTransactionByUUID 通过UUID获取交易
func (s *service) GetTransactionByUUID(uuid string) (*Transaction, error) {
	tx, err := s.repo.GetByUUID(uuid)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ErrTransactionNotFound
	}
	return tx, nil
}

// GetTransactionByHash 通过哈希获取交易
func (s *service) GetTransactionByHash(chain, txHash string) (*Transaction, error) {
	return s.repo.GetByTxHash(chain, txHash)
}

// ListTransactions 列出交易
func (s *service) ListTransactions(userID uint, page, pageSize int) ([]*Transaction, int64, error) {
	return s.repo.ListByUserID(userID, page, pageSize)
}

// SignTransaction 签名交易
func (s *service) SignTransaction(txID uint) (*Transaction, error) {
	tx, err := s.repo.GetByID(txID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ErrTransactionNotFound
	}

	if tx.Status != TxStatusPending {
		return nil, errors.New("transaction is not pending")
	}

	chain, ok := s.blockchains[tx.Chain]
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	// 构建交易
	rawTx, err := chain.BuildTransaction(tx.FromAddress, tx.ToAddress, tx.Amount, tx.ContractAddress)
	if err != nil {
		tx.Status = TxStatusFailed
		tx.ErrorMsg = err.Error()
		_ = s.repo.Update(tx)
		return tx, err
	}
	tx.RawTx = rawTx

	// 签名
	signedTx, err := s.keyManager.Sign(tx.UserID, tx.Chain, tx.FromAddress, []byte(rawTx))
	if err != nil {
		tx.Status = TxStatusFailed
		tx.ErrorMsg = err.Error()
		_ = s.repo.Update(tx)
		return tx, err
	}
	tx.SignedTx = string(signedTx)
	tx.Status = TxStatusSigned

	if err := s.repo.Update(tx); err != nil {
		return nil, err
	}

	logger.Infof("Transaction signed: %s", tx.UUID)
	return tx, nil
}

// BroadcastTransaction 广播交易
func (s *service) BroadcastTransaction(txID uint) (*Transaction, error) {
	tx, err := s.repo.GetByID(txID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ErrTransactionNotFound
	}

	if tx.Status != TxStatusSigned {
		return nil, errors.New("transaction is not signed")
	}

	chain, ok := s.blockchains[tx.Chain]
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	// 广播
	txHash, err := chain.BroadcastTransaction(tx.SignedTx)
	if err != nil {
		tx.Status = TxStatusFailed
		tx.ErrorMsg = err.Error()
		_ = s.repo.Update(tx)
		return tx, ErrBroadcastFailed
	}

	tx.TxHash = txHash
	tx.Status = TxStatusBroadcast

	if err := s.repo.Update(tx); err != nil {
		return nil, err
	}

	logger.Infof("Transaction broadcast: %s, hash: %s", tx.UUID, txHash)
	return tx, nil
}

// UpdateTransactionStatus 更新交易状态
func (s *service) UpdateTransactionStatus(txID uint, status TxStatus, errorMsg string) error {
	return s.repo.UpdateStatus(txID, status, errorMsg)
}

// ProcessPendingTransactions 处理待处理的交易
func (s *service) ProcessPendingTransactions() error {
	// 获取待签名的交易
	pendingTxs, err := s.repo.ListByStatus(TxStatusPending, 100)
	if err != nil {
		return err
	}

	for _, tx := range pendingTxs {
		_, err := s.SignTransaction(tx.ID)
		if err != nil {
			logger.Errorf("Failed to sign transaction %d: %v", tx.ID, err)
			continue
		}

		_, err = s.BroadcastTransaction(tx.ID)
		if err != nil {
			logger.Errorf("Failed to broadcast transaction %d: %v", tx.ID, err)
		}
	}

	return nil
}

// CheckConfirmations 检查交易确认
func (s *service) CheckConfirmations(chainName string) error {
	txs, err := s.repo.ListPendingConfirmation(chainName, 100)
	if err != nil {
		return err
	}

	chain, ok := s.blockchains[chainName]
	if !ok {
		return errors.New("unsupported chain")
	}

	requiredConfirmations := chain.GetRequiredConfirmations()

	for _, tx := range txs {
		if tx.TxHash == "" {
			continue
		}

		// 获取链上交易信息
		txInfo, err := chain.GetTransaction(tx.TxHash)
		if err != nil {
			logger.Warnf("Failed to get transaction %s: %v", tx.TxHash, err)
			continue
		}

		if txInfo == nil {
			continue
		}

		// 更新确认数
		if txInfo.Confirmations != tx.Confirmations {
			if err := s.repo.UpdateConfirmations(tx.ID, txInfo.Confirmations, txInfo.BlockNumber, txInfo.BlockHash); err != nil {
				logger.Errorf("Failed to update confirmations for tx %d: %v", tx.ID, err)
				continue
			}
		}

		// 检查是否达到确认要求
		if txInfo.Confirmations >= requiredConfirmations {
			now := time.Now()
			tx.Status = TxStatusConfirmed
			tx.ConfirmedAt = &now
			tx.Confirmations = txInfo.Confirmations
			tx.BlockNumber = txInfo.BlockNumber
			tx.BlockHash = txInfo.BlockHash
			if err := s.repo.Update(tx); err != nil {
				logger.Errorf("Failed to update tx %d: %v", tx.ID, err)
			}
			logger.Infof("Transaction confirmed: %s", tx.TxHash)
		} else {
			tx.Status = TxStatusConfirming
			_ = s.repo.Update(tx)
		}
	}

	return nil
}
