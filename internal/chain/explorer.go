package chain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// explorerResponse is the raw Etherscan/BlockScout-compatible API envelope.
// Result is kept as RawMessage because a failed call returns a plain string
// (e.g. "NOTOK" or an error message) while a successful call returns a JSON array.
type explorerResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type explorerTx struct {
	Hash             string `json:"hash"`
	BlockNumber      string `json:"blockNumber"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	GasUsed          string `json:"gasUsed"`
	Nonce            string `json:"nonce"`
	IsError          string `json:"isError"`
	TxReceiptStatus  string `json:"txreceipt_status"`
	Input            string `json:"input"`
	TimeStamp        string `json:"timeStamp"`
	Confirmations    string `json:"confirmations"`
	ContractAddress  string `json:"contractAddress"`
}

// knownMethods maps 4-byte selectors to human-readable names.
var knownMethods = map[string]string{
	"0xa9059cbb": "transfer",
	"0x095ea7b3": "approve",
	"0x23b872dd": "transferFrom",
	"0x39509351": "increaseAllowance",
	"0xa457c2d7": "decreaseAllowance",
	"0x7ff36ab5": "swapExactETHForTokens",
	"0x18cbafe5": "swapExactTokensForETH",
	"0x38ed1739": "swapExactTokensForTokens",
	"0xfb3bdb41": "swapETHForExactTokens",
	"0x8803dbee": "swapTokensForExactTokens",
	"0x5c11d795": "swapExactTokensForTokensSupportingFeeOnTransferTokens",
	"0xb6f9de95": "swapExactETHForTokensSupportingFeeOnTransferTokens",
	"0x791ac947": "swapExactTokensForETHSupportingFeeOnTransferTokens",
	"0x414bf389": "exactInputSingle",   // Uniswap V3
	"0xdb3e2198": "exactOutputSingle",
	"0xac9650d8": "multicall",
	"0x5ae401dc": "multicall",           // Uniswap V3 multicall
	"0x12aa3caf": "swap",                // 1inch v5
	"0x0502b1c5": "unoswap",             // 1inch
	"0x5f575529": "swap",                // 0x swap
	"0xd9627aa4": "sellToUniswap",       // 0x
	"0xfe2298b9": "bridgeTokensTo",      // cross-chain bridge
	"0xe8e33700": "addLiquidity",
	"0xf305d719": "addLiquidityETH",
	"0xbaa2abde": "removeLiquidity",
	"0x02751cec": "removeLiquidityETH",
	"0x6a627842": "mint",
	"0x42966c68": "burn",
	"0x4e71d92d": "claim",
	"0x2e7ba6ef": "claim",
	"0x379607f5": "claim",
	"0x3d18b912": "getReward",
	"0xe9fad8ee": "exit",
	"0xa694fc3a": "stake",
	"0x2e1a7d4d": "withdraw",
	"0x51cff8d9": "withdraw",
	"0xd0e30db0": "deposit",
	"0xb6b55f25": "deposit",
	"0x4e487b71": "Panic",
	"0x08c379a0": "Error",
	"0x70a08231": "balanceOf",
	"0x313ce567": "decimals",
	"0x06fdde03": "name",
	"0x95d89b41": "symbol",
	"0x18160ddd": "totalSupply",
}

// DecodeMethod is the exported form of decodeMethod for use by other packages.
func DecodeMethod(input string) string { return decodeMethod(input) }

// decodeMethod returns a human-readable method name from calldata input.
// Returns "transfer" for plain ETH sends, or the 4-byte hex if unknown.
func decodeMethod(input string) string {
	if input == "" || input == "0x" {
		return "transfer"
	}
	// Strip 0x prefix and grab first 4 bytes (8 hex chars).
	clean := strings.TrimPrefix(input, "0x")
	if len(clean) < 8 {
		return "call"
	}
	selector := "0x" + strings.ToLower(clean[:8])
	if name, ok := knownMethods[selector]; ok {
		return name
	}
	return selector[:10] // show "0xabcd1234" (10 chars)
}

// GetTransactionsFromExplorer fetches recent transactions using an
// Etherscan/BlockScout-compatible block explorer API.
// apiKey may be empty (free BlockScout tier) or a paid key (Etherscan, BlockScout Pro).
func GetTransactionsFromExplorer(apiURL, address string, n int, apiKey string) ([]*Transaction, error) {
	// Use "&" when apiURL already contains a "?" (e.g. Etherscan V2 includes ?chainid=X).
	sep := "?"
	if strings.Contains(apiURL, "?") {
		sep = "&"
	}
	url := fmt.Sprintf(
		"%s%smodule=account&action=txlist&address=%s&startblock=0&endblock=99999999&page=1&offset=%d&sort=desc",
		apiURL, sep, address, n,
	)
	if apiKey != "" {
		url += "&apikey=" + apiKey
	}

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
		var msg string
		if err := json.Unmarshal(envelope.Result, &msg); err == nil && msg != "" {
			return nil, fmt.Errorf("explorer API: %s", msg)
		}
		return nil, fmt.Errorf("explorer API: %s", envelope.Message)
	}

	var raw []explorerTx
	if err := json.Unmarshal(envelope.Result, &raw); err != nil {
		return nil, fmt.Errorf("parsing explorer tx list: %w", err)
	}

	var txs []*Transaction
	for _, et := range raw {
		tx := &Transaction{
			Hash:         et.Hash,
			From:         et.From,
			To:           et.To,
			Success:      et.TxReceiptStatus == "1" && et.IsError == "0",
			FunctionName: decodeMethod(et.Input),
			IsContract:   et.Input != "" && et.Input != "0x",
		}
		// Contract creation: To is empty, ContractAddress is set.
		if et.To == "" && et.ContractAddress != "" {
			tx.To = et.ContractAddress
			tx.FunctionName = "deploy"
			tx.IsContract = true
		}

		// Parse numeric fields (explorer returns decimal strings).
		if v, ok := new(big.Int).SetString(et.Value, 10); ok {
			tx.Value = v
			tx.ValueETH = weiToETH(v)
		}
		if g, ok := new(big.Int).SetString(et.Gas, 10); ok {
			tx.Gas = g.Uint64()
		}
		if gu, ok := new(big.Int).SetString(et.GasUsed, 10); ok {
			tx.GasUsed = gu.Uint64()
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
		if ts, ok := new(big.Int).SetString(et.TimeStamp, 10); ok {
			tx.Timestamp = ts.Uint64()
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

// contractSourceResult is the inner object from getsourcecode.
type contractSourceResult struct {
	ContractName string `json:"ContractName"`
}

// contractSourceResponse wraps the getsourcecode API call.
type contractSourceResponse struct {
	Status  string                 `json:"status"`
	Result  []contractSourceResult `json:"result"`
}

// FetchContractNames queries the BlockScout-compatible API for the contract
// name of each unique address in parallel.  Unknown or EOA addresses are
// omitted from the returned map.  Never returns an error — failures per address
// are silently skipped so the caller always gets a partial (or empty) result.
// apiKey may be empty for free tier usage.
func FetchContractNames(apiURL string, addresses []string, apiKey string) map[string]string {
	// Deduplicate.
	seen := make(map[string]struct{})
	var unique []string
	for _, a := range addresses {
		lo := strings.ToLower(a)
		if _, ok := seen[lo]; !ok {
			seen[lo] = struct{}{}
			unique = append(unique, a)
		}
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		names  = make(map[string]string)
		client = &http.Client{Timeout: 6 * time.Second}
	)

	for _, addr := range unique {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			url := fmt.Sprintf(
				"%s?module=contract&action=getsourcecode&address=%s",
				apiURL, addr,
			)
			if apiKey != "" {
				url += "&apikey=" + apiKey
			}
			resp, err := client.Get(url)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			var result contractSourceResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return
			}
			if result.Status != "1" || len(result.Result) == 0 {
				return
			}
			name := result.Result[0].ContractName
			if name == "" {
				return
			}
			mu.Lock()
			names[strings.ToLower(addr)] = name
			mu.Unlock()
		}(addr)
	}

	wg.Wait()
	return names
}
