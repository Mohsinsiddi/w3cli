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

// SUIClient is a minimal JSON-RPC client for SUI.
type SUIClient struct {
	url    string
	client *http.Client
}

// NewSUIClient creates a new SUI RPC client.
func NewSUIClient(url string) *SUIClient {
	return &SUIClient{
		url:    url,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// GetBalance returns the SUI balance for an address.
func (c *SUIClient) GetBalance(address string) (*Balance, error) {
	result, err := c.call("suix_getBalance", address, "0x2::sui::SUI")
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(result)
	var resp struct {
		TotalBalance string `json:"totalBalance"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing sui balance: %w", err)
	}

	// SUI uses MIST (1 SUI = 1e9 MIST)
	mist, ok := new(big.Int).SetString(resp.TotalBalance, 10)
	if !ok {
		mist = big.NewInt(0)
	}
	sui := suiMistToSUI(mist)

	return &Balance{
		Wei: mist,
		ETH: sui,
	}, nil
}

// GetLatestCheckpoint returns the latest checkpoint sequence number.
func (c *SUIClient) GetLatestCheckpoint() (uint64, error) {
	result, err := c.call("sui_getLatestCheckpointSequenceNumber")
	if err != nil {
		return 0, err
	}
	switch v := result.(type) {
	case string:
		n, ok := new(big.Int).SetString(v, 10)
		if !ok {
			return 0, fmt.Errorf("could not parse checkpoint: %s", v)
		}
		return n.Uint64(), nil
	case float64:
		return uint64(v), nil
	}
	return 0, fmt.Errorf("unexpected checkpoint type: %T", result)
}

// Ping tests the SUI endpoint and returns latency + checkpoint.
func (c *SUIClient) Ping(ctx context.Context) (time.Duration, uint64, error) {
	start := time.Now()
	cp, err := c.GetLatestCheckpoint()
	latency := time.Since(start)
	return latency, cp, err
}

// --- internal ---

type suiRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type suiResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *SUIClient) call(method string, params ...interface{}) (interface{}, error) {
	body, _ := json.Marshal(suiRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})

	resp, err := c.client.Post(c.url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("SUI RPC request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var rpcResp suiResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parsing SUI response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("SUI RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	var result interface{}
	json.Unmarshal(rpcResp.Result, &result)
	return result, nil
}

func suiMistToSUI(mist *big.Int) string {
	f := new(big.Float).SetInt(mist)
	f.Quo(f, new(big.Float).SetFloat64(1e9))
	return f.Text('f', 9)
}
