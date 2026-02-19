package providers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// moralisChainID maps chain slugs to Moralis hex chain identifiers.
var moralisChainID = map[string]string{
	"ethereum":  "0x1",
	"base":      "0x2105",
	"polygon":   "0x89",
	"arbitrum":  "0xa4b1",
	"optimism":  "0xa",
	"bnb":       "0x38",
	"avalanche": "0xa86a",
	"fantom":    "0xfa",
	"gnosis":    "0x64",
	"linea":     "0xe708",
	"scroll":    "0x82750",
	"zksync":    "0x144",
	"mantle":    "0x1388",
	"celo":      "0xa4ec",
}

const moralisBaseURL = "https://deep-index.moralis.io/api/v2.2"

// Moralis is a provider backed by the Moralis Deep Index API.
// It requires an API key and is nil-guarded: NewMoralis returns nil if no key is set.
type Moralis struct {
	chainName string
	hexChain  string
	apiKey    string
	baseURL   string // defaults to moralisBaseURL; overridable in tests
}

// NewMoralis creates a Moralis provider.
// Returns nil if apiKey is empty or the chain is not supported.
func NewMoralis(chainName, apiKey string) *Moralis {
	if apiKey == "" {
		return nil
	}
	hex, ok := moralisChainID[chainName]
	if !ok {
		return nil
	}
	return &Moralis{chainName: chainName, hexChain: hex, apiKey: apiKey, baseURL: moralisBaseURL}
}

func (m *Moralis) Name() string { return "moralis" }

// moralisTx is a single transaction from the Moralis API.
type moralisTx struct {
	Hash           string `json:"hash"`
	FromAddress    string `json:"from_address"`
	ToAddress      string `json:"to_address"`
	Value          string `json:"value"`           // decimal wei
	Gas            string `json:"gas"`             // decimal
	GasPrice       string `json:"gas_price"`       // decimal wei
	GasUsed        string `json:"receipt_gas_used"` // decimal
	Nonce          string `json:"nonce"`           // decimal
	BlockNumber    string `json:"block_number"`    // decimal
	ReceiptStatus  string `json:"receipt_status"`  // "1" = success
	BlockTimestamp string `json:"block_timestamp"` // ISO 8601
	Input          string `json:"input"`
}

type moralisResp struct {
	Result []moralisTx `json:"result"`
}

func (m *Moralis) GetTransactions(address string, n int) ([]*chain.Transaction, error) {
	url := fmt.Sprintf("%s/%s/transactions?chain=%s&limit=%d", m.baseURL, address, m.hexChain, n)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-API-Key", m.apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result moralisResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	txs := make([]*chain.Transaction, 0, len(result.Result))
	for _, t := range result.Result {
		tx := &chain.Transaction{
			Hash:         t.Hash,
			From:         t.FromAddress,
			To:           t.ToAddress,
			FunctionName: chain.DecodeMethod(t.Input),
			IsContract:   t.Input != "" && t.Input != "0x",
			Success:      t.ReceiptStatus == "1",
		}

		if v, ok := decimalBigInt(t.Value); ok {
			tx.Value = v
			tx.ValueETH = chain.WeiToETH(v)
		}
		if g, ok := decimalUint64(t.Gas); ok {
			tx.Gas = g
		}
		if gu, ok := decimalUint64(t.GasUsed); ok {
			tx.GasUsed = gu
		}
		if gp, ok := decimalBigInt(t.GasPrice); ok {
			tx.GasPrice = gp
		}
		if nn, ok := decimalUint64(t.Nonce); ok {
			tx.Nonce = nn
		}
		if bn, ok := decimalUint64(t.BlockNumber); ok {
			tx.BlockNum = bn
		}

		txs = append(txs, tx)
	}
	return txs, nil
}

// decimalBigInt parses a base-10 decimal string into a big.Int.
func decimalBigInt(s string) (*big.Int, bool) {
	n, ok := new(big.Int).SetString(s, 10)
	return n, ok
}

// decimalUint64 parses a base-10 decimal string into a uint64.
func decimalUint64(s string) (uint64, bool) {
	n, ok := decimalBigInt(s)
	if !ok {
		return 0, false
	}
	return n.Uint64(), true
}
