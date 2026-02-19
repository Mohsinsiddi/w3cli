# w3cli ⚡

> The Web3 Power CLI — 26 chains · Smart RPC · Contract Studio · Beautiful TUI

```
  ██╗    ██╗██████╗  ██████╗██╗     ██╗
  ██║    ██║╚════██╗██╔════╝██║     ██║
  ██║ █╗ ██║ █████╔╝██║     ██║     ██║
  ██║███╗██║ ╚═══██╗██║     ██║     ██║
  ╚███╔███╔╝██████╔╝╚██████╗███████╗██║
   ╚══╝╚══╝ ╚═════╝  ╚═════╝╚══════╝╚═╝

     The Web3 Power CLI  ⚡  v1.0.0
  ✦ 26 chains  ✦ Smart RPC  ✦ Contract Studio
```

Check balances, query transactions, interact with smart contracts, sign and send transactions, and monitor wallets — across **26 blockchain networks** — all from your terminal.

---

## Install

### npm (recommended)

```bash
npm install -g w3cli
```

### Build from source

```bash
git clone https://github.com/Mohsinsiddi/w3cli.git
cd w3cli
go build -o w3cli .
```

Requires Go 1.25+.

---

## Quick Start

```bash
# Setup wizard
w3cli init

# Check a wallet balance
w3cli balance --wallet 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --network ethereum

# List all 26 supported chains
w3cli network list

# Watch live balance dashboard
w3cli balance --live
```

---

## Commands

### Network
```bash
w3cli network list                    # List all 26 chains
w3cli network use base                # Set default network
w3cli network use base --testnet      # Switch to testnet
```

### Wallet
```bash
w3cli wallet add mywallet 0x1234...   # Add watch-only wallet
w3cli wallet list                     # List wallets
w3cli wallet use mywallet             # Set default wallet
w3cli wallet remove mywallet          # Remove wallet
```

### Balance
```bash
w3cli balance                         # Default wallet + network
w3cli balance --wallet mywallet       # Specific wallet (name or address)
w3cli balance --network polygon       # Specific network
w3cli balance --token 0xUSDC...       # ERC-20 token balance
w3cli balance --live                  # Live auto-refresh dashboard
```

### Transactions
```bash
w3cli txs                             # Last 10 transactions
w3cli txs --last 25                   # Last N transactions
w3cli tx 0xHASH                       # Single transaction details
```

### Send
```bash
w3cli send --to 0x... --value 0.1     # Send native token
w3cli send --to 0x... --value 100 --token 0xUSDC  # Send ERC-20
w3cli send --gas fast                 # Gas speed: slow / standard / fast
```

### RPC Management
```bash
w3cli rpc add base https://custom.rpc.url    # Add custom RPC
w3cli rpc list base                          # List RPCs for chain
w3cli rpc benchmark base                     # Benchmark all RPCs
w3cli rpc algorithm set fastest             # fastest | round-robin | failover
```

### Contract Studio
```bash
w3cli contract add MyToken 0xADDR --abi ./abi.json   # Register contract
w3cli contract add MyToken 0xADDR --fetch            # Auto-fetch ABI
w3cli contract list                                   # List contracts
w3cli contract call MyToken balanceOf 0xWALLET        # Read function
w3cli contract send MyToken transfer 0xTO 1000        # Write function
w3cli contract studio MyToken                         # Interactive TUI
```

### Config
```bash
w3cli config list                          # Show full config
w3cli config list --verbose                # Show full config + raw JSON (includes saved keys)
w3cli config set-default-network base      # Set default network
w3cli config set-default-wallet mywallet   # Set default wallet
w3cli config set-network-mode testnet      # Persist testnet as default mode
w3cli config set-explorer-key <key>        # Set BlockScout / Etherscan explorer key
w3cli config set-rpc base https://my.rpc   # Add a custom RPC for a chain
```

### Provider API Keys (Transaction History)

Unlock richer transaction history by adding provider API keys. Free providers work out-of-the-box; keyed providers give full indexed history.

```bash
# Key-gated providers — activate when key is set
w3cli config set-key etherscan <key>   # Etherscan V2 — 14+ EVM chains (etherscan.io/apis)
w3cli config set-key alchemy <key>     # Alchemy — ETH, Base, Polygon, ARB, OP (alchemy.com)
w3cli config set-key moralis <key>     # Moralis — 14+ chains incl. BNB, FTM (moralis.io)
w3cli config set-key ankr <key>        # Ankr — higher rate limits (ankr.com/rpc/apps)

# Verify saved keys
w3cli config list --verbose
```

Provider priority order (first that returns data wins):
```
Etherscan → Alchemy → Moralis → BlockScout (free) → Ankr (free) → RPC (last 200 blocks)
```

See [PROVIDERS.md](./PROVIDERS.md) for full per-chain coverage details.

### Sync
```bash
w3cli sync set-source https://yourproject.com/deployments.json
w3cli sync run                         # Fetch latest addresses + ABIs
```

### Watch
```bash
w3cli watch                            # Monitor wallet across chains
w3cli watch --network base             # Single chain
```

---

## Supported Chains

| # | Chain | Chain ID | Type |
|---|-------|----------|------|
| 1 | Ethereum | 1 | EVM |
| 2 | Base | 8453 | EVM |
| 3 | Polygon | 137 | EVM |
| 4 | Arbitrum | 42161 | EVM |
| 5 | Optimism | 10 | EVM |
| 6 | BNB Chain | 56 | EVM |
| 7 | Avalanche | 43114 | EVM |
| 8 | Fantom | 250 | EVM |
| 9 | Linea | 59144 | EVM |
| 10 | zkSync Era | 324 | EVM |
| 11 | Scroll | 534352 | EVM |
| 12 | Mantle | 5000 | EVM |
| 13 | Celo | 42220 | EVM |
| 14 | Gnosis | 100 | EVM |
| 15 | Blast | 81457 | EVM |
| 16 | Mode | 34443 | EVM |
| 17 | Zora | 7777777 | EVM |
| 18 | Moonbeam | 1284 | EVM |
| 19 | Cronos | 25 | EVM |
| 20 | Klaytn | 8217 | EVM |
| 21 | Aurora | 1313161554 | EVM |
| 22 | Polygon zkEVM | 1101 | EVM |
| 23 | Hyperliquid EVM | 999 | EVM |
| 24 | Boba Network | 288 | EVM |
| 25 | Solana | — | Solana |
| 26 | SUI | — | SUI |

---

## Smart RPC Selection

w3cli automatically picks the best RPC endpoint using three algorithms:

- **Fastest** (default) — pings all RPCs in parallel, scores by latency + block recency, caches winner for 5 minutes
- **Round-robin** — cycles through healthy endpoints evenly
- **Failover** — always tries primary, falls back on failure

Stale nodes (>3 blocks behind best) are automatically discarded.

---

## Config

Config is stored at `~/.w3cli/` (override with `CHAIN_CONFIG_DIR` env var).

```
~/.w3cli/
├── config.json      # Networks, defaults, RPC algorithm
├── wallets.json     # Wallet addresses
├── contracts.json   # Registered contracts + ABIs
└── sync.json        # Auto-sync source + timestamp
```

---

## Tech Stack

- **[Cobra](https://github.com/spf13/cobra)** — CLI command routing
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** — Interactive TUI
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** — Colors & styling
- **[go-ethereum](https://github.com/ethereum/go-ethereum)** — EVM RPC & ABI
- **[99designs/keyring](https://github.com/99designs/keyring)** — OS keychain storage

---

## Development

```bash
# Run all tests
go test ./... -race

# Build
go build -o w3cli .

# Build all platforms
VERSION=1.0.0 ./scripts/build-all.sh

# Release
VERSION=1.0.0 ./scripts/release.sh
```

---

## License

MIT © [siddi_404](https://github.com/Mohsinsiddi)
# w3cli
# w3cli
# w3cli
