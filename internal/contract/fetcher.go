package contract

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Fetcher retrieves ABIs from block explorers or URLs.
type Fetcher struct {
	client *http.Client
	apiKey string
}

// NewFetcher creates a new ABI fetcher.
func NewFetcher(apiKey string) *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: 15 * time.Second},
		apiKey: apiKey,
	}
}

// FetchFromExplorer fetches an ABI from an Etherscan-compatible API.
// explorerAPIURL example: "https://api.etherscan.io"
func (f *Fetcher) FetchFromExplorer(explorerAPIURL, address string) ([]ABIEntry, error) {
	url := fmt.Sprintf(
		"%s/api?module=contract&action=getabi&address=%s&apikey=%s",
		explorerAPIURL, address, f.apiKey,
	)

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching ABI: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing ABI response: %w", err)
	}

	if result.Status != "1" {
		return nil, fmt.Errorf("explorer error: %s", result.Message)
	}

	return parseABI([]byte(result.Result))
}

// FetchFromURL fetches a raw ABI JSON array from any URL.
func (f *Fetcher) FetchFromURL(url string) ([]ABIEntry, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching ABI from URL: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return parseABI(body)
}

// LoadFromFile loads an ABI JSON array from a local file path.
func LoadFromFile(path string) ([]ABIEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading ABI file %s: %w", path, err)
	}
	return parseABI(data)
}

func parseABI(data []byte) ([]ABIEntry, error) {
	var abi []ABIEntry
	if err := json.Unmarshal(data, &abi); err != nil {
		return nil, fmt.Errorf("parsing ABI JSON: %w", err)
	}
	return abi, nil
}
