package chain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// decodeMethod — pure function
// ---------------------------------------------------------------------------

func TestDecodeMethodEmpty(t *testing.T) {
	assert.Equal(t, "transfer", decodeMethod(""))
}

func TestDecodeMethodBareHexPrefix(t *testing.T) {
	assert.Equal(t, "transfer", decodeMethod("0x"))
}

func TestDecodeMethodShortInput(t *testing.T) {
	// Less than 8 hex chars after stripping 0x → "call"
	assert.Equal(t, "call", decodeMethod("0xabcd"))
}

func TestDecodeMethodKnownTransfer(t *testing.T) {
	// 0xa9059cbb + 64 bytes of params
	input := "0xa9059cbb" + "000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266" +
		"0000000000000000000000000000000000000000000000000de0b6b3a7640000"
	assert.Equal(t, "transfer", decodeMethod(input))
}

func TestDecodeMethodKnownApprove(t *testing.T) {
	assert.Equal(t, "approve", decodeMethod("0x095ea7b3"+"00"))
}

func TestDecodeMethodKnownSwapExactETH(t *testing.T) {
	assert.Equal(t, "swapExactETHForTokens", decodeMethod("0x7ff36ab5"+"00"))
}

func TestDecodeMethodKnownDeposit(t *testing.T) {
	assert.Equal(t, "deposit", decodeMethod("0xd0e30db0"))
}

func TestDecodeMethodKnownMint(t *testing.T) {
	assert.Equal(t, "mint", decodeMethod("0x6a627842"+"00"))
}

func TestDecodeMethodKnownMulticall(t *testing.T) {
	assert.Equal(t, "multicall", decodeMethod("0xac9650d8"+"00"))
}

func TestDecodeMethodKnownBalanceOf(t *testing.T) {
	assert.Equal(t, "balanceOf", decodeMethod("0x70a08231"+"00"))
}

func TestDecodeMethodUnknownSelector(t *testing.T) {
	// Unknown 4-byte selector → returns "0x" + first 8 hex chars
	result := decodeMethod("0xdeadbeef" + "00")
	assert.Equal(t, "0xdeadbeef", result)
}

func TestDecodeMethodUpperCaseInput(t *testing.T) {
	// Selector matching is case-insensitive
	assert.Equal(t, "transfer", decodeMethod("0xA9059CBB"+"00"))
}

func TestDecodeMethodAllKnownSelectorsDecodeToSomething(t *testing.T) {
	// Every selector in knownMethods must decode to its registered name (non-empty).
	for selector, name := range knownMethods {
		result := decodeMethod(selector + "00000000") // add dummy params
		assert.Equal(t, name, result, "selector %s should decode to %q", selector, name)
	}
}

// ---------------------------------------------------------------------------
// helpers for explorer API tests
// ---------------------------------------------------------------------------

// explorerServer creates a mock BlockScout-compatible API server.
// The handler func receives the full URL query and returns a response body.
func explorerServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func okTxListResponse(txs []explorerTx) []byte {
	b, _ := json.Marshal(txs)
	env := map[string]interface{}{
		"status":  "1",
		"message": "OK",
		"result":  json.RawMessage(b),
	}
	out, _ := json.Marshal(env)
	return out
}

func errorResponse(msg string) []byte {
	env := map[string]interface{}{
		"status":  "0",
		"message": msg,
		"result":  json.RawMessage(`"` + msg + `"`),
	}
	out, _ := json.Marshal(env)
	return out
}

// ---------------------------------------------------------------------------
// GetTransactionsFromExplorer
// ---------------------------------------------------------------------------

func TestGetTransactionsSuccess(t *testing.T) {
	txs := []explorerTx{
		{
			Hash:            "0xhash1",
			From:            "0xfrom1",
			To:              "0xto1",
			Value:           "1000000000000000000", // 1 ETH
			Gas:             "21000",
			GasUsed:         "21000",
			GasPrice:        "2000000000",
			Nonce:           "5",
			BlockNumber:     "1000",
			TimeStamp:       "1700000000",
			IsError:         "0",
			TxReceiptStatus: "1",
			Input:           "0x",
		},
		{
			Hash:            "0xhash2",
			From:            "0xfrom2",
			To:              "0xto2",
			Value:           "0",
			Gas:             "100000",
			GasUsed:         "80000",
			GasPrice:        "1000000000",
			Nonce:           "3",
			BlockNumber:     "999",
			TimeStamp:       "1699999000",
			IsError:         "0",
			TxReceiptStatus: "1",
			Input:           "0xa9059cbb0000",
		},
	}

	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(okTxListResponse(txs)) //nolint:errcheck
	})
	defer srv.Close()

	result, err := GetTransactionsFromExplorer(srv.URL, "0xaddr", 10, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, "0xhash1", result[0].Hash)
	assert.Equal(t, "1.000000000000000000", result[0].ValueETH)
	assert.True(t, result[0].Success)
	assert.Equal(t, "transfer", result[0].FunctionName) // "0x" input → transfer
	assert.False(t, result[0].IsContract)

	assert.Equal(t, "0xhash2", result[1].Hash)
	assert.Equal(t, "transfer", result[1].FunctionName) // 0xa9059cbb
	assert.True(t, result[1].IsContract)
}

func TestGetTransactionsFailedTx(t *testing.T) {
	// A tx with IsError=1 → Success=false.
	txs := []explorerTx{
		{
			Hash:            "0xfailed",
			IsError:         "1",
			TxReceiptStatus: "0",
			Input:           "0x",
		},
	}
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(okTxListResponse(txs)) //nolint:errcheck
	})
	defer srv.Close()

	result, err := GetTransactionsFromExplorer(srv.URL, "0xaddr", 10, "")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.False(t, result[0].Success)
}

func TestGetTransactionsContractDeploy(t *testing.T) {
	// To="" and ContractAddress set → deploy.
	txs := []explorerTx{
		{
			Hash:            "0xdeploy",
			To:              "",
			ContractAddress: "0xnewcontract",
			Input:           "0x608060",
			IsError:         "0",
			TxReceiptStatus: "1",
		},
	}
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(okTxListResponse(txs)) //nolint:errcheck
	})
	defer srv.Close()

	result, err := GetTransactionsFromExplorer(srv.URL, "0xaddr", 10, "")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "deploy", result[0].FunctionName)
	assert.Equal(t, "0xnewcontract", result[0].To)
	assert.True(t, result[0].IsContract)
}

func TestGetTransactionsEmptyResult(t *testing.T) {
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(okTxListResponse([]explorerTx{})) //nolint:errcheck
	})
	defer srv.Close()

	result, err := GetTransactionsFromExplorer(srv.URL, "0xaddr", 10, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetTransactionsAPIError(t *testing.T) {
	// status=0 with error message.
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(errorResponse("Max rate limit reached")) //nolint:errcheck
	})
	defer srv.Close()

	_, err := GetTransactionsFromExplorer(srv.URL, "0xaddr", 10, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Max rate limit reached")
}

func TestGetTransactionsInvalidJSON(t *testing.T) {
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{not valid}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := GetTransactionsFromExplorer(srv.URL, "0xaddr", 10, "")
	require.Error(t, err)
}

func TestGetTransactionsConnectionRefused(t *testing.T) {
	_, err := GetTransactionsFromExplorer("http://127.0.0.1:19998", "0xaddr", 10, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "explorer request failed")
}

func TestGetTransactionsWithAPIKeyAppended(t *testing.T) {
	// Verify the API key is appended to the query string.
	var capturedURL string
	srv := explorerServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.RawQuery
		w.Write(okTxListResponse([]explorerTx{})) //nolint:errcheck
	})
	defer srv.Close()

	GetTransactionsFromExplorer(srv.URL, "0xaddr", 5, "MYAPIKEY123") //nolint:errcheck
	assert.Contains(t, capturedURL, "apikey=MYAPIKEY123")
}

func TestGetTransactionsNoAPIKeyWhenEmpty(t *testing.T) {
	var capturedURL string
	srv := explorerServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.RawQuery
		w.Write(okTxListResponse([]explorerTx{})) //nolint:errcheck
	})
	defer srv.Close()

	GetTransactionsFromExplorer(srv.URL, "0xaddr", 5, "") //nolint:errcheck
	assert.NotContains(t, capturedURL, "apikey")
}

func TestGetTransactionsNumericFieldsParsed(t *testing.T) {
	txs := []explorerTx{
		{
			Hash:            "0xparsed",
			Value:           "500000000000000000", // 0.5 ETH
			Gas:             "50000",
			GasUsed:         "40000",
			GasPrice:        "5000000000",
			Nonce:           "7",
			BlockNumber:     "20000000",
			TimeStamp:       "1710000000",
			IsError:         "0",
			TxReceiptStatus: "1",
			Input:           "0x",
		},
	}
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(okTxListResponse(txs)) //nolint:errcheck
	})
	defer srv.Close()

	result, err := GetTransactionsFromExplorer(srv.URL, "0xaddr", 10, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	tx := result[0]
	assert.Equal(t, "0.500000000000000000", tx.ValueETH)
	assert.Equal(t, uint64(50000), tx.Gas)
	assert.Equal(t, uint64(40000), tx.GasUsed)
	assert.Equal(t, uint64(7), tx.Nonce)
	assert.Equal(t, uint64(20000000), tx.BlockNum)
	assert.Equal(t, uint64(1710000000), tx.Timestamp)
	assert.NotNil(t, tx.GasPrice)
}

// ---------------------------------------------------------------------------
// FetchContractNames
// ---------------------------------------------------------------------------

func TestFetchContractNamesSuccess(t *testing.T) {
	srv := explorerServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(contractSourceResponse{ //nolint:errcheck
			Status: "1",
			Result: []contractSourceResult{{ContractName: "FiatTokenProxy"}},
		})
	})
	defer srv.Close()

	names := FetchContractNames(srv.URL, []string{"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"}, "")
	require.Len(t, names, 1)
	// Key is lowercased.
	assert.Equal(t, "FiatTokenProxy", names["0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"])
}

func TestFetchContractNamesDeduplicate(t *testing.T) {
	callCount := 0
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(contractSourceResponse{ //nolint:errcheck
			Status: "1",
			Result: []contractSourceResult{{ContractName: "SomeContract"}},
		})
	})
	defer srv.Close()

	// Pass same address 3 times → only 1 HTTP request.
	addr := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	FetchContractNames(srv.URL, []string{addr, addr, addr}, "")
	assert.Equal(t, 1, callCount)
}

func TestFetchContractNamesEOAReturnsEmpty(t *testing.T) {
	// status=0 → not a contract → omit from map.
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(contractSourceResponse{ //nolint:errcheck
			Status: "0",
			Result: []contractSourceResult{},
		})
	})
	defer srv.Close()

	names := FetchContractNames(srv.URL, []string{"0xregularwallet"}, "")
	assert.Empty(t, names)
}

func TestFetchContractNamesEmptyInput(t *testing.T) {
	srv := explorerServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("should not make any HTTP requests for empty input")
	})
	defer srv.Close()

	names := FetchContractNames(srv.URL, []string{}, "")
	assert.Empty(t, names)
}

func TestFetchContractNamesPartialFailure(t *testing.T) {
	// First address: success. Second: returns empty name (treated as unknown).
	requestCount := 0
	addrs := []string{
		"0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
	}
	srv := explorerServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Query().Get("address") == addrs[0] {
			json.NewEncoder(w).Encode(contractSourceResponse{ //nolint:errcheck
				Status: "1",
				Result: []contractSourceResult{{ContractName: "GoodContract"}},
			})
		} else {
			// Second address: empty contract name.
			json.NewEncoder(w).Encode(contractSourceResponse{ //nolint:errcheck
				Status: "1",
				Result: []contractSourceResult{{ContractName: ""}},
			})
		}
	})
	defer srv.Close()

	names := FetchContractNames(srv.URL, addrs, "")
	assert.Equal(t, 1, len(names))
	// Map keys are lowercased inside FetchContractNames.
	assert.Equal(t, "GoodContract", names[strings.ToLower(addrs[0])])
}

func TestFetchContractNamesUnreachableServerSkipped(t *testing.T) {
	// FetchContractNames should never return an error — failures are silently skipped.
	names := FetchContractNames("http://127.0.0.1:19997", []string{"0xabc"}, "")
	assert.NotNil(t, names)
	assert.Empty(t, names)
}

func TestFetchContractNamesAPIKeyAppended(t *testing.T) {
	var capturedQuery string
	srv := explorerServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(contractSourceResponse{ //nolint:errcheck
			Status: "0",
		})
	})
	defer srv.Close()

	FetchContractNames(srv.URL, []string{"0xabc"}, "MYKEY")
	assert.Contains(t, capturedQuery, "MYKEY")
}
