package grpc

import (
	"context"

	"custodial-wallet/internal/withdrawal"
	pb "custodial-wallet/api/proto/wallet/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WithdrawalServer gRPC提现服务
type WithdrawalServer struct {
	pb.UnimplementedWithdrawalServiceServer
	service withdrawal.Service
}

// NewWithdrawalServer 创建提现服务
func NewWithdrawalServer(service withdrawal.Service) *WithdrawalServer {
	return &WithdrawalServer{service: service}
}

// CreateWithdrawal 创建提现
func (s *WithdrawalServer) CreateWithdrawal(ctx context.Context, req *pb.CreateWithdrawalRequest) (*pb.CreateWithdrawalResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	w, err := s.service.CreateWithdrawal(&withdrawal.CreateWithdrawalRequest{
		UserID:          userID,
		Chain:           req.Chain,
		ToAddress:       req.ToAddress,
		Currency:        req.Currency,
		Amount:          req.Amount,
		ContractAddress: req.ContractAddress,
		Memo:            req.Memo,
	})
	if err != nil {
		switch err {
		case withdrawal.ErrInsufficientBalance:
			return nil, status.Error(codes.FailedPrecondition, "insufficient balance")
		case withdrawal.ErrExceedDailyLimit, withdrawal.ErrExceedSingleLimit:
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		case withdrawal.ErrBelowMinAmount:
			return nil, status.Error(codes.InvalidArgument, "below minimum amount")
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.CreateWithdrawalResponse{
		Withdrawal: withdrawalToProto(w),
	}, nil
}

// GetWithdrawal 获取提现
func (s *WithdrawalServer) GetWithdrawal(ctx context.Context, req *pb.GetWithdrawalRequest) (*pb.GetWithdrawalResponse, error) {
	var w *withdrawal.Withdrawal
	var err error

	if req.Uuid != "" {
		w, err = s.service.GetWithdrawalByUUID(req.Uuid)
	} else {
		w, err = s.service.GetWithdrawal(uint(req.Id))
	}

	if err != nil {
		if err == withdrawal.ErrWithdrawalNotFound {
			return nil, status.Error(codes.NotFound, "withdrawal not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetWithdrawalResponse{
		Withdrawal: withdrawalToProto(w),
	}, nil
}

// ListWithdrawals 列出提现
func (s *WithdrawalServer) ListWithdrawals(ctx context.Context, req *pb.ListWithdrawalsRequest) (*pb.ListWithdrawalsResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	withdrawals, total, err := s.service.ListWithdrawals(userID, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbWithdrawals := make([]*pb.Withdrawal, 0, len(withdrawals))
	for _, w := range withdrawals {
		pbWithdrawals = append(pbWithdrawals, withdrawalToProto(w))
	}

	return &pb.ListWithdrawalsResponse{
		Withdrawals: pbWithdrawals,
		Total:       total,
		Page:        int32(page),
		PageSize:    int32(pageSize),
	}, nil
}

// CancelWithdrawal 取消提现
func (s *WithdrawalServer) CancelWithdrawal(ctx context.Context, req *pb.CancelWithdrawalRequest) (*pb.CancelWithdrawalResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.service.CancelWithdrawal(uint(req.Id), userID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CancelWithdrawalResponse{}, nil
}

// withdrawalToProto 转换Withdrawal到Proto
func withdrawalToProto(w *withdrawal.Withdrawal) *pb.Withdrawal {
	if w == nil {
		return nil
	}
	pbWithdrawal := &pb.Withdrawal{
		Id:            uint64(w.ID),
		Uuid:          w.UUID,
		UserId:        uint64(w.UserID),
		Chain:         w.Chain,
		TxHash:        w.TxHash,
		FromAddress:   w.FromAddress,
		ToAddress:     w.ToAddress,
		Currency:      w.Currency,
		Amount:        w.Amount,
		Fee:           w.Fee,
		Status:        int32(w.Status),
		RiskLevel:     int32(w.RiskLevel),
		ManualReview:  w.ManualReview,
		Confirmations: int32(w.Confirmations),
		Memo:          w.Memo,
		ErrorMsg:      w.ErrorMsg,
		CreatedAt:     w.CreatedAt,
	}
	if w.CompletedAt != nil {
		pbWithdrawal.CompletedAt = *w.CompletedAt
	}
	return pbWithdrawal
}
