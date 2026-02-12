package grpc

import (
	"net"

	"custodial-wallet/internal/account"
	"custodial-wallet/internal/asset"
	"custodial-wallet/internal/deposit"
	"custodial-wallet/internal/wallet"
	"custodial-wallet/internal/withdrawal"
	pb "custodial-wallet/api/proto/wallet/v1"
	"custodial-wallet/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server gRPC服务器
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
}

// Services 服务集合
type Services struct {
	Account    account.Service
	Wallet     wallet.Service
	Deposit    deposit.Service
	Withdrawal withdrawal.Service
	Asset      asset.Service
}

// NewServer 创建gRPC服务器
func NewServer(cfg *ServerConfig, services *Services) (*Server, error) {
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return nil, err
	}

	// 创建gRPC服务器，添加拦截器
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor,
			LoggingInterceptor,
			AuthInterceptor,
		),
		grpc.ChainStreamInterceptor(
			StreamAuthInterceptor,
		),
	)

	// 注册服务
	pb.RegisterAccountServiceServer(grpcServer, NewAccountServer(services.Account))
	pb.RegisterWalletServiceServer(grpcServer, NewWalletServer(services.Wallet))
	pb.RegisterDepositServiceServer(grpcServer, NewDepositServer(services.Deposit))
	pb.RegisterWithdrawalServiceServer(grpcServer, NewWithdrawalServer(services.Withdrawal))
	pb.RegisterAssetServiceServer(grpcServer, NewAssetServer(services.Asset))

	// 注册反射服务，方便调试
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		listener:   lis,
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	logger.Infof("gRPC server listening on %s", s.listener.Addr().String())
	return s.grpcServer.Serve(s.listener)
}

// Stop 停止服务器
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
