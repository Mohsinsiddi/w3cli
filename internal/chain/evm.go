package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// EVMClient is a minimal JSON-RPC client for EVM chains.
type EVMClient struct {
	url    string
	client *http.Client
}

// Balance holds a native balance result.
type Balance struct {
	Wei       *big.Int
	ETH       string
	USD       float64
}

// TokenBalance holds an ERC-20 token balance result.
type TokenBalance struct {
	Raw       *big.Int
	Formatted string
	Decimals  int
}

// Transaction holds a simplified transaction record.
type Transaction struct {
	Hash         string
	From         string
	To           string
	Value        *big.Int
	ValueETH     string
	Gas          uint64
	GasUsed      uint64
	GasPrice     *big.Int
	Nonce        uint64
	BlockNum     uint64
	Timestamp    uint64
	Success      bool   // true = confirmed success, false = failed/unknown
	FunctionName string // decoded method name, e.g. "transfer", "swap", "0xa9059cbb"
	IsContract   bool   // true when input data is present (contract call)
}

// NewEVMClient creates a new EVM JSON-RPC client pointed at url.
func NewEVMClient(url string) *EVMClient {
	return &EVMClient{
		url: url,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// GetBalance returns the native balance (in ETH string) for an address.
func (c *EVMClient) GetBalance(address string) (*Balance, error) {
	result, err := c.call("eth_getBalance", address, "latest")
	if err != nil {
		return nil, err
	}

	hexStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	wei, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("could not parse balance hex: %s", hexStr)
	}

	eth := weiToETH(wei)
	return &Balance{
		Wei: wei,
		ETH: eth,
	}, nil
}

// GetTokenBalance returns an ERC-20 token balance.
func (c *EVMClient) GetTokenBalance(tokenAddr, walletAddr string, decimals int) (*TokenBalance, error) {
	// balanceOf(address) selector = 0x70a08231
	data := "0x70a08231" + fmt.Sprintf("%064s", strings.TrimPrefix(walletAddr, "0x"))

	result, err := c.call("eth_call", map[string]string{
		"to":   tokenAddr,
		"data": data,
	}, "latest")
	if err != nil {
		return nil, err
	}

	hexStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result: %T", result)
	}

	raw, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("could not parse token balance: %s", hexStr)
	}

	formatted := formatToken(raw, decimals)
	return &TokenBalance{
		Raw:       raw,
		Formatted: formatted,
		Decimals:  decimals,
	}, nil
}

// GetBlockNumber returns the latest block number.
func (c *EVMClient) GetBlockNumber() (uint64, error) {
	result, err := c.call("eth_blockNumber")
	if err != nil {
		return 0, err
	}

	hexStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected result: %T", result)
	}

	n, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return 0, fmt.Errorf("could not parse block number: %s", hexStr)
	}

	return n.Uint64(), nil
}

// GetRecentTransactions returns the last n transactions involving address
// by scanning the most recent 200 blocks in parallel.
func (c *EVMClient) GetRecentTransactions(address string, n int) ([]*Transaction, error) {
	latest, err := c.GetBlockNumber()
	if err != nil {
		return nil, err
	}

	addrLower := strings.ToLower(address)
	const scanBlocks = 200

	type blockResult struct {
		num uint64
		txs []*Transaction
	}

	results := make(chan blockResult, scanBlocks)
	var wg sync.WaitGroup

	for i := 0; i < scanBlocks; i++ {
		blockNum := latest - uint64(i)
		wg.Add(1)
		go func(num uint64) {
			defer wg.Done()
			txList, err := c.getBlock(num)
			if err != nil {
				results <- blockResult{num: num}
				return
			}
			var matching []*Transaction
			for _, t := range txList {
				if strings.EqualFold(t.From, addrLower) || strings.EqualFold(t.To, addrLower) {
					matching = append(matching, t)
				}
			}
			results <- blockResult{num: num, txs: matching}
		}(blockNum)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	all := make([]blockResult, 0, scanBlocks)
	for r := range results {
		all = append(all, r)
	}

	// Sort newest-first (highest block number first).
	sort.Slice(all, func(i, j int) bool { return all[i].num > all[j].num })

	var txs []*Transaction
	for _, r := range all {
		txs = append(txs, r.txs...)
		if len(txs) >= n {
			txs = txs[:n]
			break
		}
	}

	return txs, nil
}

// SendRawTransaction broadcasts a signed raw transaction.
func (c *EVMClient) SendRawTransaction(rawTx string) (string, error) {
	result, err := c.call("eth_sendRawTransaction", rawTx)
	if err != nil {
		return "", err
	}
	hash, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result: %T", result)
	}
	return hash, nil
}

// EstimateGas estimates gas for a transaction.
func (c *EVMClient) EstimateGas(from, to, data string, value *big.Int) (uint64, error) {
	params := map[string]string{
		"from": from,
		"to":   to,
	}
	if data != "" {
		params["data"] = data
	}
	if value != nil && value.Sign() > 0 {
		params["value"] = "0x" + value.Text(16)
	}

	result, err := c.call("eth_estimateGas", params, "latest")
	if err != nil {
		return 0, err
	}

	hexStr, ok := result.(string)
	if !ok {
		return 21000, nil
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return 21000, nil
	}
	return n.Uint64(), nil
}

// GasPrice returns the current gas price.
func (c *EVMClient) GasPrice() (*big.Int, error) {
	result, err := c.call("eth_gasPrice")
	if err != nil {
		return nil, err
	}
	hexStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result: %T", result)
	}
	gp, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("could not parse gas price: %s", hexStr)
	}
	return gp, nil
}

// ChainID returns the chain's ID.
func (c *EVMClient) ChainID() (int64, error) {
	result, err := c.call("eth_chainId")
	if err != nil {
		return 0, err
	}
	hexStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected result: %T", result)
	}
	id, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return 0, fmt.Errorf("could not parse chain id: %s", hexStr)
	}
	return id.Int64(), nil
}

// GetTransactionByHash returns a transaction by hash.
func (c *EVMClient) GetTransactionByHash(hash string) (*Transaction, error) {
	result, err := c.call("eth_getTransactionByHash", hash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("transaction not found: %s", hash)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var rt rawTx
	if err := json.Unmarshal(raw, &rt); err != nil {
		return nil, err
	}
	return rt.toTx(), nil
}

// --- internal JSON-RPC plumbing ---

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *EVMClient) call(method string, params ...interface{}) (interface{}, error) {
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Post(c.url, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	var result interface{}
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, fmt.Errorf("parsing result: %w", err)
	}

	return result, nil
}

type rawTx struct {
	Hash      string `json:"hash"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Gas       string `json:"gas"`
	GasPrice  string `json:"gasPrice"`
	Nonce     string `json:"nonce"`
	BlockNum  string `json:"blockNumber"`
}

func (rt *rawTx) toTx() *Transaction {
	tx := &Transaction{
		Hash: rt.Hash,
		From: rt.From,
		To:   rt.To,
	}
	if v, ok := parseBigHex(rt.Value); ok {
		tx.Value = v
		tx.ValueETH = weiToETH(v)
	}
	if g, ok := parseBigHex(rt.Gas); ok {
		tx.Gas = g.Uint64()
	}
	if gp, ok := parseBigHex(rt.GasPrice); ok {
		tx.GasPrice = gp
	}
	if n, ok := parseBigHex(rt.Nonce); ok {
		tx.Nonce = n.Uint64()
	}
	if bn, ok := parseBigHex(rt.BlockNum); ok {
		tx.BlockNum = bn.Uint64()
	}
	return tx
}

type rawBlock struct {
	Transactions []json.RawMessage `json:"transactions"`
}

func (c *EVMClient) getBlock(num uint64) ([]*Transaction, error) {
	hexNum := fmt.Sprintf("0x%x", num)
	result, err := c.call("eth_getBlockByNumber", hexNum, true)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	raw, _ := json.Marshal(result)
	var block rawBlock
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, err
	}

	var txs []*Transaction
	for _, txRaw := range block.Transactions {
		var rt rawTx
		if err := json.Unmarshal(txRaw, &rt); err != nil {
			continue
		}
		txs = append(txs, rt.toTx())
	}
	return txs, nil
}

// --- math helpers ---

var eth1 = new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

// WeiToETH converts a wei amount to an ETH decimal string.
func WeiToETH(wei *big.Int) string { return weiToETH(wei) }

func weiToETH(wei *big.Int) string {
	f := new(big.Float).SetInt(wei)
	f.Quo(f, eth1)
	return f.Text('f', 18)
}

func formatToken(raw *big.Int, decimals int) string {
	if decimals <= 0 {
		return raw.String()
	}
	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	f := new(big.Float).SetInt(raw)
	f.Quo(f, new(big.Float).SetInt(div))
	return f.Text('f', decimals)
}

func parseBigHex(s string) (*big.Int, bool) {
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	return n, ok
}

// GetNonce returns the transaction count (nonce) for an address.
func (c *EVMClient) GetNonce(address string) (uint64, error) {
	result, err := c.call("eth_getTransactionCount", address, "latest")
	if err != nil {
		return 0, err
	}
	hexStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected result: %T", result)
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return 0, fmt.Errorf("could not parse nonce: %s", hexStr)
	}
	return n.Uint64(), nil
}

// CallContract calls a smart contract read function with the given calldata.
func (c *EVMClient) CallContract(toAddr, calldata string) (string, error) {
	result, err := c.call("eth_call", map[string]string{
		"to":   toAddr,
		"data": calldata,
	}, "latest")
	if err != nil {
		return "", err
	}
	s, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result: %T", result)
	}
	return s, nil
}

// TxReceipt holds the on-chain receipt of a mined transaction.
type TxReceipt struct {
	Hash            string
	Status          uint64 // 1 = success, 0 = reverted
	BlockNumber     uint64
	GasUsed         uint64
	ContractAddress string // non-empty when a contract was deployed
}

// GetTransactionReceipt fetches the receipt for hash.
// Returns nil, nil if the transaction is still pending.
func (c *EVMClient) GetTransactionReceipt(hash string) (*TxReceipt, error) {
	result, err := c.call("eth_getTransactionReceipt", hash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil // still pending
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var r struct {
		Status          string `json:"status"`
		BlockNumber     string `json:"blockNumber"`
		GasUsed         string `json:"gasUsed"`
		ContractAddress string `json:"contractAddress"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}

	receipt := &TxReceipt{Hash: hash, ContractAddress: r.ContractAddress}
	if s, ok := parseBigHex(r.Status); ok {
		receipt.Status = s.Uint64()
	}
	if bn, ok := parseBigHex(r.BlockNumber); ok {
		receipt.BlockNumber = bn.Uint64()
	}
	if gu, ok := parseBigHex(r.GasUsed); ok {
		receipt.GasUsed = gu.Uint64()
	}
	return receipt, nil
}

// WaitForReceipt polls every 2 s until the transaction is mined or timeout
// expires. Returns an error if the transaction reverted (Status == 0).
func (c *EVMClient) WaitForReceipt(hash string, timeout time.Duration) (*TxReceipt, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		receipt, err := c.GetTransactionReceipt(hash)
		if err != nil {
			return nil, err
		}
		if receipt != nil {
			if receipt.Status == 0 {
				return receipt, fmt.Errorf("transaction reverted (hash: %s)", hash)
			}
			return receipt, nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("transaction %s not mined within %s", hash, timeout)
}

// GetAllowance returns the ERC-20 allowance that owner has granted to spender
// on the given token contract. Uses the allowance(address,address) selector 0xdd62ed3e.
func (c *EVMClient) GetAllowance(tokenAddr, owner, spender string) (*big.Int, error) {
	data := "0xdd62ed3e" +
		fmt.Sprintf("%064s", strings.TrimPrefix(owner, "0x")) +
		fmt.Sprintf("%064s", strings.TrimPrefix(spender, "0x"))

	result, err := c.call("eth_call", map[string]string{
		"to":   tokenAddr,
		"data": data,
	}, "latest")
	if err != nil {
		return nil, err
	}
	hexStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result: %T", result)
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("could not parse allowance: %s", hexStr)
	}
	return n, nil
}

// GetPendingNonce returns the transaction count including pending (queued)
// transactions, using the "pending" block tag.
func (c *EVMClient) GetPendingNonce(address string) (uint64, error) {
	result, err := c.call("eth_getTransactionCount", address, "pending")
	if err != nil {
		return 0, err
	}
	hexStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected result: %T", result)
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return 0, fmt.Errorf("could not parse pending nonce: %s", hexStr)
	}
	return n.Uint64(), nil
}

// SimulateCall simulates a contract call using eth_call (with from field).
// Returns (true, returnData, nil) on success or (false, revertReason, nil) if
// the call reverts. Network errors return (false, "", err).
func (c *EVMClient) SimulateCall(from, to, data string, value *big.Int) (bool, string, error) {
	params := map[string]string{
		"from": from,
		"to":   to,
	}
	if data != "" {
		params["data"] = data
	}
	if value != nil && value.Sign() > 0 {
		params["value"] = "0x" + value.Text(16)
	}

	result, err := c.call("eth_call", params, "latest")
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "revert") || strings.Contains(errMsg, "execution") {
			return false, extractRevertReason(errMsg), nil
		}
		return false, "", err
	}

	hexStr, _ := result.(string)
	return true, hexStr, nil
}

// extractRevertReason tries to pull the revert reason out of an RPC error message.
func extractRevertReason(errMsg string) string {
	// Common pattern: "execution reverted: <reason>"
	if idx := strings.Index(errMsg, "execution reverted:"); idx >= 0 {
		return strings.TrimSpace(errMsg[idx:])
	}
	if idx := strings.Index(errMsg, "revert"); idx >= 0 {
		return strings.TrimSpace(errMsg[idx:])
	}
	return errMsg
}

// GetStorageAt reads a raw storage slot from a contract.
func (c *EVMClient) GetStorageAt(address, slot string) (string, error) {
	// Pad slot to 32 bytes if needed.
	if !strings.HasPrefix(slot, "0x") {
		slot = "0x" + slot
	}
	result, err := c.call("eth_getStorageAt", address, slot, "latest")
	if err != nil {
		return "", err
	}
	s, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result: %T", result)
	}
	return s, nil
}

// GetCode returns the bytecode at an address. Empty "0x" means EOA (no code).
func (c *EVMClient) GetCode(address string) (string, error) {
	result, err := c.call("eth_getCode", address, "latest")
	if err != nil {
		return "", err
	}
	s, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result: %T", result)
	}
	return s, nil
}

// LogEntry holds one event log.
type LogEntry struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber string   `json:"blockNumber"`
	TxHash      string   `json:"transactionHash"`
	LogIndex    string   `json:"logIndex"`
}

// GetLogs queries event logs matching the given filter.
func (c *EVMClient) GetLogs(address string, topics []string, fromBlock, toBlock string) ([]LogEntry, error) {
	filter := map[string]interface{}{
		"address":   address,
		"fromBlock": fromBlock,
		"toBlock":   toBlock,
	}
	if len(topics) > 0 {
		filter["topics"] = topics
	}

	result, err := c.call("eth_getLogs", filter)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var logs []LogEntry
	if err := json.Unmarshal(raw, &logs); err != nil {
		return nil, fmt.Errorf("parsing logs: %w", err)
	}
	return logs, nil
}

// Ping tests the RPC endpoint and returns latency + block number.
func (c *EVMClient) Ping(ctx context.Context) (latency time.Duration, blockNum uint64, err error) {
	start := time.Now()
	result, err := c.callCtx(ctx, "eth_blockNumber")
	latency = time.Since(start)
	if err != nil {
		return latency, 0, err
	}
	hexStr, ok := result.(string)
	if !ok {
		return latency, 0, fmt.Errorf("unexpected result: %T", result)
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return latency, 0, fmt.Errorf("could not parse block number")
	}
	return latency, n.Uint64(), nil
}

func (c *EVMClient) callCtx(ctx context.Context, method string, params ...interface{}) (interface{}, error) {
	reqBody, _ := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	var result interface{}
	json.Unmarshal(rpcResp.Result, &result)
	return result, nil
}
