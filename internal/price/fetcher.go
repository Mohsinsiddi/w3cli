package price

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Fetcher retrieves token prices from CoinGecko.
type Fetcher struct {
	client   *http.Client
	currency string
}

// NewFetcher creates a new price fetcher.
func NewFetcher(currency string) *Fetcher {
	if currency == "" {
		currency = "usd"
	}
	return &Fetcher{
		client:   &http.Client{Timeout: 10 * time.Second},
		currency: strings.ToLower(currency),
	}
}

// coinGeckoIDs maps chain names to CoinGecko coin IDs.
var coinGeckoIDs = map[string]string{
	"ethereum":     "ethereum",
	"base":         "ethereum",
	"polygon":      "matic-network",
	"arbitrum":     "ethereum",
	"optimism":     "ethereum",
	"bnb":          "binancecoin",
	"avalanche":    "avalanche-2",
	"fantom":       "fantom",
	"linea":        "ethereum",
	"zksync":       "ethereum",
	"scroll":       "ethereum",
	"mantle":       "mantle",
	"celo":         "celo",
	"gnosis":       "xdai",
	"blast":        "ethereum",
	"mode":         "ethereum",
	"zora":         "ethereum",
	"moonbeam":     "moonbeam",
	"cronos":       "crypto-com-chain",
	"klaytn":       "klay-token",
	"aurora":       "ethereum",
	"polygon-zkevm": "ethereum",
	"hyperliquid":  "hyperliquid",
	"boba":         "ethereum",
	"solana":       "solana",
	"sui":          "sui",
}

// GetPrice returns the USD price for a chain's native token.
func (f *Fetcher) GetPrice(chainName string) (float64, error) {
	id, ok := coinGeckoIDs[strings.ToLower(chainName)]
	if !ok {
		return 0, fmt.Errorf("unknown chain: %s", chainName)
	}
	return f.getByID(id)
}

// GetPrices fetches prices for multiple coin IDs at once.
func (f *Fetcher) GetPrices(chainNames []string) (map[string]float64, error) {
	ids := make(map[string]string)
	for _, cn := range chainNames {
		if id, ok := coinGeckoIDs[strings.ToLower(cn)]; ok {
			ids[cn] = id
		}
	}

	// Collect unique coin IDs.
	uniqueIDs := make(map[string]struct{})
	for _, id := range ids {
		uniqueIDs[id] = struct{}{}
	}
	idList := make([]string, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		idList = append(idList, id)
	}

	prices, err := f.fetchBatch(idList)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for cn, id := range ids {
		if p, ok := prices[id]; ok {
			result[cn] = p
		}
	}
	return result, nil
}

func (f *Fetcher) getByID(id string) (float64, error) {
	prices, err := f.fetchBatch([]string{id})
	if err != nil {
		return 0, err
	}
	p, ok := prices[id]
	if !ok {
		return 0, fmt.Errorf("price not available for: %s", id)
	}
	return p, nil
}

func (f *Fetcher) fetchBatch(ids []string) (map[string]float64, error) {
	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=%s",
		strings.Join(ids, ","),
		f.currency,
	)

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching prices: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading price response: %w", err)
	}

	// Response: {"ethereum":{"usd":1234.56}, ...}
	var raw map[string]map[string]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing price response: %w", err)
	}

	prices := make(map[string]float64)
	for id, currencies := range raw {
		if p, ok := currencies[f.currency]; ok {
			prices[id] = p
		}
	}
	return prices, nil
}
