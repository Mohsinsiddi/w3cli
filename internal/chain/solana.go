package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// SolanaClient is a minimal JSON-RPC client for Solana.
type SolanaClient struct {
	url    string
	client *http.Client
}

// NewSolanaClient creates a new Solana RPC client.
func NewSolanaClient(url string) *SolanaClient {
	return &SolanaClient{
		url:    url,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// GetBalance returns the SOL balance for a public key address.
func (c *SolanaClient) GetBalance(address string) (*Balance, error) {
	result, err := c.call("getBalance", address)
	if err != nil {
		return nil, err
	}

	// result is {"context":{"slot":N},"value":N}
	raw, _ := json.Marshal(result)
	var resp struct {
		Value uint64 `json:"value"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing solana balance: %w", err)
	}

	// Solana balances are in lamports (1 SOL = 1e9 lamports)
	lamports := new(big.Int).SetUint64(resp.Value)
	sol := solanaLamportsToSOL(lamports)

	return &Balance{
		Wei: lamports,
		ETH: sol, // using ETH field for SOL value
	}, nil
}

// GetSlot returns the current slot (analogous to block number).
func (c *SolanaClient) GetSlot() (uint64, error) {
	result, err := c.call("getSlot")
	if err != nil {
		return 0, err
	}
	switch v := result.(type) {
	case float64:
		return uint64(v), nil
	case json.Number:
		n, _ := v.Int64()
		return uint64(n), nil
	}
	return 0, fmt.Errorf("unexpected slot type: %T", result)
}

// Ping tests the Solana endpoint and returns latency + slot.
func (c *SolanaClient) Ping(ctx context.Context) (time.Duration, uint64, error) {
	start := time.Now()
	slot, err := c.GetSlot()
	latency := time.Since(start)
	return latency, slot, err
}

// --- internal ---

type solanaRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type solanaResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *SolanaClient) call(method string, params ...interface{}) (interface{}, error) {
	body, _ := json.Marshal(solanaRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})

	resp, err := c.client.Post(c.url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("solana RPC request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var rpcResp solanaResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parsing solana response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("solana RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	var result interface{}
	json.Unmarshal(rpcResp.Result, &result)
	return result, nil
}

func solanaLamportsToSOL(lamports *big.Int) string {
	f := new(big.Float).SetInt(lamports)
	f.Quo(f, new(big.Float).SetFloat64(1e9))
	return f.Text('f', 9)
}
