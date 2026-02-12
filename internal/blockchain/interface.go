package blockchain

// Chain 区块链接口
type Chain interface {
	// GetName 获取链名称
	GetName() string

	// GetBalance 获取地址余额
	GetBalance(address string) (string, error)

	// GetTokenBalance 获取代币余额
	GetTokenBalance(address, contractAddress string) (string, error)

	// GetTransaction 获取交易信息
	GetTransaction(txHash string) (*TransactionInfo, error)

	// GetBlockNumber 获取最新区块号
	GetBlockNumber() (uint64, error)

	// BuildTransaction 构建交易
	BuildTransaction(from, to, amount, contractAddress string) (string, error)

	// BroadcastTransaction 广播交易
	BroadcastTransaction(signedTx string) (string, error)

	// EstimateFee 估算手续费
	EstimateFee(from, to, amount string) (string, error)

	// ValidateAddress 验证地址
	ValidateAddress(address string) bool

	// GetRequiredConfirmations 获取所需确认数
	GetRequiredConfirmations() int
}

// TransactionInfo 交易信息
type TransactionInfo struct {
	TxHash        string `json:"tx_hash"`
	From          string `json:"from"`
	To            string `json:"to"`
	Amount        string `json:"amount"`
	Fee           string `json:"fee"`
	GasPrice      string `json:"gas_price"`
	GasUsed       uint64 `json:"gas_used"`
	Nonce         uint64 `json:"nonce"`
	BlockNumber   uint64 `json:"block_number"`
	BlockHash     string `json:"block_hash"`
	Confirmations int    `json:"confirmations"`
	Status        int    `json:"status"` // 0=pending, 1=success, 2=failed
	Timestamp     int64  `json:"timestamp"`
}

// Block 区块信息
type Block struct {
	Number       uint64   `json:"number"`
	Hash         string   `json:"hash"`
	ParentHash   string   `json:"parent_hash"`
	Timestamp    int64    `json:"timestamp"`
	Transactions []string `json:"transactions"`
}
