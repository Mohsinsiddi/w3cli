package chain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

// GasInfo holds current gas pricing data for a chain.
type GasInfo struct {
	GasPrice     *big.Int // legacy eth_gasPrice (Wei)
	BaseFee      *big.Int // EIP-1559 base fee (Wei), nil on legacy chains
	GasPriceGwei float64
	BaseFeeGwei  float64
}

// GasPriceDisplay returns the best gas price for display (Gwei) and whether
// the chain supports EIP-1559.
func (g *GasInfo) GasPriceDisplay() (gwei float64, isEIP1559 bool) {
	if g.BaseFee != nil && g.BaseFeeGwei > 0 {
		return g.BaseFeeGwei, true
	}
	return g.GasPriceGwei, false
}

// GetGasInfo fetches gas price via eth_gasPrice and base fee from latest block.
func (c *EVMClient) GetGasInfo() (*GasInfo, error) {
	gp, err := c.GasPrice()
	if err != nil {
		return nil, err
	}
	info := &GasInfo{
		GasPrice:     gp,
		GasPriceGwei: WeiToGwei(gp),
	}
	// Try EIP-1559 base fee from latest block header.
	blockResult, err := c.call("eth_getBlockByNumber", "latest", false)
	if err == nil && blockResult != nil {
		raw, _ := json.Marshal(blockResult)
		var rb struct {
			BaseFeePerGas string `json:"baseFeePerGas"`
		}
		if json.Unmarshal(raw, &rb) == nil && rb.BaseFeePerGas != "" {
			if bf, ok := parseBigHex(rb.BaseFeePerGas); ok {
				info.BaseFee = bf
				info.BaseFeeGwei = WeiToGwei(bf)
			}
		}
	}
	return info, nil
}

// BlockInfo holds summary data for a block header.
type BlockInfo struct {
	Number    uint64
	Hash      string
	Timestamp uint64
	TxCount   int
	GasUsed   uint64
	GasLimit  uint64
	BaseFee   *big.Int // nil on pre-EIP-1559 chains
	Miner     string
}

// Age returns a human-readable relative age string.
func (b *BlockInfo) Age() string {
	if b.Timestamp == 0 {
		return "unknown"
	}
	diff := uint64(time.Now().Unix()) - b.Timestamp
	switch {
	case diff < 60:
		return fmt.Sprintf("%ds ago", diff)
	case diff < 3600:
		return fmt.Sprintf("%dm ago", diff/60)
	default:
		return fmt.Sprintf("%dh ago", diff/3600)
	}
}

// GasUsedPct returns gas utilisation as a percentage string.
func (b *BlockInfo) GasUsedPct() string {
	if b.GasLimit == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", float64(b.GasUsed)/float64(b.GasLimit)*100)
}

// GetLatestBlockInfo fetches the latest block header (no full transaction objects).
func (c *EVMClient) GetLatestBlockInfo() (*BlockInfo, error) {
	result, err := c.call("eth_getBlockByNumber", "latest", false)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("block not found")
	}
	raw, _ := json.Marshal(result)
	var rb struct {
		Number        string        `json:"number"`
		Hash          string        `json:"hash"`
		Timestamp     string        `json:"timestamp"`
		Transactions  []interface{} `json:"transactions"`
		GasUsed       string        `json:"gasUsed"`
		GasLimit      string        `json:"gasLimit"`
		BaseFeePerGas string        `json:"baseFeePerGas"`
		Miner         string        `json:"miner"`
	}
	if err := json.Unmarshal(raw, &rb); err != nil {
		return nil, fmt.Errorf("parsing block: %w", err)
	}
	info := &BlockInfo{Hash: rb.Hash, Miner: rb.Miner, TxCount: len(rb.Transactions)}
	if n, ok := parseBigHex(rb.Number); ok {
		info.Number = n.Uint64()
	}
	if ts, ok := parseBigHex(rb.Timestamp); ok {
		info.Timestamp = ts.Uint64()
	}
	if gu, ok := parseBigHex(rb.GasUsed); ok {
		info.GasUsed = gu.Uint64()
	}
	if gl, ok := parseBigHex(rb.GasLimit); ok {
		info.GasLimit = gl.Uint64()
	}
	if rb.BaseFeePerGas != "" {
		if bf, ok := parseBigHex(rb.BaseFeePerGas); ok {
			info.BaseFee = bf
		}
	}
	return info, nil
}

// GetBlockTransactions returns all transactions in a specific block number.
func (c *EVMClient) GetBlockTransactions(blockNum uint64) ([]*Transaction, error) {
	return c.getBlock(blockNum)
}

// WeiToGwei converts a Wei value to Gwei as float64.
func WeiToGwei(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}
	f, _ := new(big.Float).Quo(
		new(big.Float).SetInt(wei),
		new(big.Float).SetFloat64(1e9),
	).Float64()
	return f
}
