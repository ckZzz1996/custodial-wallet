package grpc

import (
	"context"
	"time"

	pb "custodial-wallet/api/proto/wallet/v1"
	"custodial-wallet/internal/account"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AccountServer gRPC账户服务
type AccountServer struct {
	pb.UnimplementedAccountServiceServer
	service account.Service
}

// NewAccountServer 创建账户服务
func NewAccountServer(service account.Service) *AccountServer {
	return &AccountServer{service: service}
}

// Register 用户注册
func (s *AccountServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, err := s.service.Register(&account.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
	})
	if err != nil {
		if err == account.ErrUserExists {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.RegisterResponse{
		User: userToProto(user),
	}, nil
}

// Login 用户登录
func (s *AccountServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// 从metadata获取客户端信息
	md, _ := metadata.FromIncomingContext(ctx)
	ip := ""
	userAgent := ""
	if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
		ip = ips[0]
	}
	if agents := md.Get("user-agent"); len(agents) > 0 {
		userAgent = agents[0]
	}

	resp, err := s.service.Login(&account.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		TwoFACode: req.TwoFaCode,
	}, ip, userAgent)
	if err != nil {
		if err == account.ErrUserNotFound || err == account.ErrInvalidPassword {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.LoginResponse{
		Token:     resp.Token,
		ExpiresAt: resp.ExpiresAt,
		User:      userToProto(resp.User),
	}, nil
}

// GetProfile 获取用户资料
func (s *AccountServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.service.GetUser(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &pb.GetProfileResponse{
		User: userToProto(user),
	}, nil
}

// UpdateProfile 更新用户资料
func (s *AccountServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.service.UpdateUser(userID, &account.UpdateUserRequest{
		Phone: req.Phone,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateProfileResponse{
		User: userToProto(user),
	}, nil
}

// ChangePassword 修改密码
func (s *AccountServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.service.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		if err == account.ErrInvalidPassword {
			return nil, status.Error(codes.InvalidArgument, "invalid old password")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ChangePasswordResponse{}, nil
}

// Enable2FA 启用两步验证
func (s *AccountServer) Enable2FA(ctx context.Context, req *pb.Enable2FARequest) (*pb.Enable2FAResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := s.service.Enable2FA(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Enable2FAResponse{
		Secret: secret,
		// TODO: 生成二维码URL
	}, nil
}

// Verify2FA 验证两步验证
func (s *AccountServer) Verify2FA(ctx context.Context, req *pb.Verify2FARequest) (*pb.Verify2FAResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	success := s.service.Verify2FA(userID, req.Code)
	return &pb.Verify2FAResponse{
		Success: success,
	}, nil
}

// GetLoginHistory 获取登录历史
func (s *AccountServer) GetLoginHistory(ctx context.Context, req *pb.GetLoginHistoryRequest) (*pb.GetLoginHistoryResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	histories, err := s.service.ListLoginHistory(userID, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbHistories := make([]*pb.LoginHistory, 0, len(histories))
	for _, h := range histories {
		pbHistories = append(pbHistories, &pb.LoginHistory{
			Id:        uint64(h.ID),
			Ip:        h.IP,
			UserAgent: h.UserAgent,
			Device:    h.Device,
			Location:  h.Location,
			Status:    int32(h.Status),
			CreatedAt: h.CreatedAt,
		})
	}

	return &pb.GetLoginHistoryResponse{
		Histories: pbHistories,
	}, nil
}

// userToProto 转换User到Proto
func userToProto(user *account.User) *pb.User {
	if user == nil {
		return nil
	}
	pbUser := &pb.User{
		Id:           uint64(user.ID),
		Uuid:         user.UUID,
		Email:        user.Email,
		Phone:        user.Phone,
		Status:       int32(user.Status),
		KycStatus:    int32(user.KYCStatus),
		KycLevel:     int32(user.KYCLevel),
		TwoFaEnabled: user.TwoFAEnabled,
		LastLoginIp:  user.LastLoginIP,
		CreatedAt:    user.CreatedAt,
	}
	if user.LastLoginAt != nil {
		pbUser.LastLoginAt = *user.LastLoginAt
	}
	return pbUser
}

// Helper function to convert time
func timeToProto(t time.Time) interface{} {
	return t
}
