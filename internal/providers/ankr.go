package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// ankrError handles both string and object error formats from Ankr.
type ankrError struct{ msg string }

func (e *ankrError) UnmarshalJSON(b []byte) error {
	// Try object: {"message":"..."}
	var obj struct{ Message string }
	if json.Unmarshal(b, &obj) == nil && obj.Message != "" {
		e.msg = obj.Message
		return nil
	}
	// Try plain string
	var s string
	if json.Unmarshal(b, &s) == nil {
		e.msg = s
		return nil
	}
	e.msg = string(b)
	return nil
}

const ankrEndpoint = "https://rpc.ankr.com/multichain"

// ankrChain maps our chain slugs to Ankr's blockchain identifiers.
var ankrChain = map[string]string{
	"ethereum":  "eth",
	"base":      "base",
	"polygon":   "polygon",
	"arbitrum":  "arbitrum",
	"optimism":  "optimism",
	"zksync":    "zksync_era",
	"scroll":    "scroll",
	"bnb":       "bsc",
	"fantom":    "fantom",
	"linea":     "linea",
	"mantle":    "mantle",
	"avalanche": "avalanche",
	"gnosis":    "gnosis",
	"celo":      "celo",
}

// Ankr uses Ankr's free Advanced API (no key required for basic use).
// With an API key the rate limit and credit allowance are higher.
type Ankr struct {
	chainName string
	apiKey    string // empty = free public endpoint
}

// NewAnkr creates an Ankr provider for the given chain.
// Returns nil if the chain is not supported by Ankr.
func NewAnkr(chainName, apiKey string) *Ankr {
	if _, ok := ankrChain[chainName]; !ok {
		return nil
	}
	return &Ankr{chainName: chainName, apiKey: apiKey}
}

func (a *Ankr) Name() string { return "ankr" }

// ankrReq is the JSON-RPC request body.
type ankrReq struct {
	JSONRPC string    `json:"jsonrpc"`
	Method  string    `json:"method"`
	Params  ankrParam `json:"params"`
	ID      int       `json:"id"`
}

type ankrParam struct {
	Blockchain string `json:"blockchain"`
	Address    string `json:"address"`
	PageSize   int    `json:"pageSize"`
	DescOrder  bool   `json:"descOrder"`
}

// ankrResp is the JSON-RPC response body.
type ankrResp struct {
	Result *ankrResult `json:"result"`
	Error  *ankrError  `json:"error"`
}

type ankrResult struct {
	Transactions []ankrTx `json:"transactions"`
}

type ankrTx struct {
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`       // hex
	Input       string `json:"input"`
	Gas         string `json:"gas"`         // hex
	GasPrice    string `json:"gasPrice"`    // hex
	GasUsed     string `json:"gasUsed"`     // hex
	Timestamp   string `json:"timestamp"`   // hex unix seconds
	Nonce       string `json:"nonce"`       // hex
	BlockNumber string `json:"blockNumber"` // hex
	Status      string `json:"status"`      // "0x1" = success
}

func (a *Ankr) GetTransactions(address string, n int) ([]*chain.Transaction, error) {
	blockchain := ankrChain[a.chainName]

	url := ankrEndpoint
	if a.apiKey != "" {
		url += "/" + a.apiKey
	}

	body, _ := json.Marshal(ankrReq{
		JSONRPC: "2.0",
		Method:  "ankr_getTransactionsByAddress",
		Params: ankrParam{
			Blockchain: blockchain,
			Address:    address,
			PageSize:   n,
			DescOrder:  true,
		},
		ID: 1,
	})

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result ankrResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%s", result.Error.msg)
	}
	if result.Result == nil {
		return nil, fmt.Errorf("empty result")
	}

	var txs []*chain.Transaction
	for _, t := range result.Result.Transactions {
		tx := &chain.Transaction{
			Hash:         t.Hash,
			From:         t.From,
			To:           t.To,
			FunctionName: chain.DecodeMethod(t.Input),
			IsContract:   t.Input != "" && t.Input != "0x",
			Success:      t.Status == "0x1" || t.Status == "1",
		}
		if v, ok := hexBigInt(t.Value); ok {
			tx.Value = v
			tx.ValueETH = chain.WeiToETH(v)
		}
		if g, ok := hexUint64(t.Gas); ok {
			tx.Gas = g
		}
		if gu, ok := hexUint64(t.GasUsed); ok {
			tx.GasUsed = gu
		}
		if gp, ok := hexBigInt(t.GasPrice); ok {
			tx.GasPrice = gp
		}
		if nn, ok := hexUint64(t.Nonce); ok {
			tx.Nonce = nn
		}
		if bn, ok := hexUint64(t.BlockNumber); ok {
			tx.BlockNum = bn
		}
		if ts, ok := hexUint64(t.Timestamp); ok {
			tx.Timestamp = ts
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func hexBigInt(h string) (*big.Int, bool) {
	h = strings.TrimPrefix(h, "0x")
	n, ok := new(big.Int).SetString(h, 16)
	return n, ok
}

func hexUint64(h string) (uint64, bool) {
	n, ok := hexBigInt(h)
	if !ok {
		return 0, false
	}
	return n.Uint64(), true
}
