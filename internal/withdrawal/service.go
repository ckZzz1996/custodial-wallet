package withdrawal

import (
	"errors"
	"time"

	"custodial-wallet/internal/blockchain"
	"custodial-wallet/internal/keymanager"
	"custodial-wallet/internal/riskcontrol"
	"custodial-wallet/internal/wallet"
	"custodial-wallet/pkg/logger"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrWithdrawalNotFound    = errors.New("withdrawal not found")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrExceedDailyLimit      = errors.New("exceed daily limit")
	ErrExceedSingleLimit     = errors.New("exceed single limit")
	ErrBelowMinAmount        = errors.New("below minimum amount")
	ErrAddressNotWhitelisted = errors.New("address not whitelisted")
)

// Service 提现服务接口
type Service interface {
	CreateWithdrawal(req *CreateWithdrawalRequest) (*Withdrawal, error)
	GetWithdrawal(withdrawalID uint) (*Withdrawal, error)
	GetWithdrawalByUUID(uuid string) (*Withdrawal, error)
	ListWithdrawals(userID uint, page, pageSize int) ([]*Withdrawal, int64, error)

	ApproveWithdrawal(withdrawalID uint, reviewerID uint, note string) error
	RejectWithdrawal(withdrawalID uint, reviewerID uint, note string) error
	CancelWithdrawal(withdrawalID uint, userID uint) error

	ProcessApprovedWithdrawals() error
	CheckConfirmations(chain string) error

	SetLimit(userID uint, chain, currency string, limit *WithdrawalLimit) error
	GetLimit(userID uint, chain, currency string) (*WithdrawalLimit, error)
}

type service struct {
	repo        Repository
	walletRepo  wallet.Repository
	keyManager  keymanager.Service
	riskControl riskcontrol.Service
	blockchains map[string]blockchain.Chain
}

// NewService 创建提现服务
func NewService(
	repo Repository,
	walletRepo wallet.Repository,
	keyManager keymanager.Service,
	riskControl riskcontrol.Service,
	blockchains map[string]blockchain.Chain,
) Service {
	return &service{
		repo:        repo,
		walletRepo:  walletRepo,
		keyManager:  keyManager,
		riskControl: riskControl,
		blockchains: blockchains,
	}
}

// CreateWithdrawalRequest 创建提现请求
type CreateWithdrawalRequest struct {
	UserID          uint   `json:"-"`
	WalletID        uint   `json:"wallet_id"`
	Chain           string `json:"chain" binding:"required"`
	ToAddress       string `json:"to_address" binding:"required"`
	Currency        string `json:"currency" binding:"required"`
	Amount          string `json:"amount" binding:"required"`
	ContractAddress string `json:"contract_address"`
	Memo            string `json:"memo"`
}

// CreateWithdrawal 创建提现
func (s *service) CreateWithdrawal(req *CreateWithdrawalRequest) (*Withdrawal, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, errors.New("invalid amount")
	}

	// 检查余额
	balance, err := s.walletRepo.GetBalance(req.UserID, wallet.Chain(req.Chain), req.Currency)
	if err != nil {
		return nil, err
	}
	if balance == nil {
		return nil, ErrInsufficientBalance
	}

	availableBalance, _ := decimal.NewFromString(balance.Available)
	if availableBalance.LessThan(amount) {
		return nil, ErrInsufficientBalance
	}

	// 检查限额
	if err := s.checkLimits(req.UserID, req.Chain, req.Currency, amount); err != nil {
		return nil, err
	}

	// 风控检查
	riskResult, err := s.riskControl.CheckWithdrawalRisk(&riskcontrol.WithdrawalRiskRequest{
		UserID:    req.UserID,
		Chain:     req.Chain,
		ToAddress: req.ToAddress,
		Currency:  req.Currency,
		Amount:    req.Amount,
	})
	if err != nil {
		return nil, err
	}

	// 冻结余额
	if err := s.walletRepo.FreezeBalance(req.UserID, wallet.Chain(req.Chain), req.Currency, req.Amount); err != nil {
		return nil, err
	}

	// 估算手续费
	chain, ok := s.blockchains[req.Chain]
	var fee string
	if ok {
		fee, _ = chain.EstimateFee("", req.ToAddress, req.Amount)
	}

	// 创建提现记录
	withdrawal := &Withdrawal{
		UUID:            uuid.New().String(),
		UserID:          req.UserID,
		WalletID:        req.WalletID,
		Chain:           req.Chain,
		ToAddress:       req.ToAddress,
		Currency:        req.Currency,
		ContractAddress: req.ContractAddress,
		Amount:          req.Amount,
		Fee:             fee,
		Status:          WithdrawalStatusPending,
		RiskLevel:       riskResult.RiskLevel,
		Memo:            req.Memo,
	}

	// 根据风控结果设置状态
	if riskResult.NeedManualReview {
		withdrawal.Status = WithdrawalStatusManualReview
		withdrawal.ManualReview = true
	} else if riskResult.RiskLevel > 0 {
		withdrawal.Status = WithdrawalStatusRiskReview
		withdrawal.RiskReview = true
	} else {
		withdrawal.Status = WithdrawalStatusApproved
	}

	if err := s.repo.Create(withdrawal); err != nil {
		// 回滚冻结
		_ = s.walletRepo.UnfreezeBalance(req.UserID, wallet.Chain(req.Chain), req.Currency, req.Amount)
		return nil, err
	}

	logger.Infof("Withdrawal created: %s, %s %s to %s, status: %d",
		withdrawal.UUID, req.Amount, req.Currency, req.ToAddress, withdrawal.Status)
	return withdrawal, nil
}

func (s *service) checkLimits(userID uint, chain, currency string, amount decimal.Decimal) error {
	// 获取用户限额或全局限额
	limit, err := s.repo.GetLimit(userID, chain, currency)
	if err != nil {
		return err
	}
	if limit == nil {
		limit, err = s.repo.GetGlobalLimit(chain, currency)
		if err != nil {
			return err
		}
	}

	if limit != nil {
		// 检查最小金额
		if limit.MinAmount != "" {
			minAmount, _ := decimal.NewFromString(limit.MinAmount)
			if amount.LessThan(minAmount) {
				return ErrBelowMinAmount
			}
		}

		// 检查单笔限额
		if limit.MaxAmount != "" {
			maxAmount, _ := decimal.NewFromString(limit.MaxAmount)
			if amount.GreaterThan(maxAmount) {
				return ErrExceedSingleLimit
			}
		}

		// 检查日限额
		if limit.DailyLimit != "" {
			dailyWithdrawal, _ := s.repo.GetUserDailyWithdrawal(userID, chain, currency)
			dailyTotal, _ := decimal.NewFromString(dailyWithdrawal)
			dailyLimit, _ := decimal.NewFromString(limit.DailyLimit)
			if dailyTotal.Add(amount).GreaterThan(dailyLimit) {
				return ErrExceedDailyLimit
			}
		}
	}

	return nil
}

// GetWithdrawal 获取提现
func (s *service) GetWithdrawal(withdrawalID uint) (*Withdrawal, error) {
	w, err := s.repo.GetByID(withdrawalID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWithdrawalNotFound
	}
	return w, nil
}

// GetWithdrawalByUUID 通过UUID获取提现
func (s *service) GetWithdrawalByUUID(uuid string) (*Withdrawal, error) {
	w, err := s.repo.GetByUUID(uuid)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWithdrawalNotFound
	}
	return w, nil
}

// ListWithdrawals 列出提现
func (s *service) ListWithdrawals(userID uint, page, pageSize int) ([]*Withdrawal, int64, error) {
	return s.repo.ListByUserID(userID, page, pageSize)
}

// ApproveWithdrawal 批准提现
func (s *service) ApproveWithdrawal(withdrawalID uint, reviewerID uint, note string) error {
	w, err := s.repo.GetByID(withdrawalID)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrWithdrawalNotFound
	}

	if w.Status != WithdrawalStatusRiskReview && w.Status != WithdrawalStatusManualReview {
		return errors.New("withdrawal is not pending review")
	}

	now := time.Now()
	w.Status = WithdrawalStatusApproved
	w.ReviewedBy = reviewerID
	w.ReviewedAt = &now
	w.ReviewNote = note

	if err := s.repo.Update(w); err != nil {
		return err
	}

	logger.Infof("Withdrawal approved: %s by user %d", w.UUID, reviewerID)
	return nil
}

// RejectWithdrawal 拒绝提现
func (s *service) RejectWithdrawal(withdrawalID uint, reviewerID uint, note string) error {
	w, err := s.repo.GetByID(withdrawalID)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrWithdrawalNotFound
	}

	now := time.Now()
	w.Status = WithdrawalStatusRejected
	w.ReviewedBy = reviewerID
	w.ReviewedAt = &now
	w.ReviewNote = note

	if err := s.repo.Update(w); err != nil {
		return err
	}

	// 解冻余额
	_ = s.walletRepo.UnfreezeBalance(w.UserID, wallet.Chain(w.Chain), w.Currency, w.Amount)

	logger.Infof("Withdrawal rejected: %s by user %d", w.UUID, reviewerID)
	return nil
}

// CancelWithdrawal 取消提现
func (s *service) CancelWithdrawal(withdrawalID uint, userID uint) error {
	w, err := s.repo.GetByID(withdrawalID)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrWithdrawalNotFound
	}

	if w.UserID != userID {
		return errors.New("withdrawal does not belong to user")
	}

	if w.Status != WithdrawalStatusPending &&
		w.Status != WithdrawalStatusRiskReview &&
		w.Status != WithdrawalStatusManualReview {
		return errors.New("withdrawal cannot be cancelled")
	}

	w.Status = WithdrawalStatusCancelled
	if err := s.repo.Update(w); err != nil {
		return err
	}

	// 解冻余额
	_ = s.walletRepo.UnfreezeBalance(w.UserID, wallet.Chain(w.Chain), w.Currency, w.Amount)

	logger.Infof("Withdrawal cancelled: %s by user %d", w.UUID, userID)
	return nil
}

// ProcessApprovedWithdrawals 处理已批准的提现
func (s *service) ProcessApprovedWithdrawals() error {
	withdrawals, err := s.repo.ListByStatus(WithdrawalStatusApproved, 50)
	if err != nil {
		return err
	}

	for _, w := range withdrawals {
		if err := s.processWithdrawal(w); err != nil {
			logger.Errorf("Failed to process withdrawal %d: %v", w.ID, err)
		}
	}

	return nil
}

func (s *service) processWithdrawal(w *Withdrawal) error {
	chain, ok := s.blockchains[w.Chain]
	if !ok {
		return errors.New("unsupported chain")
	}

	// 更新状态为处理中
	w.Status = WithdrawalStatusProcessing
	if err := s.repo.Update(w); err != nil {
		return err
	}

	// 获取热钱包地址
	// TODO: 从配置或热钱包管理获取
	hotWalletAddress := ""

	// 构建交易
	rawTx, err := chain.BuildTransaction(hotWalletAddress, w.ToAddress, w.Amount, w.ContractAddress)
	if err != nil {
		w.Status = WithdrawalStatusFailed
		w.ErrorMsg = err.Error()
		_ = s.repo.Update(w)
		return err
	}

	// 签名
	signature, err := s.keyManager.Sign(0, w.Chain, hotWalletAddress, []byte(rawTx))
	if err != nil {
		w.Status = WithdrawalStatusFailed
		w.ErrorMsg = err.Error()
		_ = s.repo.Update(w)
		return err
	}

	// 广播
	txHash, err := chain.BroadcastTransaction(string(signature))
	if err != nil {
		w.Status = WithdrawalStatusFailed
		w.ErrorMsg = err.Error()
		_ = s.repo.Update(w)
		return err
	}

	w.TxHash = txHash
	w.FromAddress = hotWalletAddress
	w.Status = WithdrawalStatusBroadcast
	if err := s.repo.Update(w); err != nil {
		return err
	}

	logger.Infof("Withdrawal broadcast: %s, hash: %s", w.UUID, txHash)
	return nil
}

// CheckConfirmations 检查确认
func (s *service) CheckConfirmations(chainName string) error {
	chain, ok := s.blockchains[chainName]
	if !ok {
		return errors.New("unsupported chain")
	}

	withdrawals, err := s.repo.ListPendingConfirmation(chainName, 100)
	if err != nil {
		return err
	}

	requiredConfirmations := chain.GetRequiredConfirmations()

	for _, w := range withdrawals {
		if w.TxHash == "" {
			continue
		}

		txInfo, err := chain.GetTransaction(w.TxHash)
		if err != nil {
			continue
		}

		if txInfo == nil {
			continue
		}

		w.Confirmations = txInfo.Confirmations
		w.BlockNumber = txInfo.BlockNumber

		if txInfo.Status == 2 { // Failed
			w.Status = WithdrawalStatusFailed
			w.ErrorMsg = "transaction failed on chain"
			// 解冻余额
			_ = s.walletRepo.UnfreezeBalance(w.UserID, wallet.Chain(w.Chain), w.Currency, w.Amount)
		} else if txInfo.Confirmations >= requiredConfirmations {
			now := time.Now()
			w.Status = WithdrawalStatusCompleted
			w.CompletedAt = &now
			// 从冻结余额扣除
			_ = s.walletRepo.DecrementBalance(w.UserID, wallet.Chain(w.Chain), w.Currency, w.Amount)
			logger.Infof("Withdrawal completed: %s", w.UUID)
		} else {
			w.Status = WithdrawalStatusConfirming
		}

		_ = s.repo.Update(w)
	}

	return nil
}

// SetLimit 设置限额
func (s *service) SetLimit(userID uint, chain, currency string, limit *WithdrawalLimit) error {
	existing, err := s.repo.GetLimit(userID, chain, currency)
	if err != nil {
		return err
	}

	if existing != nil {
		existing.MinAmount = limit.MinAmount
		existing.MaxAmount = limit.MaxAmount
		existing.DailyLimit = limit.DailyLimit
		existing.MonthlyLimit = limit.MonthlyLimit
		existing.RequireReview = limit.RequireReview
		return s.repo.UpdateLimit(existing)
	}

	limit.UserID = userID
	limit.Chain = chain
	limit.Currency = currency
	return s.repo.CreateLimit(limit)
}

// GetLimit 获取限额
func (s *service) GetLimit(userID uint, chain, currency string) (*WithdrawalLimit, error) {
	limit, err := s.repo.GetLimit(userID, chain, currency)
	if err != nil {
		return nil, err
	}
	if limit == nil {
		return s.repo.GetGlobalLimit(chain, currency)
	}
	return limit, nil
}
