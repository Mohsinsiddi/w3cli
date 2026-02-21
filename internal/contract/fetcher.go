package contract

import (
	"bytes"
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

// LoadFromFile loads a raw ABI JSON array from a local file path.
func LoadFromFile(path string) ([]ABIEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading ABI file %s: %w", path, err)
	}
	return parseABI(data)
}

// LoadFromArtifact loads an ABI from a local file that is either:
//   - a raw ABI JSON array: [{"type":"function",...}, ...]
//   - a Hardhat/Foundry artifact: {"abi":[...],"bytecode":"0x...",...}
//
// Both formats are detected automatically.
func LoadFromArtifact(path string) ([]ABIEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read ABI file: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("ABI file is empty: %s", path)
	}

	// Attempt to detect a Hardhat/Foundry artifact (object with an "abi" key).
	var artifact struct {
		ABI json.RawMessage `json:"abi"`
	}
	if json.Unmarshal(data, &artifact) == nil && len(artifact.ABI) > 1 {
		// artifact.ABI[0] should be '[' for a valid ABI array inside the artifact.
		if artifact.ABI[0] == '[' {
			abi, err := parseABI(artifact.ABI)
			if err != nil {
				return nil, err
			}
			if err := validateABI(abi, path); err != nil {
				return nil, err
			}
			return abi, nil
		}
	}

	// Fall back: treat the whole file as a raw ABI array.
	abi, err := parseABI(data)
	if err != nil {
		return nil, err
	}
	if err := validateABI(abi, path); err != nil {
		return nil, err
	}
	return abi, nil
}

func parseABI(data []byte) ([]ABIEntry, error) {
	var abi []ABIEntry
	if err := json.Unmarshal(data, &abi); err != nil {
		// Provide a user-friendly error depending on the JSON content.
		data = bytes.TrimSpace(data)
		if len(data) > 0 && data[0] == '{' {
			return nil, fmt.Errorf("file is a JSON object, not an ABI array — if this is a Hardhat/Foundry artifact it must have an \"abi\" key")
		}
		return nil, fmt.Errorf("invalid ABI JSON: expected an array of function/event definitions, got parse error: %w", err)
	}
	return abi, nil
}

// validateABI checks that the parsed ABI has at least one function or event.
func validateABI(abi []ABIEntry, path string) error {
	if len(abi) == 0 {
		return fmt.Errorf("ABI is empty (no functions or events found): %s", path)
	}
	hasFuncOrEvent := false
	for _, e := range abi {
		if e.Type == "function" || e.Type == "event" || e.Type == "constructor" {
			hasFuncOrEvent = true
			break
		}
	}
	if !hasFuncOrEvent {
		return fmt.Errorf("ABI has %d entries but none are functions or events — check the file format: %s", len(abi), path)
	}
	return nil
}
