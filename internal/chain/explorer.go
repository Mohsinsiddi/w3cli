package chain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

// explorerResponse is the raw Etherscan/BlockScout-compatible API envelope.
// Result is kept as RawMessage because a failed call returns a plain string
// (e.g. "NOTOK" or an error message) while a successful call returns a JSON
// array.  We decode it in two steps to avoid a type-mismatch error.
type explorerResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type explorerTx struct {
	Hash        string `json:"hash"`
	BlockNumber string `json:"blockNumber"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	Gas         string `json:"gas"`
	GasPrice    string `json:"gasPrice"`
	Nonce       string `json:"nonce"`
	IsError     string `json:"isError"`
}

// GetTransactionsFromExplorer fetches recent transactions using an
// Etherscan/BlockScout-compatible block explorer API.
// apiURL should be the base API endpoint, e.g. "https://base.blockscout.com/api".
func GetTransactionsFromExplorer(apiURL, address string, n int) ([]*Transaction, error) {
	url := fmt.Sprintf(
		"%s?module=account&action=txlist&address=%s&startblock=0&endblock=99999999&page=1&offset=%d&sort=desc",
		apiURL, address, n,
	)

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("explorer request failed: %w", err)
	}
	defer resp.Body.Close()

	var envelope explorerResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("parsing explorer response: %w", err)
	}

	// Non-success: result may be a plain error string, not an array.
	if envelope.Status != "1" {
		// Extract string message if available.
		var msg string
		if err := json.Unmarshal(envelope.Result, &msg); err == nil && msg != "" {
			return nil, fmt.Errorf("explorer API: %s", msg)
		}
		return nil, fmt.Errorf("explorer API: %s", envelope.Message)
	}

	// Decode the result array.
	var raw []explorerTx
	if err := json.Unmarshal(envelope.Result, &raw); err != nil {
		return nil, fmt.Errorf("parsing explorer tx list: %w", err)
	}

	var txs []*Transaction
	for _, et := range raw {
		tx := &Transaction{
			Hash: et.Hash,
			From: et.From,
			To:   et.To,
		}
		// Explorer returns decimal strings, not hex.
		if v, ok := new(big.Int).SetString(et.Value, 10); ok {
			tx.Value = v
			tx.ValueETH = weiToETH(v)
		}
		if g, ok := new(big.Int).SetString(et.Gas, 10); ok {
			tx.Gas = g.Uint64()
		}
		if gp, ok := new(big.Int).SetString(et.GasPrice, 10); ok {
			tx.GasPrice = gp
		}
		if nonce, ok := new(big.Int).SetString(et.Nonce, 10); ok {
			tx.Nonce = nonce.Uint64()
		}
		if bn, ok := new(big.Int).SetString(et.BlockNumber, 10); ok {
			tx.BlockNum = bn.Uint64()
		}
		txs = append(txs, tx)
	}
	return txs, nil
}
