package riskcontrol

import (
	"encoding/json"
	"time"

	"custodial-wallet/pkg/logger"

	"github.com/shopspring/decimal"
)

// Service 风控服务接口
type Service interface {
	// 风险检查
	CheckWithdrawalRisk(req *WithdrawalRiskRequest) (*RiskCheckResult, error)
	CheckDepositRisk(req *DepositRiskRequest) (*RiskCheckResult, error)
	CheckLoginRisk(req *LoginRiskRequest) (*RiskCheckResult, error)

	// 规则管理
	CreateRule(rule *RiskRule) error
	GetRule(ruleID uint) (*RiskRule, error)
	ListRules(ruleType RuleType) ([]*RiskRule, error)
	UpdateRule(rule *RiskRule) error
	DeleteRule(ruleID uint) error

	// 黑名单管理
	AddToBlacklist(blType, value, chain, reason string, createdBy uint) error
	RemoveFromBlacklist(id uint) error
	IsBlacklisted(blType, value, chain string) (bool, error)
	ListBlacklist(blType string, page, pageSize int) ([]*Blacklist, int64, error)

	// 用户风险画像
	GetUserRiskProfile(userID uint) (*UserRiskProfile, error)
	GetUserRiskScore(userID uint) (int, error)

	// 风控日志
	ListRiskLogs(userID uint, limit int) ([]*RiskLog, error)
}

type service struct {
	repo Repository
}

// NewService 创建风控服务
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// WithdrawalRiskRequest 提现风险检查请求
type WithdrawalRiskRequest struct {
	UserID    uint   `json:"user_id"`
	Chain     string `json:"chain"`
	ToAddress string `json:"to_address"`
	Currency  string `json:"currency"`
	Amount    string `json:"amount"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

// DepositRiskRequest 充值风险检查请求
type DepositRiskRequest struct {
	UserID      uint   `json:"user_id"`
	Chain       string `json:"chain"`
	FromAddress string `json:"from_address"`
	Currency    string `json:"currency"`
	Amount      string `json:"amount"`
}

// LoginRiskRequest 登录风险检查请求
type LoginRiskRequest struct {
	UserID    uint   `json:"user_id"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Device    string `json:"device"`
}

// RiskCheckResult 风险检查结果
type RiskCheckResult struct {
	Passed           bool   `json:"passed"`
	RiskLevel        int    `json:"risk_level"` // 0=low, 1=medium, 2=high
	NeedManualReview bool   `json:"need_manual_review"`
	Blocked          bool   `json:"blocked"`
	Reason           string `json:"reason"`
	MatchedRules     []uint `json:"matched_rules"`
}

// CheckWithdrawalRisk 检查提现风险
func (s *service) CheckWithdrawalRisk(req *WithdrawalRiskRequest) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:       true,
		RiskLevel:    0,
		MatchedRules: []uint{},
	}

	// 检查地址黑名单
	isBlacklisted, err := s.repo.CheckBlacklist("address", req.ToAddress, req.Chain)
	if err != nil {
		return nil, err
	}
	if isBlacklisted {
		result.Passed = false
		result.Blocked = true
		result.Reason = "address is blacklisted"
		s.logRiskCheck(req.UserID, "withdrawal", 0, "block", req)
		return result, nil
	}

	// 检查用户黑名单
	isUserBlacklisted, _ := s.repo.CheckBlacklist("user", string(rune(req.UserID)), "")
	if isUserBlacklisted {
		result.Passed = false
		result.Blocked = true
		result.Reason = "user is blacklisted"
		return result, nil
	}

	// 获取所有活跃规则
	rules, err := s.repo.ListActiveRules()
	if err != nil {
		return nil, err
	}

	amount, _ := decimal.NewFromString(req.Amount)

	for _, rule := range rules {
		if rule.Chain != "" && rule.Chain != req.Chain {
			continue
		}
		if rule.Currency != "" && rule.Currency != req.Currency {
			continue
		}

		matched, action := s.evaluateRule(rule, amount, req.UserID)
		if matched {
			result.MatchedRules = append(result.MatchedRules, rule.ID)
			if rule.RiskLevel > result.RiskLevel {
				result.RiskLevel = rule.RiskLevel
			}

			switch action {
			case "block":
				result.Passed = false
				result.Blocked = true
				result.Reason = rule.Name
			case "review":
				result.NeedManualReview = true
			}
		}
	}

	// 记录日志
	logResult := "pass"
	if result.Blocked {
		logResult = "block"
	} else if result.NeedManualReview {
		logResult = "review"
	}
	s.logRiskCheck(req.UserID, "withdrawal", result.RiskLevel, logResult, req)

	return result, nil
}

func (s *service) evaluateRule(rule *RiskRule, amount decimal.Decimal, userID uint) (bool, string) {
	var condition map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Condition), &condition); err != nil {
		return false, ""
	}

	switch rule.Type {
	case RuleTypeAmountLimit:
		if maxStr, ok := condition["max_amount"].(string); ok {
			maxAmount, _ := decimal.NewFromString(maxStr)
			if amount.GreaterThan(maxAmount) {
				return true, rule.Action
			}
		}
	case RuleTypeFrequencyLimit:
		// TODO: 实现频率限制检查
	case RuleTypeKYCRequired:
		// TODO: 实现KYC检查
	}

	return false, ""
}

func (s *service) logRiskCheck(userID uint, action string, riskLevel int, result string, req interface{}) {
	reqData, _ := json.Marshal(req)
	log := &RiskLog{
		UserID:      userID,
		Action:      action,
		RiskLevel:   riskLevel,
		Result:      result,
		RequestData: string(reqData),
	}
	_ = s.repo.CreateRiskLog(log)
}

// CheckDepositRisk 检查充值风险
func (s *service) CheckDepositRisk(req *DepositRiskRequest) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:       true,
		RiskLevel:    0,
		MatchedRules: []uint{},
	}

	// 检查来源地址黑名单
	isBlacklisted, err := s.repo.CheckBlacklist("address", req.FromAddress, req.Chain)
	if err != nil {
		return nil, err
	}
	if isBlacklisted {
		result.RiskLevel = 2
		result.NeedManualReview = true
		result.Reason = "source address is blacklisted"
	}

	return result, nil
}

// CheckLoginRisk 检查登录风险
func (s *service) CheckLoginRisk(req *LoginRiskRequest) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:       true,
		RiskLevel:    0,
		MatchedRules: []uint{},
	}

	// 检查IP黑名单
	isIPBlacklisted, _ := s.repo.CheckBlacklist("ip", req.IP, "")
	if isIPBlacklisted {
		result.Passed = false
		result.Blocked = true
		result.Reason = "IP is blacklisted"
		return result, nil
	}

	// 检查设备黑名单
	if req.Device != "" {
		isDeviceBlacklisted, _ := s.repo.CheckBlacklist("device", req.Device, "")
		if isDeviceBlacklisted {
			result.Passed = false
			result.Blocked = true
			result.Reason = "device is blacklisted"
			return result, nil
		}
	}

	return result, nil
}

// CreateRule 创建规则
func (s *service) CreateRule(rule *RiskRule) error {
	if err := s.repo.CreateRule(rule); err != nil {
		return err
	}
	logger.Infof("Risk rule created: %s", rule.Name)
	return nil
}

// GetRule 获取规则
func (s *service) GetRule(ruleID uint) (*RiskRule, error) {
	return s.repo.GetRuleByID(ruleID)
}

// ListRules 列出规则
func (s *service) ListRules(ruleType RuleType) ([]*RiskRule, error) {
	return s.repo.ListRules(ruleType, -1)
}

// UpdateRule 更新规则
func (s *service) UpdateRule(rule *RiskRule) error {
	return s.repo.UpdateRule(rule)
}

// DeleteRule 删除规则
func (s *service) DeleteRule(ruleID uint) error {
	return s.repo.DeleteRule(ruleID)
}

// AddToBlacklist 添加到黑名单
func (s *service) AddToBlacklist(blType, value, chain, reason string, createdBy uint) error {
	bl := &Blacklist{
		Type:      blType,
		Value:     value,
		Chain:     chain,
		Reason:    reason,
		Status:    1,
		CreatedBy: createdBy,
	}
	if err := s.repo.CreateBlacklist(bl); err != nil {
		return err
	}
	logger.Infof("Added to blacklist: %s=%s", blType, value)
	return nil
}

// RemoveFromBlacklist 从黑名单移除
func (s *service) RemoveFromBlacklist(id uint) error {
	return s.repo.DeleteBlacklist(id)
}

// IsBlacklisted 检查是否在黑名单
func (s *service) IsBlacklisted(blType, value, chain string) (bool, error) {
	return s.repo.CheckBlacklist(blType, value, chain)
}

// ListBlacklist 列出黑名单
func (s *service) ListBlacklist(blType string, page, pageSize int) ([]*Blacklist, int64, error) {
	return s.repo.ListBlacklist(blType, page, pageSize)
}

// GetUserRiskProfile 获取用户风险画像
func (s *service) GetUserRiskProfile(userID uint) (*UserRiskProfile, error) {
	profile, err := s.repo.GetUserRiskProfile(userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		// 创建默认画像
		profile = &UserRiskProfile{
			UserID:    userID,
			RiskScore: 0,
		}
		if err := s.repo.CreateUserRiskProfile(profile); err != nil {
			return nil, err
		}
	}
	return profile, nil
}

// GetUserRiskScore 获取用户风险分数
func (s *service) GetUserRiskScore(userID uint) (int, error) {
	profile, err := s.GetUserRiskProfile(userID)
	if err != nil {
		return 0, err
	}
	return profile.RiskScore, nil
}

// ListRiskLogs 列出风控日志
func (s *service) ListRiskLogs(userID uint, limit int) ([]*RiskLog, error) {
	return s.repo.ListRiskLogsByUserID(userID, limit)
}

// UpdateUserRiskScore 更新用户风险分数
func (s *service) UpdateUserRiskScore(userID uint, delta int) error {
	profile, err := s.GetUserRiskProfile(userID)
	if err != nil {
		return err
	}
	profile.RiskScore += delta
	if profile.RiskScore < 0 {
		profile.RiskScore = 0
	}
	if profile.RiskScore > 100 {
		profile.RiskScore = 100
	}
	now := time.Now()
	profile.LastRiskCheckAt = &now
	return s.repo.UpdateUserRiskProfile(profile)
}
