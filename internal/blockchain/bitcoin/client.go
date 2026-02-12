package bitcoin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"custodial-wallet/internal/blockchain"
)

// Client 比特币 RPC 客户端（JSON-RPC）
type Client struct {
	url           string
	user          string
	pass          string
	confirmations int
	httpClient    *http.Client
}

// NewClient 创建比特币客户端
func NewClient(rpcURL, rpcUser, rpcPass, network string, confirmations int) (*Client, error) {
	c := &Client{
		url:           rpcURL,
		user:          rpcUser,
		pass:          rpcPass,
		confirmations: confirmations,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}
	return c, nil
}

type rpcReq struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResp struct {
	Result json.RawMessage `json:"result"`
	Error  interface{}     `json:"error"`
}

func (c *Client) callRPC(method string, params []interface{}) (json.RawMessage, error) {
	reqBody := rpcReq{Jsonrpc: "1.0", ID: "go-client", Method: method, Params: params}
	b, _ := json.Marshal(reqBody)
	request, _ := http.NewRequest("POST", c.url, bytes.NewReader(b))
	request.Header.Set("Content-Type", "application/json")
	if c.user != "" {
		request.SetBasicAuth(c.user, c.pass)
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var r rpcResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return r.Result, nil
}

// GetName 获取链名称
func (c *Client) GetName() string { return "bitcoin" }

// GetBalance TODO: implement
func (c *Client) GetBalance(address string) (string, error) {
	// Not implemented: requires address index or wallet.
	return "0", nil
}

// GetTokenBalance Bitcoin 无 token
func (c *Client) GetTokenBalance(address, contractAddress string) (string, error) {
	return "0", nil
}

// GetTransaction 获取交易信息
func (c *Client) GetTransaction(txHash string) (*blockchain.TransactionInfo, error) {
	// Use getrawtransaction with verbose=true
	res, err := c.callRPC("getrawtransaction", []interface{}{txHash, true})
	if err != nil {
		return nil, err
	}
	var txRaw map[string]interface{}
	if err := json.Unmarshal(res, &txRaw); err != nil {
		return nil, err
	}
	info := &blockchain.TransactionInfo{TxHash: txHash}
	// parse vout to find to address and amount (sum of outputs to same address not aggregated here)
	if vouts, ok := txRaw["vout"].([]interface{}); ok && len(vouts) > 0 {
		// pick first vout with addresses
		for _, v := range vouts {
			m, _ := v.(map[string]interface{})
			if scriptPubKey, ok := m["scriptPubKey"].(map[string]interface{}); ok {
				if addrs, ok := scriptPubKey["addresses"].([]interface{}); ok && len(addrs) > 0 {
					addr := fmt.Sprintf("%v", addrs[0])
					info.To = addr
					if val, ok := m["value"].(float64); ok {
						info.Amount = fmt.Sprintf("%f", val)
					}
					break
				}
			}
		}
	}
	// blockhash and block number
	if bh, ok := txRaw["blockhash"].(string); ok && bh != "" {
		// getblock to fetch height
		res2, err := c.callRPC("getblock", []interface{}{bh})
		if err == nil {
			var b map[string]interface{}
			if err := json.Unmarshal(res2, &b); err == nil {
				if height, ok := b["height"].(float64); ok {
					info.BlockNumber = uint64(height)
				}
			}
		}
	}
	return info, nil
}

// GetBlockNumber 获取最新区块号
func (c *Client) GetBlockNumber() (uint64, error) {
	res, err := c.callRPC("getblockcount", nil)
	if err != nil {
		return 0, err
	}
	var height float64
	if err := json.Unmarshal(res, &height); err != nil {
		return 0, err
	}
	return uint64(height), nil
}

// BuildTransaction 构建交易（简化）
func (c *Client) BuildTransaction(from, to, amount, contractAddress string) (string, error) {
	// Building raw transaction is out of scope here.
	return "", fmt.Errorf("not implemented")
}

// BroadcastTransaction 广播交易（使用 sendrawtransaction）
func (c *Client) BroadcastTransaction(signedTx string) (string, error) {
	res, err := c.callRPC("sendrawtransaction", []interface{}{signedTx})
	if err != nil {
		return "", err
	}
	var txid string
	if err := json.Unmarshal(res, &txid); err != nil {
		return "", err
	}
	return txid, nil
}

// EstimateFee 简化实现
func (c *Client) EstimateFee(from, to, amount string) (string, error) {
	// Use fee rates from estimatesmartfee
	res, err := c.callRPC("estimatesmartfee", []interface{}{6})
	if err != nil {
		return "0", err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(res, &m); err != nil {
		return "0", err
	}
	if feerate, ok := m["feerate"].(float64); ok {
		return fmt.Sprintf("%f", feerate), nil
	}
	return "0", nil
}

// ValidateAddress 验证地址（简化）
func (c *Client) ValidateAddress(address string) bool {
	// Basic length check
	return address != ""
}

// GetRequiredConfirmations 获取所需确认数
func (c *Client) GetRequiredConfirmations() int { return c.confirmations }

// GetBlock 获取区块（实现 blockchain.Block）
func (c *Client) GetBlock(blockNumber uint64) (*blockchain.Block, error) {
	// getblockhash then getblock with verbosity=1
	res, err := c.callRPC("getblockhash", []interface{}{blockNumber})
	if err != nil {
		return nil, err
	}
	var bh string
	if err := json.Unmarshal(res, &bh); err != nil {
		return nil, err
	}
	res2, err := c.callRPC("getblock", []interface{}{bh, 1})
	if err != nil {
		return nil, err
	}
	var b map[string]interface{}
	if err := json.Unmarshal(res2, &b); err != nil {
		return nil, err
	}
	blk := &blockchain.Block{Number: blockNumber}
	if txs, ok := b["tx"].([]interface{}); ok {
		for _, t := range txs {
			if s, ok := t.(string); ok {
				blk.Transactions = append(blk.Transactions, s)
			}
		}
	}
	if hash, ok := b["hash"].(string); ok {
		blk.Hash = hash
	}
	if parent, ok := b["previousblockhash"].(string); ok {
		blk.ParentHash = parent
	}
	return blk, nil
}

// Ensure Client implements blockchain.Chain
var _ blockchain.Chain = (*Client)(nil)
