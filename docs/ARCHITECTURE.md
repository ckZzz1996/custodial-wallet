# 托管钱包架构设计

## 项目结构

```
custodial-wallet/
├── cmd/                    # 应用入口
│   ├── api/               # HTTP API 服务
│   └── worker/            # 后台任务处理
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
│       ├── bitcoin/
│       ├── ethereum/
│       └── tron/
├── pkg/                   # 公共工具包
│   ├── crypto/            # 加密工具
│   ├── database/          # 数据库连接
│   ├── cache/             # 缓存 (Redis)
│   ├── queue/             # 消息队列
│   ├── logger/            # 日志
│   ├── config/            # 配置管理
│   └── httputil/          # HTTP 工具
├── api/                   # API 定义
│   └── v1/
├── configs/               # 配置文件
├── scripts/               # 脚本
└── docs/                  # 文档
```

## 核心模块说明

### 1. Account (账户管理)
- **功能**: 用户账户的创建、认证和权限管理
- **核心实体**: User, Account, Role, Permission
- **主要接口**:
  - `CreateUser()` - 创建用户
  - `Authenticate()` - 用户认证
  - `VerifyKYC()` - KYC 验证
  - `SetPermission()` - 设置权限

### 2. Wallet (钱包管理)
- **功能**: 托管钱包的创建和管理
- **核心实体**: Wallet, Address, Chain
- **主要接口**:
  - `CreateWallet()` - 创建钱包
  - `GenerateAddress()` - 生成地址
  - `GetBalance()` - 查询余额
  - `ListAddresses()` - 列出地址

### 3. KeyManager (密钥管理) ⚠️ 核心安全模块
- **功能**: 私钥的安全存储和签名服务
- **安全措施**:
  - AES-256-GCM 加密存储
  - HSM/KMS 集成支持
  - 密钥分片 (Shamir Secret Sharing)
  - 签名请求审计
- **主要接口**:
  - `GenerateKey()` - 生成密钥对
  - `Sign()` - 签名交易
  - `EncryptKey()` - 加密存储密钥
  - `DecryptKey()` - 解密密钥

### 4. Transaction (交易处理)
- **功能**: 交易的构建、签名和广播
- **核心实体**: Transaction, TxInput, TxOutput
- **主要接口**:
  - `BuildTransaction()` - 构建交易
  - `SignTransaction()` - 签名交易
  - `BroadcastTransaction()` - 广播交易
  - `GetTransactionStatus()` - 查询交易状态

### 5. Asset (资产管理)
- **功能**: 多币种资产的管理和估值
- **核心实体**: Asset, Token, Price
- **支持类型**:
  - 原生币: BTC, ETH, TRX
  - 代币: ERC20, TRC20, BEP20
- **主要接口**:
  - `AddAsset()` - 添加资产
  - `GetAssetBalance()` - 获取资产余额
  - `GetAssetPrice()` - 获取资产价格

### 6. Deposit (充值管理)
- **功能**: 用户充值的监控和处理
- **流程**:
  1. 分配充值地址
  2. 监听链上交易
  3. 确认交易 (等待区块确认)
  4. 入账用户余额
  5. 资金归集 (可选)
- **主要接口**:
  - `AllocateAddress()` - 分配充值地址
  - `MonitorDeposit()` - 监控充值
  - `ConfirmDeposit()` - 确认充值
  - `Sweep()` - 资金归集

### 7. Withdrawal (提现管理)
- **功能**: 用户提现的处理和风控
- **流程**:
  1. 用户提交提现请求
  2. 风控审核
  3. 人工审核 (大额)
  4. 执行提现
  5. 链上确认
- **主要接口**:
  - `CreateWithdrawal()` - 创建提现请求
  - `ReviewWithdrawal()` - 审核提现
  - `ExecuteWithdrawal()` - 执行提现
  - `GetWithdrawalStatus()` - 查询状态

### 8. RiskControl (风控系统)
- **功能**: 交易风险控制和安全防护
- **规则类型**:
  - 单笔限额
  - 日限额
  - 频率限制
  - 地址黑白名单
  - 异常行为检测
- **主要接口**:
  - `CheckRisk()` - 风险检查
  - `AddBlacklist()` - 添加黑名单
  - `SetLimit()` - 设置限额
  - `GetRiskScore()` - 获取风险评分

### 9. Notification (通知服务)
- **功能**: 系统通知和告警
- **通知渠道**:
  - 邮件
  - 短信
  - Webhook
  - 站内信
- **通知类型**:
  - 充值到账
  - 提现完成
  - 安全告警
  - 系统公告

### 10. Audit (审计日志)
- **功能**: 记录所有敏感操作
- **记录内容**:
  - 用户操作日志
  - 交易记录
  - 管理员操作
  - 系统事件
- **主要接口**:
  - `Log()` - 记录日志
  - `Query()` - 查询日志
  - `Export()` - 导出报告

### 11. Blockchain (区块链适配器)
- **功能**: 不同区块链的统一接口封装
- **支持链**:
  - Bitcoin (BTC)
  - Ethereum (ETH, ERC20)
  - Tron (TRX, TRC20)
  - 可扩展其他链
- **统一接口**:
  - `GetBalance()` - 查询余额
  - `GetTransaction()` - 查询交易
  - `SendTransaction()` - 发送交易
  - `EstimateFee()` - 估算费用

## 技术栈建议

### 后端
- **语言**: Go 1.23+
- **Web框架**: Gin / Echo
- **ORM**: GORM / Ent
- **数据库**: PostgreSQL (主库) + Redis (缓存)
- **消息队列**: RabbitMQ / Kafka

### 安全
- **密钥存储**: HashiCorp Vault / AWS KMS
- **加密**: AES-256-GCM, ECDSA
- **认证**: JWT + 2FA

### 区块链
- **Bitcoin**: btcd / bitcoind RPC
- **Ethereum**: go-ethereum (geth)
- **Tron**: tron-api-go

### 监控
- **日志**: Zap + ELK
- **指标**: Prometheus + Grafana
- **链路追踪**: Jaeger

## 安全考虑

1. **私钥安全**
   - 永不明文存储
   - 使用 HSM 签名
   - 密钥分片存储

2. **API 安全**
   - HTTPS only
   - 请求签名验证
   - Rate limiting

3. **数据安全**
   - 敏感数据加密
   - 数据库加密
   - 定期备份

4. **运维安全**
   - 最小权限原则
   - 审计日志
   - 入侵检测

