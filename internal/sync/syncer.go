package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/Mohsinsiddi/w3cli/internal/config"
)

// Manifest is the structure of a deployments.json manifest.
type Manifest struct {
	Contracts map[string]map[string]ManifestEntry `json:"contracts"`
}

// ManifestEntry is a single contract deployment entry.
type ManifestEntry struct {
	Address string `json:"address"`
	ABIUrl  string `json:"abi_url"`
}

// Syncer handles fetching and updating contracts from a remote manifest.
type Syncer struct {
	cfg     *config.Config
	reg     *contract.Registry
	fetcher *contract.Fetcher
	client  *http.Client
}

// New creates a new Syncer.
func New(cfg *config.Config, reg *contract.Registry) *Syncer {
	return &Syncer{
		cfg:     cfg,
		reg:     reg,
		fetcher: contract.NewFetcher(""),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Run fetches the manifest from the configured source and updates the contract registry.
func (s *Syncer) Run(ctx context.Context) error {
	syncCfg, err := s.cfg.LoadSync()
	if err != nil {
		return fmt.Errorf("loading sync config: %w", err)
	}

	if syncCfg.Source == "" {
		return fmt.Errorf("no sync source configured — run: w3cli sync set-source <url>")
	}

	manifest, err := s.fetchManifest(ctx, syncCfg.Source)
	if err != nil {
		return fmt.Errorf("fetching manifest: %w", err)
	}

	for name, networks := range manifest.Contracts {
		for network, entry := range networks {
			abi, err := s.fetcher.FetchFromURL(entry.ABIUrl)
			if err != nil {
				fmt.Printf("warning: could not fetch ABI for %s on %s: %v\n", name, network, err)
				abi = nil
			}

			s.reg.Add(&contract.Entry{
				Name:    name,
				Network: network,
				Address: entry.Address,
				ABI:     abi,
				ABIUrl:  entry.ABIUrl,
			})
		}
	}

	if err := s.reg.Save(); err != nil {
		return fmt.Errorf("saving contracts: %w", err)
	}

	// Update last synced timestamp.
	syncCfg.LastSynced = time.Now().UTC().Format(time.RFC3339)
	return s.cfg.SaveSync(syncCfg)
}

// SetSource sets the remote manifest URL.
func (s *Syncer) SetSource(url string) error {
	syncCfg, err := s.cfg.LoadSync()
	if err != nil {
		return err
	}
	syncCfg.Source = url
	return s.cfg.SaveSync(syncCfg)
}

// Watch runs Syncer.Run on a ticker until ctx is cancelled.
func (s *Syncer) Watch(ctx context.Context, interval time.Duration) error {
	if err := s.Run(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.Run(ctx) //nolint:errcheck
		}
	}
}

func (s *Syncer) fetchManifest(ctx context.Context, url string) (*Manifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}
