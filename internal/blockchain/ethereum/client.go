package ethereum

import (
	"context"
	"math/big"
	"strings"
	"time"

	"custodial-wallet/internal/blockchain"
	"custodial-wallet/pkg/logger"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client 以太坊客户端
type Client struct {
	client        *ethclient.Client
	chainID       *big.Int
	confirmations int
	name          string
}

// NewClient 创建以太坊客户端 (默认 name = "ethereum")
func NewClient(rpcURL string, chainID int64, confirmations int) (*Client, error) {
	return NewClientWithName(rpcURL, chainID, confirmations, "ethereum")
}

// NewClientWithName 创建以太坊兼容链客户端，允许指定链名称（例如: bsc, polygon）
func NewClientWithName(rpcURL string, chainID int64, confirmations int, name string) (*Client, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		client:        client,
		chainID:       big.NewInt(chainID),
		confirmations: confirmations,
		name:          name,
	}, nil
}

// GetName 获取链名称
func (c *Client) GetName() string {
	if c.name != "" {
		return c.name
	}
	return "ethereum"
}

// GetBalance 获取ETH余额
func (c *Client) GetBalance(address string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addr := common.HexToAddress(address)
	balance, err := c.client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return "0", err
	}

	return balance.String(), nil
}

// GetTokenBalance 获取ERC20代币余额
func (c *Client) GetTokenBalance(address, contractAddress string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ERC20 balanceOf(address)
	data := common.Hex2Bytes("70a08231000000000000000000000000" + strings.TrimPrefix(address, "0x"))

	contractAddr := common.HexToAddress(contractAddress)
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	result, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return "0", err
	}

	balance := new(big.Int).SetBytes(result)
	return balance.String(), nil
}

// GetTransaction 获取交易信息
func (c *Client) GetTransaction(txHash string) (*blockchain.TransactionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hash := common.HexToHash(txHash)
	tx, isPending, err := c.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	info := &blockchain.TransactionInfo{
		TxHash:   txHash,
		Amount:   tx.Value().String(),
		GasPrice: tx.GasPrice().String(),
		Nonce:    tx.Nonce(),
	}

	if tx.To() != nil {
		info.To = tx.To().Hex()
	}

	if isPending {
		info.Status = 0
		return info, nil
	}

	// 获取交易收据
	receipt, err := c.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return info, nil
	}

	info.BlockNumber = receipt.BlockNumber.Uint64()
	info.BlockHash = receipt.BlockHash.Hex()
	info.GasUsed = receipt.GasUsed
	info.Status = int(receipt.Status)

	// 计算确认数
	currentBlock, err := c.client.BlockNumber(ctx)
	if err == nil {
		info.Confirmations = int(currentBlock - info.BlockNumber + 1)
	}

	// 获取发送者地址
	chainID, _ := c.client.ChainID(ctx)
	signer := types.LatestSignerForChainID(chainID)
	from, err := types.Sender(signer, tx)
	if err == nil {
		info.From = from.Hex()
	}

	return info, nil
}

// GetBlockNumber 获取最新区块号
func (c *Client) GetBlockNumber() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.BlockNumber(ctx)
}

// BuildTransaction 构建交易
func (c *Client) BuildTransaction(from, to, amount, contractAddress string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)

	nonce, err := c.client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return "", err
	}

	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	value := new(big.Int)
	value.SetString(amount, 10)

	var data []byte
	var gasLimit uint64 = 21000

	if contractAddress != "" {
		// ERC20 transfer
		contractAddr := common.HexToAddress(contractAddress)
		data = buildERC20TransferData(toAddr, value)
		toAddr = contractAddr
		value = big.NewInt(0)
		gasLimit = 100000
	}

	tx := types.NewTransaction(nonce, toAddr, value, gasLimit, gasPrice, data)

	// 序列化交易（未签名）
	return tx.Hash().Hex(), nil
}

func buildERC20TransferData(to common.Address, amount *big.Int) []byte {
	// transfer(address,uint256) = 0xa9059cbb
	methodID := common.Hex2Bytes("a9059cbb")
	paddedAddress := common.LeftPadBytes(to.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	return data
}

// BroadcastTransaction 广播交易
func (c *Client) BroadcastTransaction(signedTx string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	txBytes := common.Hex2Bytes(signedTx)
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		return "", err
	}

	if err := c.client.SendTransaction(ctx, tx); err != nil {
		return "", err
	}

	logger.Infof("Transaction broadcast: %s", tx.Hash().Hex())
	return tx.Hash().Hex(), nil
}

// EstimateFee 估算手续费
func (c *Client) EstimateFee(from, to, amount string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return "0", err
	}

	// 假设标准转账 21000 gas
	gasLimit := big.NewInt(21000)
	fee := new(big.Int).Mul(gasPrice, gasLimit)

	return fee.String(), nil
}

// ValidateAddress 验证地址
func (c *Client) ValidateAddress(address string) bool {
	if !common.IsHexAddress(address) {
		return false
	}
	return true
}

// GetRequiredConfirmations 获取所需确认数
func (c *Client) GetRequiredConfirmations() int {
	return c.confirmations
}

// Close 关闭客户端
func (c *Client) Close() {
	c.client.Close()
}

// SubscribeNewBlocks 订阅新区块
func (c *Client) SubscribeNewBlocks(blockChan chan<- *types.Header) (ethereum.Subscription, error) {
	ctx := context.Background()
	return c.client.SubscribeNewHead(ctx, blockChan)
}

// GetBlock 获取区块
func (c *Client) GetBlock(blockNumber uint64) (*blockchain.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	block, err := c.client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return nil, err
	}

	txHashes := make([]string, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		txHashes[i] = tx.Hash().Hex()
	}

	return &blockchain.Block{
		Number:       block.NumberU64(),
		Hash:         block.Hash().Hex(),
		ParentHash:   block.ParentHash().Hex(),
		Timestamp:    int64(block.Time()),
		Transactions: txHashes,
	}, nil
}

// GetLogs 获取日志
func (c *Client) GetLogs(fromBlock, toBlock uint64, addresses []string) ([]types.Log, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addrs := make([]common.Address, len(addresses))
	for i, addr := range addresses {
		addrs[i] = common.HexToAddress(addr)
	}

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: addrs,
	}

	return c.client.FilterLogs(ctx, query)
}

// Ensure Client implements blockchain.Chain
var _ blockchain.Chain = (*Client)(nil)
