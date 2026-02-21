package cmd

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// convertFromETH — internal logic tests
// ---------------------------------------------------------------------------

func TestConvertETHToWei_OneETH(t *testing.T) {
	f, _ := new(big.Float).SetString("1")
	weiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	weiF := new(big.Float).Mul(f, weiPerETH)
	wei, _ := weiF.Int(nil)
	expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	assert.Equal(t, expected, wei)
}

func TestConvertETHToWei_FractionalETH(t *testing.T) {
	f, _ := new(big.Float).SetString("1.5")
	weiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	weiF := new(big.Float).Mul(f, weiPerETH)
	wei, _ := weiF.Int(nil)
	expected, _ := new(big.Int).SetString("1500000000000000000", 10)
	assert.Equal(t, expected, wei)
}

func TestConvertETHToWei_Zero(t *testing.T) {
	f, _ := new(big.Float).SetString("0")
	weiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	weiF := new(big.Float).Mul(f, weiPerETH)
	wei, _ := weiF.Int(nil)
	assert.Equal(t, big.NewInt(0), wei)
}

func TestConvertETHToGwei_OneETH(t *testing.T) {
	f, _ := new(big.Float).SetString("1")
	gweiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil))
	gweiF := new(big.Float).Mul(f, gweiPerETH)
	gwei, _ := gweiF.Int(nil)
	expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	assert.Equal(t, expected, gwei)
}

func TestConvertGweiToETH(t *testing.T) {
	f, _ := new(big.Float).SetString("1000000000") // 1 gwei = 1e9
	gweiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil))
	ethF := new(big.Float).Quo(f, gweiPerETH)
	expected, _ := new(big.Float).SetString("1")
	assert.Equal(t, expected.Text('f', 18), ethF.Text('f', 18))
}

func TestConvertGweiToWei(t *testing.T) {
	f, _ := new(big.Float).SetString("50")
	weiPerGwei := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil))
	weiF := new(big.Float).Mul(f, weiPerGwei)
	wei, _ := weiF.Int(nil)
	expected := big.NewInt(50_000_000_000)
	assert.Equal(t, expected, wei)
}

func TestConvertWeiToETH(t *testing.T) {
	wei := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	weiPerETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	ethF := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(weiPerETH))
	assert.Equal(t, "1.000000000000000000", ethF.Text('f', 18))
}

func TestConvertWeiToGwei(t *testing.T) {
	wei := big.NewInt(50_000_000_000)
	weiPerGwei := new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	gweiF := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(weiPerGwei))
	assert.Equal(t, "50.000000000", gweiF.Text('f', 9))
}

func TestConvertHexToDec_FF(t *testing.T) {
	n, ok := new(big.Int).SetString("ff", 16)
	assert.True(t, ok)
	assert.Equal(t, int64(255), n.Int64())
}

func TestConvertHexToDec_Zero(t *testing.T) {
	n, ok := new(big.Int).SetString("0", 16)
	assert.True(t, ok)
	assert.Equal(t, int64(0), n.Int64())
}

func TestConvertDecToHex_255(t *testing.T) {
	n := big.NewInt(255)
	assert.Equal(t, "ff", n.Text(16))
}

func TestConvertDecToHex_Zero(t *testing.T) {
	n := big.NewInt(0)
	assert.Equal(t, "0", n.Text(16))
}

func TestConvertVeryLargeValue(t *testing.T) {
	// 1000 ETH in wei
	thousandETH, _ := new(big.Int).SetString("1000000000000000000000", 10)
	weiPerETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	ethF := new(big.Float).Quo(new(big.Float).SetInt(thousandETH), new(big.Float).SetInt(weiPerETH))
	assert.Equal(t, "1000.000000000000000000", ethF.Text('f', 18))
}

func TestConvertOneWei(t *testing.T) {
	wei := big.NewInt(1)
	weiPerETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	ethF := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(weiPerETH))
	assert.Equal(t, "0.000000000000000001", ethF.Text('f', 18))
}

// ---------------------------------------------------------------------------
// splitHexWords (used by decode)
// ---------------------------------------------------------------------------

func TestSplitHexWords_Empty(t *testing.T) {
	words := splitHexWords("")
	assert.Empty(t, words)
}

func TestSplitHexWords_SingleWord(t *testing.T) {
	// 64-char word.
	word := "000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
	words := splitHexWords(word)
	assert.Equal(t, 1, len(words))
	assert.Equal(t, word, words[0])
}

func TestSplitHexWords_TwoWords(t *testing.T) {
	w1 := "000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
	w2 := "0000000000000000000000000000000000000000000000000de0b6b3a7640000"
	words := splitHexWords(w1 + w2)
	assert.Equal(t, 2, len(words))
	assert.Equal(t, w1, words[0])
	assert.Equal(t, w2, words[1])
}

func TestSplitHexWords_ShortData(t *testing.T) {
	words := splitHexWords("abcdef")
	assert.Equal(t, 1, len(words))
	assert.Equal(t, "abcdef", words[0])
}
