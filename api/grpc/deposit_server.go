package grpc

import (
	"context"

	"custodial-wallet/internal/deposit"
	pb "custodial-wallet/api/proto/wallet/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DepositServer gRPC充值服务
type DepositServer struct {
	pb.UnimplementedDepositServiceServer
	service deposit.Service
}

// NewDepositServer 创建充值服务
func NewDepositServer(service deposit.Service) *DepositServer {
	return &DepositServer{service: service}
}

// GetDeposit 获取充值记录
func (s *DepositServer) GetDeposit(ctx context.Context, req *pb.GetDepositRequest) (*pb.GetDepositResponse, error) {
	var d *deposit.Deposit
	var err error

	if req.Uuid != "" {
		d, err = s.service.GetDepositByTxHash(req.Uuid)
	} else {
		d, err = s.service.GetDeposit(uint(req.Id))
	}

	if err != nil {
		if err == deposit.ErrDepositNotFound {
			return nil, status.Error(codes.NotFound, "deposit not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetDepositResponse{
		Deposit: depositToProto(d),
	}, nil
}

// ListDeposits 列出充值记录
func (s *DepositServer) ListDeposits(ctx context.Context, req *pb.ListDepositsRequest) (*pb.ListDepositsResponse, error) {
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

	deposits, total, err := s.service.ListDeposits(userID, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbDeposits := make([]*pb.Deposit, 0, len(deposits))
	for _, d := range deposits {
		pbDeposits = append(pbDeposits, depositToProto(d))
	}

	return &pb.ListDepositsResponse{
		Deposits: pbDeposits,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

// AllocateDepositAddress 分配充值地址
func (s *DepositServer) AllocateDepositAddress(ctx context.Context, req *pb.AllocateDepositAddressRequest) (*pb.AllocateDepositAddressResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	addr, err := s.service.AllocateDepositAddress(userID, req.Chain)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.AllocateDepositAddressResponse{
		Address: depositAddressToProto(addr),
	}, nil
}

// ListDepositAddresses 列出充值地址
func (s *DepositServer) ListDepositAddresses(ctx context.Context, req *pb.ListDepositAddressesRequest) (*pb.ListDepositAddressesResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	addresses, err := s.service.ListDepositAddresses(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbAddresses := make([]*pb.DepositAddress, 0, len(addresses))
	for _, addr := range addresses {
		pbAddresses = append(pbAddresses, depositAddressToProto(addr))
	}

	return &pb.ListDepositAddressesResponse{
		Addresses: pbAddresses,
	}, nil
}

// depositToProto 转换Deposit到Proto
func depositToProto(d *deposit.Deposit) *pb.Deposit {
	if d == nil {
		return nil
	}
	pbDeposit := &pb.Deposit{
		Id:            uint64(d.ID),
		Uuid:          d.UUID,
		UserId:        uint64(d.UserID),
		Chain:         d.Chain,
		TxHash:        d.TxHash,
		FromAddress:   d.FromAddress,
		ToAddress:     d.ToAddress,
		Currency:      d.Currency,
		Amount:        d.Amount,
		Fee:           d.Fee,
		Status:        int32(d.Status),
		Confirmations: int32(d.Confirmations),
		BlockNumber:   d.BlockNumber,
		CreatedAt:     d.CreatedAt,
	}
	if d.CreditedAt != nil {
		pbDeposit.CreditedAt = *d.CreditedAt
	}
	return pbDeposit
}

// depositAddressToProto 转换DepositAddress到Proto
func depositAddressToProto(addr *deposit.DepositAddress) *pb.DepositAddress {
	if addr == nil {
		return nil
	}
	return &pb.DepositAddress{
		Id:        uint64(addr.ID),
		UserId:    uint64(addr.UserID),
		Chain:     addr.Chain,
		Address:   addr.Address,
		Label:     addr.Label,
		Status:    int32(addr.Status),
		CreatedAt: addr.CreatedAt,
	}
}
