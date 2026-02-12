package tron

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"custodial-wallet/internal/blockchain"
)

// Client Tron 简单 HTTP 客户端（使用 TronGrid/TronFullNode API）
type Client struct {
	url           string
	apiKey        string
	confirmations int
	httpClient    *http.Client
}

func NewClient(rpcURL, apiKey, network string, confirmations int) (*Client, error) {
	return &Client{url: rpcURL, apiKey: apiKey, confirmations: confirmations, httpClient: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *Client) call(path string, method string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, strings.TrimRight(c.url, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (c *Client) GetName() string { return "tron" }

// GetBalance 获取TRX余额
func (c *Client) GetBalance(address string) (string, error) {
	path := fmt.Sprintf("/wallet/getaccount?address=%s", address)
	b, err := c.call(path, "GET", nil)
	if err != nil {
		return "0", err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return "0", err
	}
	if balance, ok := m["balance"].(float64); ok {
		return fmt.Sprintf("%f", balance/1e6), nil // TRX单位为sun
	}
	return "0", nil
}

func (c *Client) GetTokenBalance(address, contractAddress string) (string, error) {
	// Tron token balance requires calling contract via /wallet/getaccount or /wallet/getassetissue
	return "0", nil
}

func (c *Client) GetTransaction(txHash string) (*blockchain.TransactionInfo, error) {
	// Tron txHash is hex; use /wallet/gettransactionbyid
	path := fmt.Sprintf("/wallet/gettransactionbyid?value=%s", txHash)
	b, err := c.call(path, "GET", nil)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	info := &blockchain.TransactionInfo{TxHash: txHash}
	// parse raw_data -> contract to find transfer
	if raw, ok := m["raw_data"].(map[string]interface{}); ok {
		if contracts, ok := raw["contract"].([]interface{}); ok && len(contracts) > 0 {
			c0 := contracts[0].(map[string]interface{})
			if param, ok := c0["parameter"].(map[string]interface{}); ok {
				if value, ok := param["value"].(map[string]interface{}); ok {
					if to, ok := value["to_address"].(string); ok {
						// tron base58 decode to hex not implemented here; keep as given
						info.To = to
					}
					if owner, ok := value["owner_address"].(string); ok {
						info.From = owner
					}
				}
			}
		}
	}
	return info, nil
}

func (c *Client) GetBlockNumber() (uint64, error) {
	b, err := c.call("/wallet/getnowblock", "GET", nil)
	if err != nil {
		return 0, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return 0, err
	}
	if raw, ok := m["block_header"].(map[string]interface{}); ok {
		if number, ok := raw["raw_data"].(map[string]interface{}); ok {
			if bn, ok := number["number"].(float64); ok {
				return uint64(bn), nil
			}
		}
	}
	return 0, nil
}

func (c *Client) BuildTransaction(from, to, amount, contractAddress string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *Client) BroadcastTransaction(signedTx string) (string, error) {
	// sendrawtransaction
	b, err := c.call("/wallet/broadcasthex", "POST", []byte(signedTx))
	if err != nil {
		return "", err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}
	if txid, ok := m["txid"].(string); ok {
		return txid, nil
	}
	return "", fmt.Errorf("no txid")
}

func (c *Client) EstimateFee(from, to, amount string) (string, error) {
	return "0", nil
}

func (c *Client) ValidateAddress(address string) bool {
	// naive check
	return address != ""
}

func (c *Client) GetRequiredConfirmations() int { return c.confirmations }

// For tron, implement GetBlock to return transaction list
func (c *Client) GetBlock(blockNumber uint64) (*blockchain.Block, error) {
	// /wallet/getblockbynum
	path := fmt.Sprintf("/wallet/getblockbynum?num=%d", blockNumber)
	b, err := c.call(path, "GET", nil)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	blk := &blockchain.Block{Number: blockNumber}
	if raw, ok := m["block_header"].(map[string]interface{}); ok {
		if rawData, ok := raw["raw_data"].(map[string]interface{}); ok {
			if txs, ok := rawData["transactions"].([]interface{}); ok {
				for _, t := range txs {
					if tx, ok := t.(map[string]interface{}); ok {
						if txid, ok := tx["txID"].(string); ok {
							blk.Transactions = append(blk.Transactions, txid)
						}
					}
				}
			}
		}
	}
	return blk, nil
}

// Tron does not support logs like ethereum; but we used GetBlock + GetTransaction

// Ensure Client implements blockchain.Chain
var _ blockchain.Chain = (*Client)(nil)
