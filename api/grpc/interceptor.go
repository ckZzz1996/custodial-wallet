package grpc

import (
	"context"
	"time"

	"custodial-wallet/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
)

var jwtSecret []byte

// SetJWTSecret 设置JWT密钥
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

// GetUserIDFromContext 从上下文获取用户ID
func GetUserIDFromContext(ctx context.Context) (uint, error) {
	userID, ok := ctx.Value(userIDKey).(uint)
	if !ok {
		return 0, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	return userID, nil
}

// AuthInterceptor 认证拦截器
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 不需要认证的方法
	publicMethods := map[string]bool{
		"/wallet.v1.AccountService/Register": true,
		"/wallet.v1.AccountService/Login":    true,
	}

	if publicMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	// 从metadata获取token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	authHeader := authHeaders[0]
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	tokenString := authHeader[7:]

	// 解析token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}

	// token 中 user_id 可能为 float64
	if uid, ok := claims["user_id"].(float64); ok {
		userID := uint(uid)
		ctx = context.WithValue(ctx, userIDKey, userID)
	}

	return handler(ctx, req)
}

// LoggingInterceptor 日志拦截器
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	latency := time.Since(start)
	if err != nil {
		logger.Errorf("gRPC %s error: %v, took=%s", info.FullMethod, err, latency)
	} else {
		logger.Infof("gRPC %s OK took=%s", info.FullMethod, latency)
	}
	return resp, err
}

// RecoveryInterceptor 恢复拦截器
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = status.Errorf(codes.Internal, "panic: %v", r)
		}
	}()
	return handler(ctx, req)
}

// StreamAuthInterceptor 流式认证拦截器
func StreamAuthInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// 流式方法的认证
	ctx := ss.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization header")
	}

	authHeader := authHeaders[0]
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	tokenString := authHeader[7:]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return status.Error(codes.Unauthenticated, "invalid token")
	}

	return handler(srv, ss)
}
