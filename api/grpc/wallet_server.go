package grpc

import (
	"context"

	"custodial-wallet/internal/wallet"
	pb "custodial-wallet/api/proto/wallet/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WalletServer gRPC钱包服务
type WalletServer struct {
	pb.UnimplementedWalletServiceServer
	service wallet.Service
}

// NewWalletServer 创建钱包服务
func NewWalletServer(service wallet.Service) *WalletServer {
	return &WalletServer{service: service}
}

// CreateWallet 创建钱包
func (s *WalletServer) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.CreateWalletResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	walletType := wallet.WalletTypeHot
	if req.Type == 2 {
		walletType = wallet.WalletTypeCold
	}

	w, err := s.service.CreateWallet(userID, req.Name, walletType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CreateWalletResponse{
		Wallet: walletToProto(w),
	}, nil
}

// GetWallet 获取钱包
func (s *WalletServer) GetWallet(ctx context.Context, req *pb.GetWalletRequest) (*pb.GetWalletResponse, error) {
	var w *wallet.Wallet
	var err error

	if req.Uuid != "" {
		w, err = s.service.GetWalletByUUID(req.Uuid)
	} else {
		w, err = s.service.GetWallet(uint(req.Id))
	}

	if err != nil {
		if err == wallet.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetWalletResponse{
		Wallet: walletToProto(w),
	}, nil
}

// ListWallets 列出钱包
func (s *WalletServer) ListWallets(ctx context.Context, req *pb.ListWalletsRequest) (*pb.ListWalletsResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	wallets, err := s.service.ListWallets(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbWallets := make([]*pb.Wallet, 0, len(wallets))
	for _, w := range wallets {
		pbWallets = append(pbWallets, walletToProto(w))
	}

	return &pb.ListWalletsResponse{
		Wallets: pbWallets,
	}, nil
}

// UpdateWallet 更新钱包
func (s *WalletServer) UpdateWallet(ctx context.Context, req *pb.UpdateWalletRequest) (*pb.UpdateWalletResponse, error) {
	w, err := s.service.UpdateWallet(uint(req.Id), req.Name)
	if err != nil {
		if err == wallet.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateWalletResponse{
		Wallet: walletToProto(w),
	}, nil
}

// DeleteWallet 删除钱包
func (s *WalletServer) DeleteWallet(ctx context.Context, req *pb.DeleteWalletRequest) (*pb.DeleteWalletResponse, error) {
	if err := s.service.DeleteWallet(uint(req.Id)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.DeleteWalletResponse{}, nil
}

// GenerateAddress 生成地址
func (s *WalletServer) GenerateAddress(ctx context.Context, req *pb.GenerateAddressRequest) (*pb.GenerateAddressResponse, error) {
	addr, err := s.service.GenerateAddress(uint(req.WalletId), wallet.Chain(req.Chain), req.Label)
	if err != nil {
		if err == wallet.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GenerateAddressResponse{
		Address: addressToProto(addr),
	}, nil
}

// ListAddresses 列出地址
func (s *WalletServer) ListAddresses(ctx context.Context, req *pb.ListAddressesRequest) (*pb.ListAddressesResponse, error) {
	addresses, err := s.service.ListAddresses(uint(req.WalletId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbAddresses := make([]*pb.Address, 0, len(addresses))
	for _, addr := range addresses {
		pbAddresses = append(pbAddresses, addressToProto(addr))
	}

	return &pb.ListAddressesResponse{
		Addresses: pbAddresses,
	}, nil
}

// GetDepositAddress 获取充值地址
func (s *WalletServer) GetDepositAddress(ctx context.Context, req *pb.GetDepositAddressRequest) (*pb.GetDepositAddressResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	addr, err := s.service.GetDepositAddress(userID, wallet.Chain(req.Chain))
	if err != nil {
		if err == wallet.ErrAddressNotFound {
			return nil, status.Error(codes.NotFound, "no deposit address found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetDepositAddressResponse{
		Address: addressToProto(addr),
	}, nil
}

// GetBalance 获取余额
func (s *WalletServer) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	balance, err := s.service.GetBalance(userID, wallet.Chain(req.Chain), req.Currency)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetBalanceResponse{
		Balance: balanceToProto(balance),
	}, nil
}

// ListBalances 列出余额
func (s *WalletServer) ListBalances(ctx context.Context, req *pb.ListBalancesRequest) (*pb.ListBalancesResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	balances, err := s.service.ListBalances(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbBalances := make([]*pb.Balance, 0, len(balances))
	for _, b := range balances {
		pbBalances = append(pbBalances, balanceToProto(b))
	}

	return &pb.ListBalancesResponse{
		Balances: pbBalances,
	}, nil
}

// walletToProto 转换Wallet到Proto
func walletToProto(w *wallet.Wallet) *pb.Wallet {
	if w == nil {
		return nil
	}
	pbWallet := &pb.Wallet{
		Id:        uint64(w.ID),
		Uuid:      w.UUID,
		UserId:    uint64(w.UserID),
		Name:      w.Name,
		Type:      int32(w.Type),
		Status:    int32(w.Status),
		CreatedAt: w.CreatedAt,
	}
	for _, addr := range w.Addresses {
		pbWallet.Addresses = append(pbWallet.Addresses, addressToProto(&addr))
	}
	return pbWallet
}

// addressToProto 转换Address到Proto
func addressToProto(addr *wallet.Address) *pb.Address {
	if addr == nil {
		return nil
	}
	return &pb.Address{
		Id:             uint64(addr.ID),
		Uuid:           addr.UUID,
		WalletId:       uint64(addr.WalletID),
		Chain:          string(addr.Chain),
		Address:        addr.Address,
		Label:          addr.Label,
		DerivationPath: addr.DerivationPath,
		Type:           int32(addr.Type),
		Status:         int32(addr.Status),
		CreatedAt:      addr.CreatedAt,
	}
}

// balanceToProto 转换Balance到Proto
func balanceToProto(b *wallet.Balance) *pb.Balance {
	if b == nil {
		return nil
	}
	return &pb.Balance{
		Id:              uint64(b.ID),
		WalletId:        uint64(b.WalletID),
		Chain:           string(b.Chain),
		Currency:        b.Currency,
		ContractAddress: b.ContractAddr,
		Available:       b.Available,
		Frozen:          b.Frozen,
		Pending:         b.Pending,
		UpdatedAt:       b.UpdatedAt,
	}
}
