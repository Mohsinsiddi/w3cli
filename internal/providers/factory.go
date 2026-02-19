package providers

import (
	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/config"
)

// BuildRegistry assembles a provider Registry with all configured providers for the
// given chain, in priority order:
//
//  1. Etherscan V2  — if an "etherscan" key is configured
//  2. Alchemy       — if an "alchemy" key is configured
//  3. Moralis       — if a "moralis" key is configured
//  4. BlockScout    — if the chain has a free explorer API (always available)
//  5. Ankr          — always attempted (free tier, nil if chain unsupported)
//  6. RPC           — always last fallback
//
// Constructors that return nil (unsupported chain or missing key) are filtered out.
func BuildRegistry(chainName string, c *chain.Chain, mode string, rpcURL string, cfg *config.Config) *Registry {
	var ps []Provider

	// 1. Etherscan V2 (key-gated).
	if e := NewEtherscan(chainName, cfg.GetProviderKey("etherscan")); e != nil {
		ps = append(ps, e)
	}

	// 2. Alchemy (key-gated).
	if a := NewAlchemy(chainName, cfg.GetProviderKey("alchemy")); a != nil {
		ps = append(ps, a)
	}

	// 3. Moralis (key-gated).
	if m := NewMoralis(chainName, cfg.GetProviderKey("moralis")); m != nil {
		ps = append(ps, m)
	}

	// 4. BlockScout / chain explorer (free, no key needed).
	if explorerAPI := c.ExplorerAPIURL(mode); explorerAPI != "" {
		apiKey := cfg.GetExplorerAPIKey(chainName)
		ps = append(ps, NewBlockScout(explorerAPI, apiKey))
	}

	// 5. Ankr (free tier; nil if chain unsupported).
	if a := NewAnkr(chainName, cfg.GetProviderKey("ankr")); a != nil {
		ps = append(ps, a)
	}

	// 6. RPC fallback (always available).
	ps = append(ps, NewRPC(rpcURL))

	return New(ps...)
}
