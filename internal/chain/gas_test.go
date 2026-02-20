package chain

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// WeiToGwei
// ---------------------------------------------------------------------------

func TestWeiToGweiNil(t *testing.T) {
	assert.Equal(t, float64(0), WeiToGwei(nil))
}

func TestWeiToGweiZero(t *testing.T) {
	assert.Equal(t, float64(0), WeiToGwei(big.NewInt(0)))
}

func TestWeiToGweiOneGwei(t *testing.T) {
	oneGwei := big.NewInt(1_000_000_000)
	got := WeiToGwei(oneGwei)
	assert.InDelta(t, 1.0, got, 0.0001)
}

func TestWeiToGweiTwoGwei(t *testing.T) {
	two := big.NewInt(2_000_000_000)
	got := WeiToGwei(two)
	assert.InDelta(t, 2.0, got, 0.0001)
}

func TestWeiToGwei100Gwei(t *testing.T) {
	wei := new(big.Int).Mul(big.NewInt(100), big.NewInt(1_000_000_000))
	got := WeiToGwei(wei)
	assert.InDelta(t, 100.0, got, 0.0001)
}

// ---------------------------------------------------------------------------
// GasInfo.GasPriceDisplay
// ---------------------------------------------------------------------------

func TestGasPriceDisplayLegacy(t *testing.T) {
	info := &GasInfo{
		GasPrice:     big.NewInt(1_000_000_000),
		GasPriceGwei: 1.0,
	}
	gwei, isEIP1559 := info.GasPriceDisplay()
	assert.InDelta(t, 1.0, gwei, 0.001)
	assert.False(t, isEIP1559)
}

func TestGasPriceDisplayEIP1559(t *testing.T) {
	info := &GasInfo{
		GasPrice:     big.NewInt(2_000_000_000),
		GasPriceGwei: 2.0,
		BaseFee:      big.NewInt(3_000_000_000),
		BaseFeeGwei:  3.0,
	}
	gwei, isEIP1559 := info.GasPriceDisplay()
	assert.InDelta(t, 3.0, gwei, 0.001)
	assert.True(t, isEIP1559)
}

func TestGasPriceDisplayZeroBaseFee(t *testing.T) {
	// BaseFee set but 0 Gwei — fall back to legacy
	info := &GasInfo{
		GasPrice:     big.NewInt(5_000_000_000),
		GasPriceGwei: 5.0,
		BaseFee:      big.NewInt(0),
		BaseFeeGwei:  0,
	}
	gwei, isEIP1559 := info.GasPriceDisplay()
	assert.InDelta(t, 5.0, gwei, 0.001)
	assert.False(t, isEIP1559)
}

// ---------------------------------------------------------------------------
// BlockInfo.Age
// ---------------------------------------------------------------------------

func TestBlockInfoAgeUnknown(t *testing.T) {
	b := &BlockInfo{Timestamp: 0}
	assert.Equal(t, "unknown", b.Age())
}

func TestBlockInfoAgeSeconds(t *testing.T) {
	ts := uint64(time.Now().Unix()) - 5 // 5 seconds ago
	b := &BlockInfo{Timestamp: ts}
	age := b.Age()
	assert.Contains(t, age, "s ago")
}

func TestBlockInfoAgeMinutes(t *testing.T) {
	ts := uint64(time.Now().Unix()) - 120 // 2 minutes ago
	b := &BlockInfo{Timestamp: ts}
	age := b.Age()
	assert.Contains(t, age, "m ago")
}

func TestBlockInfoAgeHours(t *testing.T) {
	ts := uint64(time.Now().Unix()) - 7200 // 2 hours ago
	b := &BlockInfo{Timestamp: ts}
	age := b.Age()
	assert.Contains(t, age, "h ago")
}

func TestBlockInfoAgeBoundary59Seconds(t *testing.T) {
	ts := uint64(time.Now().Unix()) - 59
	b := &BlockInfo{Timestamp: ts}
	assert.Contains(t, b.Age(), "s ago")
}

func TestBlockInfoAgeBoundary60Seconds(t *testing.T) {
	ts := uint64(time.Now().Unix()) - 60
	b := &BlockInfo{Timestamp: ts}
	assert.Contains(t, b.Age(), "m ago")
}

func TestBlockInfoAgeBoundary3600Seconds(t *testing.T) {
	ts := uint64(time.Now().Unix()) - 3600
	b := &BlockInfo{Timestamp: ts}
	assert.Contains(t, b.Age(), "h ago")
}

// ---------------------------------------------------------------------------
// BlockInfo.GasUsedPct
// ---------------------------------------------------------------------------

func TestGasUsedPctZeroLimit(t *testing.T) {
	b := &BlockInfo{GasUsed: 1000, GasLimit: 0}
	assert.Equal(t, "—", b.GasUsedPct())
}

func TestGasUsedPctFull(t *testing.T) {
	b := &BlockInfo{GasUsed: 15_000_000, GasLimit: 15_000_000}
	assert.Equal(t, "100.0%", b.GasUsedPct())
}

func TestGasUsedPctHalf(t *testing.T) {
	b := &BlockInfo{GasUsed: 7_500_000, GasLimit: 15_000_000}
	assert.Equal(t, "50.0%", b.GasUsedPct())
}

func TestGasUsedPctZero(t *testing.T) {
	b := &BlockInfo{GasUsed: 0, GasLimit: 15_000_000}
	assert.Equal(t, "0.0%", b.GasUsedPct())
}

func TestGasUsedPctTypical(t *testing.T) {
	b := &BlockInfo{GasUsed: 10_000_000, GasLimit: 30_000_000}
	pct := b.GasUsedPct()
	assert.Contains(t, pct, "33.")
}

// ---------------------------------------------------------------------------
// EVMClient — GetLatestBlockInfo (via mock)
// ---------------------------------------------------------------------------

func TestGetLatestBlockInfoSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getBlockByNumber": map[string]interface{}{
			"number":           "0x1",
			"hash":             "0xblockhash",
			"timestamp":        "0x5f5e100",
			"transactions":     []interface{}{},
			"gasUsed":          "0xE4E1C0",
			"gasLimit":         "0x1C9C380",
			"baseFeePerGas":    "0x3B9ACA00",
			"miner":            "0xminer",
		},
	})
	defer srv.Close()

	info, err := NewEVMClient(srv.URL).GetLatestBlockInfo()
	require.NoError(t, err)
	assert.Equal(t, uint64(1), info.Number)
	assert.Equal(t, "0xblockhash", info.Hash)
	assert.Equal(t, 0, info.TxCount)
	assert.Equal(t, "0xminer", info.Miner)
	assert.NotNil(t, info.BaseFee)
}

func TestGetLatestBlockInfoNullResult(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getBlockByNumber": nil,
	})
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetLatestBlockInfo()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block not found")
}

func TestGetLatestBlockInfoRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "server error")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetLatestBlockInfo()
	require.Error(t, err)
}

func TestGetLatestBlockInfoNoBaseFee(t *testing.T) {
	// Legacy chain without EIP-1559.
	srv := rpcMock(t, map[string]interface{}{
		"eth_getBlockByNumber": map[string]interface{}{
			"number":       "0x100",
			"hash":         "0xabc",
			"timestamp":    "0x100",
			"transactions": []interface{}{},
			"gasUsed":      "0x5208",
			"gasLimit":     "0xE4E1C0",
			"miner":        "0xminer",
		},
	})
	defer srv.Close()

	info, err := NewEVMClient(srv.URL).GetLatestBlockInfo()
	require.NoError(t, err)
	assert.Nil(t, info.BaseFee)
}

func TestGetBlockTransactions(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getBlockByNumber": map[string]interface{}{
			"transactions": []interface{}{
				map[string]interface{}{
					"hash":        "0xtx1",
					"from":        "0xfrom",
					"to":          "0xto",
					"value":       "0x0",
					"gas":         "0x5208",
					"gasPrice":    "0x77359400",
					"nonce":       "0x1",
					"blockNumber": "0xA",
				},
			},
		},
	})
	defer srv.Close()

	txs, err := NewEVMClient(srv.URL).GetBlockTransactions(10)
	require.NoError(t, err)
	assert.Len(t, txs, 1)
	assert.Equal(t, "0xtx1", txs[0].Hash)
}

func TestGetBlockTransactionsNullBlock(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getBlockByNumber": nil,
	})
	defer srv.Close()

	txs, err := NewEVMClient(srv.URL).GetBlockTransactions(999)
	require.NoError(t, err)
	assert.Nil(t, txs)
}

// ---------------------------------------------------------------------------
// EVMClient — GetGasInfo (via mock)
// ---------------------------------------------------------------------------

func TestGetGasInfoLegacyChain(t *testing.T) {
	// Block with no baseFeePerGas = legacy chain.
	srv := rpcMock(t, map[string]interface{}{
		"eth_gasPrice": "0x77359400", // 2 Gwei
		"eth_getBlockByNumber": map[string]interface{}{
			"number":       "0x1",
			"transactions": []interface{}{},
		},
	})
	defer srv.Close()

	info, err := NewEVMClient(srv.URL).GetGasInfo()
	require.NoError(t, err)
	assert.NotNil(t, info.GasPrice)
	assert.Nil(t, info.BaseFee)
	assert.InDelta(t, 2.0, info.GasPriceGwei, 0.01)
}

func TestGetGasInfoEIP1559Chain(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_gasPrice": "0x77359400",
		"eth_getBlockByNumber": map[string]interface{}{
			"number":        "0x1",
			"transactions":  []interface{}{},
			"baseFeePerGas": "0x3B9ACA00", // 1 Gwei
		},
	})
	defer srv.Close()

	info, err := NewEVMClient(srv.URL).GetGasInfo()
	require.NoError(t, err)
	assert.NotNil(t, info.BaseFee)
	assert.InDelta(t, 1.0, info.BaseFeeGwei, 0.01)
}

func TestGetGasInfoGasPriceError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "gas price unavailable")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetGasInfo()
	require.Error(t, err)
}
