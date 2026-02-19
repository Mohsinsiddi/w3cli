package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// alchemyNetwork maps chain slugs to Alchemy network identifiers.
var alchemyNetwork = map[string]string{
	"ethereum": "eth-mainnet",
	"polygon":  "polygon-mainnet",
	"arbitrum": "arb-mainnet",
	"optimism": "opt-mainnet",
	"base":     "base-mainnet",
}

// Alchemy is a provider backed by the Alchemy Asset Transfers API.
// It requires an API key and is nil-guarded: NewAlchemy returns nil if no key is set.
type Alchemy struct {
	network string
	apiKey  string
	baseURL string // if non-empty, overrides the default constructed URL (for tests)
}

// NewAlchemy creates an Alchemy provider.
// Returns nil if apiKey is empty or the chain is not supported.
func NewAlchemy(chainName, apiKey string) *Alchemy {
	if apiKey == "" {
		return nil
	}
	net, ok := alchemyNetwork[chainName]
	if !ok {
		return nil
	}
	return &Alchemy{network: net, apiKey: apiKey}
}

func (a *Alchemy) Name() string { return "alchemy" }

// alchemyReq is the JSON-RPC request for alchemy_getAssetTransfers.
type alchemyReq struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []alchemyParam `json:"params"`
	ID      int           `json:"id"`
}

type alchemyParam struct {
	FromAddress string   `json:"fromAddress,omitempty"`
	ToAddress   string   `json:"toAddress,omitempty"`
	Category    []string `json:"category"`
	WithMetadata bool    `json:"withMetadata"`
	MaxCount    string   `json:"maxCount"`
	Order       string   `json:"order"`
}

type alchemyResp struct {
	Result *alchemyResult `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type alchemyResult struct {
	Transfers []alchemyTransfer `json:"transfers"`
}

type alchemyTransfer struct {
	Hash      string  `json:"hash"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	Value     float64 `json:"value"` // ETH (not wei)
	BlockNum  string  `json:"blockNum"`
	Metadata  *alchemyMeta `json:"metadata"`
}

type alchemyMeta struct {
	BlockTimestamp string `json:"blockTimestamp"`
}

func (a *Alchemy) GetTransactions(address string, n int) ([]*chain.Transaction, error) {
	baseURL := a.baseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.g.alchemy.com/v2/%s", a.network, a.apiKey)
	}

	maxCount := fmt.Sprintf("0x%x", n)
	param := alchemyParam{
		Category:     []string{"external"},
		WithMetadata: true,
		MaxCount:     maxCount,
		Order:        "desc",
	}

	// Fetch transfers FROM the address.
	fromParam := param
	fromParam.FromAddress = address
	fromTxs, err := a.fetchTransfers(baseURL, fromParam)
	if err != nil {
		return nil, err
	}

	// Fetch transfers TO the address.
	toParam := param
	toParam.ToAddress = address
	toTxs, err := a.fetchTransfers(baseURL, toParam)
	if err != nil {
		return nil, err
	}

	// Merge and dedup by hash.
	seen := make(map[string]struct{})
	merged := make([]*chain.Transaction, 0, len(fromTxs)+len(toTxs))
	for _, tx := range append(fromTxs, toTxs...) {
		if _, dup := seen[tx.Hash]; dup {
			continue
		}
		seen[tx.Hash] = struct{}{}
		merged = append(merged, tx)
	}

	// Sort by block number descending.
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].BlockNum > merged[j].BlockNum
	})

	// Truncate to n.
	if len(merged) > n {
		merged = merged[:n]
	}

	return merged, nil
}

func (a *Alchemy) fetchTransfers(url string, param alchemyParam) ([]*chain.Transaction, error) {
	reqBody, _ := json.Marshal(alchemyReq{
		JSONRPC: "2.0",
		Method:  "alchemy_getAssetTransfers",
		Params:  []alchemyParam{param},
		ID:      1,
	})

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result alchemyResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%s", result.Error.Message)
	}
	if result.Result == nil {
		return nil, fmt.Errorf("empty result")
	}

	txs := make([]*chain.Transaction, 0, len(result.Result.Transfers))
	for _, t := range result.Result.Transfers {
		// Convert float ETH → wei big.Int.
		weiFloat := new(big.Float).SetFloat64(t.Value)
		weiFloat.Mul(weiFloat, new(big.Float).SetFloat64(1e18))
		weiInt, _ := weiFloat.Int(nil)

		tx := &chain.Transaction{
			Hash:     t.Hash,
			From:     t.From,
			To:       t.To,
			Value:    weiInt,
			ValueETH: chain.WeiToETH(weiInt),
			Success:  true, // Alchemy asset transfers are confirmed
		}

		// Parse block number (hex string).
		if bn, ok := hexBigInt(t.BlockNum); ok {
			tx.BlockNum = bn.Uint64()
		}

		txs = append(txs, tx)
	}
	return txs, nil
}
