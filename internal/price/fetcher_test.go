package price

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// fixedTransport: replaces the HTTP client without needing a real server.
// ---------------------------------------------------------------------------

type fixedTransport struct {
	body string
	code int
	err  error
}

func (ft *fixedTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	if ft.err != nil {
		return nil, ft.err
	}
	return &http.Response{
		StatusCode: ft.code,
		Body:       io.NopCloser(strings.NewReader(ft.body)),
		Header:     make(http.Header),
	}, nil
}

// newMockFetcher returns a Fetcher whose HTTP calls are intercepted.
func newMockFetcher(body string, code int) *Fetcher {
	f := NewFetcher("usd")
	f.client = &http.Client{Transport: &fixedTransport{body: body, code: code}}
	return f
}

func newErrFetcher(err error) *Fetcher {
	f := NewFetcher("usd")
	f.client = &http.Client{Transport: &fixedTransport{err: err}}
	return f
}

// ---------------------------------------------------------------------------
// NewFetcher
// ---------------------------------------------------------------------------

func TestNewFetcherDefaultCurrency(t *testing.T) {
	f := NewFetcher("")
	assert.Equal(t, "usd", f.currency)
}

func TestNewFetcherCustomCurrency(t *testing.T) {
	f := NewFetcher("EUR")
	assert.Equal(t, "eur", f.currency, "currency must be lowercased")
}

func TestNewFetcherLowercasesGBP(t *testing.T) {
	f := NewFetcher("GBP")
	assert.Equal(t, "gbp", f.currency)
}

// ---------------------------------------------------------------------------
// GetPrice
// ---------------------------------------------------------------------------

func TestGetPriceKnownChain(t *testing.T) {
	body := `{"ethereum":{"usd":3000.50}}`
	f := newMockFetcher(body, http.StatusOK)

	price, err := f.GetPrice("ethereum")
	require.NoError(t, err)
	assert.InDelta(t, 3000.50, price, 0.001)
}

func TestGetPriceBaseUsesEthereumID(t *testing.T) {
	// Base uses CoinGecko ID "ethereum".
	body := `{"ethereum":{"usd":2500.00}}`
	f := newMockFetcher(body, http.StatusOK)

	price, err := f.GetPrice("base")
	require.NoError(t, err)
	assert.InDelta(t, 2500.0, price, 0.001)
}

func TestGetPricePolygon(t *testing.T) {
	body := `{"matic-network":{"usd":0.85}}`
	f := newMockFetcher(body, http.StatusOK)

	price, err := f.GetPrice("polygon")
	require.NoError(t, err)
	assert.InDelta(t, 0.85, price, 0.001)
}

func TestGetPriceUnknownChain(t *testing.T) {
	f := newMockFetcher("{}", http.StatusOK)
	_, err := f.GetPrice("fakechain99")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown chain")
}

func TestGetPriceCaseInsensitive(t *testing.T) {
	body := `{"ethereum":{"usd":2000.0}}`
	f := newMockFetcher(body, http.StatusOK)

	// "Ethereum" should match "ethereum" in the map.
	price, err := f.GetPrice("Ethereum")
	require.NoError(t, err)
	assert.InDelta(t, 2000.0, price, 0.001)
}

func TestGetPriceHTTPError(t *testing.T) {
	f := newErrFetcher(&networkError{msg: "connection refused"})
	_, err := f.GetPrice("ethereum")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching prices")
}

func TestGetPriceInvalidJSON(t *testing.T) {
	f := newMockFetcher("{not valid json", http.StatusOK)
	_, err := f.GetPrice("ethereum")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing price response")
}

func TestGetPriceMissingIDInResponse(t *testing.T) {
	// Response doesn't contain the coin ID we asked for.
	body := `{"bitcoin":{"usd":50000}}`
	f := newMockFetcher(body, http.StatusOK)

	_, err := f.GetPrice("ethereum")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "price not available")
}

// ---------------------------------------------------------------------------
// GetPrices (batch)
// ---------------------------------------------------------------------------

func TestGetPricesMultipleChains(t *testing.T) {
	body := `{"ethereum":{"usd":3000},"matic-network":{"usd":0.90},"binancecoin":{"usd":400}}`
	f := newMockFetcher(body, http.StatusOK)

	prices, err := f.GetPrices([]string{"ethereum", "polygon", "bnb"})
	require.NoError(t, err)

	assert.InDelta(t, 3000.0, prices["ethereum"], 0.001)
	assert.InDelta(t, 0.90, prices["polygon"], 0.001)
	assert.InDelta(t, 400.0, prices["bnb"], 0.001)
}

func TestGetPricesDedupsCoinGeckoIDs(t *testing.T) {
	// ethereum, base, arbitrum, optimism all map to "ethereum" CoinGecko ID.
	// fetchBatch should be called with just one ID.
	callCount := 0
	f := NewFetcher("usd")
	f.client = &http.Client{Transport: &countingTransport{
		body:  `{"ethereum":{"usd":3000}}`,
		count: &callCount,
	}}

	prices, err := f.GetPrices([]string{"ethereum", "base", "arbitrum", "optimism"})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "deduped IDs should produce a single HTTP request")
	// All four chains should get the ethereum price.
	for _, chain := range []string{"ethereum", "base", "arbitrum", "optimism"} {
		assert.InDelta(t, 3000.0, prices[chain], 0.001, "chain %s", chain)
	}
}

func TestGetPricesMixedKnownUnknown(t *testing.T) {
	body := `{"ethereum":{"usd":3000}}`
	f := newMockFetcher(body, http.StatusOK)

	prices, err := f.GetPrices([]string{"ethereum", "unknownchain99"})
	require.NoError(t, err)
	assert.Contains(t, prices, "ethereum")
	assert.NotContains(t, prices, "unknownchain99")
}

func TestGetPricesHTTPError(t *testing.T) {
	f := newErrFetcher(&networkError{msg: "dial tcp: connection refused"})
	_, err := f.GetPrices([]string{"ethereum"})
	require.Error(t, err)
}

func TestGetPricesInvalidJSON(t *testing.T) {
	f := newMockFetcher("NOTJSON", http.StatusOK)
	_, err := f.GetPrices([]string{"ethereum"})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// coinGeckoIDs mapping completeness
// ---------------------------------------------------------------------------

func TestCoinGeckoIDsMappingComplete(t *testing.T) {
	// Every chain registered should have a non-empty CoinGecko ID.
	for chain, id := range coinGeckoIDs {
		assert.NotEmpty(t, id, "chain %q has empty CoinGecko ID", chain)
	}
}

func TestCoinGeckoIDsHasAllMajorChains(t *testing.T) {
	required := []string{
		"ethereum", "base", "polygon", "arbitrum", "optimism",
		"bnb", "avalanche", "solana", "sui",
	}
	for _, chain := range required {
		_, ok := coinGeckoIDs[chain]
		assert.True(t, ok, "coinGeckoIDs must contain %q", chain)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// networkError satisfies the error interface for transport-level failures.
type networkError struct{ msg string }

func (e *networkError) Error() string { return e.msg }

// countingTransport counts how many HTTP requests are made.
type countingTransport struct {
	body  string
	count *int
}

func (ct *countingTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	*ct.count++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(ct.body)),
		Header:     make(http.Header),
	}, nil
}
