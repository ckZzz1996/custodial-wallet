# Custodial Wallet

一个完整的托管钱包系统，支持多链资产管理、充值、提现、风控等功能。

## 功能特性

### 核心功能
- ✅ **账户管理** - 用户注册、登录、KYC、2FA
- ✅ **钱包管理** - 多链钱包创建、地址生成
- ✅ **密钥管理** - HD钱包、私钥加密存储、签名服务
- ✅ **充值管理** - 链上监控、自动入账、资金归集
- ✅ **提现管理** - 提现申请、风控审核、自动执行
- ✅ **资产管理** - 多币种、代币管理、价格同步
- ✅ **风控系统** - 限额、黑名单、异常检测
- ✅ **通知服务** - 邮件、短信、Webhook
- ✅ **审计日志** - 操作记录、合规报告

### 支持的区块链
- Ethereum (ETH, ERC20)
- Bitcoin (BTC)
- Tron (TRX, TRC20)
- BSC (BNB, BEP20)
- Polygon (MATIC)

### 双协议支持
- **HTTP API** - 基于 Gin 框架的 RESTful API
- **gRPC** - 基于 Protocol Buffers 的高性能 RPC

## 技术栈

- **语言**: Go 1.23+
- **Web框架**: Gin
- **RPC框架**: gRPC + Protocol Buffers
- **ORM**: GORM
- **数据库**: PostgreSQL
- **缓存**: Redis
- **区块链**: go-ethereum

## 快速开始

### 环境要求

- Go 1.23+
- PostgreSQL 14+
- Redis 7+
- protoc (Protocol Buffers 编译器)
- Docker & Docker Compose (可选)

### 安装

```bash
# 克隆项目
git clone https://github.com/yourname/custodial-wallet.git
cd custodial-wallet

# 安装依赖
go mod download

# 安装 protoc 插件
make proto-install

# 生成 proto 文件
make proto

# 复制配置文件
cp configs/.env.example .env

# 编辑配置文件
vim .env
```

### 使用 Docker Compose 运行

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f api
```

### 本地运行

```bash
# 启动 PostgreSQL 和 Redis
docker-compose up -d postgres redis

# 运行 API 服务 (同时启动 HTTP 和 gRPC)
make run-api

# 运行后台 Worker
make run-worker
```

### 编译

```bash
# 编译所有
make build

# 仅编译 API
make build-api

# 仅编译 Worker
make build-worker
```

## 项目结构

```
custodial-wallet/
├── cmd/                    # 应用入口
│   ├── api/               # HTTP API + gRPC 服务
│   └── worker/            # 后台任务处理
├── api/                   # API 层
│   ├── routers/           # Gin HTTP 路由处理器
│   │   ├── account.go     # 账户相关路由
│   │   ├── wallet.go      # 钱包相关路由
│   │   ├── transaction.go # 交易相关路由
│   │   ├── asset.go       # 资产相关路由
│   │   ├── middleware.go  # 中间件
│   │   └── router.go      # 路由注册
│   ├── grpc/              # gRPC 服务实现
│   │   ├── account_server.go
│   │   ├── wallet_server.go
│   │   ├── deposit_server.go
│   │   ├── withdrawal_server.go
│   │   ├── asset_server.go
│   │   ├── interceptor.go # gRPC 拦截器
│   │   └── server.go      # gRPC 服务器
│   └── proto/             # Protocol Buffers 定义
│       └── wallet/v1/
│           └── wallet.proto
├── internal/              # 内部业务模块
│   ├── account/           # 账户管理
│   ├── wallet/            # 钱包管理
│   ├── keymanager/        # 密钥管理
│   ├── transaction/       # 交易处理
│   ├── asset/             # 资产管理
│   ├── deposit/           # 充值管理
│   ├── withdrawal/        # 提现管理
│   ├── riskcontrol/       # 风控系统
│   ├── notification/      # 通知服务
│   ├── audit/             # 审计日志
│   └── blockchain/        # 区块链适配器
├── pkg/                   # 公共工具包
├── configs/               # 配置文件
├── docs/                  # 文档
└── scripts/               # 脚本
```

## API 文档

### HTTP API (Gin)

服务端口: `8080` (默认)

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/register | 用户注册 |
| POST | /api/v1/login | 用户登录 |
| GET | /api/v1/profile | 获取用户资料 |
| PUT | /api/v1/password | 修改密码 |
| POST | /api/v1/wallets | 创建钱包 |
| GET | /api/v1/wallets | 列出钱包 |
| POST | /api/v1/wallets/:id/addresses | 生成地址 |
| GET | /api/v1/balances | 查询余额 |
| GET | /api/v1/deposits | 充值记录 |
| POST | /api/v1/withdrawals | 创建提现 |
| GET | /api/v1/assets | 资产列表 |

### gRPC API

服务端口: `8081` (默认，HTTP端口+1)

```protobuf
service AccountService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc GetProfile(GetProfileRequest) returns (GetProfileResponse);
  // ...
}

service WalletService {
  rpc CreateWallet(CreateWalletRequest) returns (CreateWalletResponse);
  rpc ListWallets(ListWalletsRequest) returns (ListWalletsResponse);
  rpc GenerateAddress(GenerateAddressRequest) returns (GenerateAddressResponse);
  // ...
}

service DepositService {
  rpc ListDeposits(ListDepositsRequest) returns (ListDepositsResponse);
  rpc AllocateDepositAddress(AllocateDepositAddressRequest) returns (AllocateDepositAddressResponse);
  // ...
}

service WithdrawalService {
  rpc CreateWithdrawal(CreateWithdrawalRequest) returns (CreateWithdrawalResponse);
  rpc CancelWithdrawal(CancelWithdrawalRequest) returns (CancelWithdrawalResponse);
  // ...
}

service AssetService {
  rpc ListAssets(ListAssetsRequest) returns (ListAssetsResponse);
  rpc GetUserAssets(GetUserAssetsRequest) returns (GetUserAssetsResponse);
  // ...
}
```

### gRPC 客户端示例

```go
import (
    "google.golang.org/grpc"
    pb "custodial-wallet/api/proto/wallet/v1"
)

conn, _ := grpc.Dial("localhost:8081", grpc.WithInsecure())
client := pb.NewWalletServiceClient(conn)

// 创建钱包
resp, _ := client.CreateWallet(ctx, &pb.CreateWalletRequest{
    Name: "My Wallet",
    Type: 1,
})
```

## 配置说明

### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| APP_PORT | HTTP API 端口 | 8080 |
| APP_ENV | 环境 | development |
| DB_HOST | 数据库主机 | localhost |
| DB_PORT | 数据库端口 | 5432 |
| DB_NAME | 数据库名 | custodial_wallet |
| REDIS_HOST | Redis 主机 | localhost |
| JWT_SECRET | JWT 密钥 | - |
| ETH_RPC_URL | 以太坊 RPC | - |

> 注: gRPC 端口 = HTTP API 端口 + 1

## 开发指南

### 生成 Proto 文件

```bash
# 安装 protoc 插件
make proto-install

# 生成代码
make proto
```

### 运行测试

```bash
make test
```

### 代码检查

```bash
make lint
```

## 安全建议

1. **私钥安全**
   - 使用 HSM 进行签名
   - 私钥加密存储
   - 实施密钥分片

2. **API 安全**
   - 强制 HTTPS
   - 请求签名验证
   - 实施限流

3. **运维安全**
   - 最小权限原则
   - 定期安全审计
   - 日志监控告警

## License

MIT License
