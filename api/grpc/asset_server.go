package grpc

import (
	"context"

	"custodial-wallet/internal/asset"
	pb "custodial-wallet/api/proto/wallet/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AssetServer gRPC资产服务
type AssetServer struct {
	pb.UnimplementedAssetServiceServer
	service asset.Service
}

// NewAssetServer 创建资产服务
func NewAssetServer(service asset.Service) *AssetServer {
	return &AssetServer{service: service}
}

// ListAssets 列出资产
func (s *AssetServer) ListAssets(ctx context.Context, req *pb.ListAssetsRequest) (*pb.ListAssetsResponse, error) {
	assets, err := s.service.ListAssets(req.Chain)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbAssets := make([]*pb.Asset, 0, len(assets))
	for _, a := range assets {
		pbAssets = append(pbAssets, assetToProto(a))
	}

	return &pb.ListAssetsResponse{
		Assets: pbAssets,
	}, nil
}

// GetUserAssets 获取用户资产
func (s *AssetServer) GetUserAssets(ctx context.Context, req *pb.GetUserAssetsRequest) (*pb.GetUserAssetsResponse, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	assets, err := s.service.GetUserAssets(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	totalValue, err := s.service.GetUserTotalValue(userID)
	if err != nil {
		totalValue = "0"
	}

	pbAssets := make([]*pb.UserAssetDetail, 0, len(assets))
	for _, a := range assets {
		pbAssets = append(pbAssets, userAssetDetailToProto(a))
	}

	return &pb.GetUserAssetsResponse{
		Assets:        pbAssets,
		TotalValueUsd: totalValue,
	}, nil
}

// GetAssetPrice 获取资产价格
func (s *AssetServer) GetAssetPrice(ctx context.Context, req *pb.GetAssetPriceRequest) (*pb.GetAssetPriceResponse, error) {
	price, err := s.service.GetPrice(req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if price == nil {
		return nil, status.Error(codes.NotFound, "price not found")
	}

	return &pb.GetAssetPriceResponse{
		Price: assetPriceToProto(price),
	}, nil
}

// assetToProto 转换Asset到Proto
func assetToProto(a *asset.Asset) *pb.Asset {
	if a == nil {
		return nil
	}
	return &pb.Asset{
		Id:              uint64(a.ID),
		Chain:           a.Chain,
		Symbol:          a.Symbol,
		Name:            a.Name,
		ContractAddress: a.ContractAddress,
		Decimals:        int32(a.Decimals),
		Type:            string(a.Type),
		IconUrl:         a.IconURL,
		DepositEnabled:  a.DepositEnabled,
		WithdrawEnabled: a.WithdrawEnabled,
		Status:          int32(a.Status),
	}
}

// assetPriceToProto 转换AssetPrice到Proto
func assetPriceToProto(p *asset.AssetPrice) *pb.AssetPrice {
	if p == nil {
		return nil
	}
	return &pb.AssetPrice{
		Symbol:     p.Symbol,
		PriceUsd:   p.PriceUSD,
		PriceBtc:   p.PriceBTC,
		Change_24H: p.Change24h,
		Volume_24H: p.Volume24h,
		UpdatedAt:  p.UpdatedAt,
	}
}

// userAssetDetailToProto 转换UserAssetDetail到Proto
func userAssetDetailToProto(a *asset.UserAssetDetail) *pb.UserAssetDetail {
	if a == nil {
		return nil
	}
	return &pb.UserAssetDetail{
		Asset:     assetToProto(a.Asset),
		Available: a.Balance.Available,
		Frozen:    a.Balance.Frozen,
		Pending:   a.Balance.Pending,
		ValueUsd:  a.ValueUSD,
	}
}
