# w3cli

> The Web3 Power CLI -- 26 chains, Smart RPC, Contract Studio, Beautiful TUI

```
  ██╗    ██╗██████╗  ██████╗██╗     ██╗
  ██║    ██║╚════██╗██╔════╝██║     ██║
  ██║ █╗ ██║ █████╔╝██║     ██║     ██║
  ██║███╗██║ ╚═══██╗██║     ██║     ██║
  ╚███╔███╔╝██████╔╝╚██████╗███████╗██║
   ╚══╝╚══╝ ╚═════╝  ╚═════╝╚══════╝╚═╝
```

Check balances, send transactions, interact with smart contracts, deploy tokens, sign messages, and debug on-chain data -- across **26 blockchain networks** -- all from your terminal.

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
cp w3cli /usr/local/bin/w3cli
```

Requires Go 1.25+.

---

## Quick Start

```bash
# Interactive setup wizard
w3cli init

# Check a wallet balance
w3cli balance --wallet 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --network ethereum

# List all 26 supported chains
w3cli network list

# Watch live balance dashboard
w3cli balance --live

# Show active defaults
w3cli default
```

---

## Commands

### Wallet Management

```bash
w3cli wallet add mywallet 0x1234...             # Add watch-only wallet
w3cli wallet add deployer --key <private-key>    # Add signing wallet (stored in OS keychain)
w3cli wallet list                                # List wallets
w3cli wallet use mywallet                        # Set default wallet
w3cli wallet unlock                              # Cache keys for session (no repeated OS prompts)
w3cli wallet lock                                # Clear session cache
w3cli wallet remove mywallet                     # Remove wallet
```

### Balance

```bash
w3cli balance                                    # Default wallet + network
w3cli balance --wallet mywallet                  # Specific wallet
w3cli balance --network polygon                  # Specific network
w3cli balance --token 0xUSDC...                  # ERC-20 token balance
w3cli balance --live                             # Live auto-refresh dashboard
w3cli allbal --wallet 0x...                      # Scan all 24 EVM chains at once
```

### Send Transactions

```bash
w3cli send --to 0x... --value 0.1                # Send native token
w3cli send --to 0x... --value 100 --token 0xUSDC # Send ERC-20
w3cli send --gas fast                            # Gas speed: slow / standard / fast
```

### Token Deploy & Manage

```bash
w3cli token deploy MyToken MTK 18 1000000        # Deploy ERC-20 (name, symbol, decimals, supply)
w3cli token mint --contract 0x... --to 0x... --amount 500
w3cli token transfer --contract 0x... --to 0x... --amount 100
```

### Transactions

```bash
w3cli txs                                        # Last 10 transactions
w3cli txs --last 25                              # Last N transactions
w3cli tx 0xHASH                                  # Single transaction details
w3cli watch                                      # Stream live transactions
```

### Contract Studio

Interactive TUI for reading and writing smart contract functions.

```bash
w3cli contract add MyToken 0xADDR --abi ./abi.json  # Register with local ABI
w3cli contract add MyToken 0xADDR --fetch            # Auto-fetch ABI from explorer
w3cli contract list                                   # List registered contracts
w3cli contract studio MyToken                         # Interactive TUI
```

The studio auto-detects function types, shows parameter hints with examples, scales token amounts by decimals, and provides a full sign-preview-broadcast flow for write functions.

### Allowance & Approve

```bash
w3cli allowance --token 0xUSDC --owner 0x... --spender 0xRouter   # Check allowance
w3cli approve --token 0xUSDC --spender 0xRouter --amount max       # Approve spending
```

### Contract Calls

```bash
w3cli call 0xUSDC balanceOf 0xWALLET                               # Built-in ERC-20
w3cli call 0xContract "getPrice(address)" 0xToken                  # Custom signature
```

### Simulate Transactions

```bash
w3cli simulate --from 0x... --to 0x... --data 0xa9059cbb... --network ethereum
```

Dry-runs via `eth_call` -- reports success/revert with decoded reason and gas estimate.

### Nonce

```bash
w3cli nonce --wallet deployer --network ethereum --testnet
```

Shows confirmed + pending nonce. Warns if they differ (stuck transactions).

### Message Signing (EIP-191)

```bash
w3cli sign "Hello Web3" --wallet deployer        # Sign message
w3cli verify "Hello Web3" --sig 0x... --address 0x...  # Verify signature
```

### ENS Resolution

```bash
w3cli ens vitalik.eth                            # Name -> address
w3cli ens 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045  # Address -> name (reverse)
```

### Developer Utilities

```bash
# Unit conversion (no RPC needed)
w3cli convert 1.5 eth                            # -> gwei + wei
w3cli convert 50 gwei                            # -> eth + wei
w3cli convert 0xff                               # -> 255 (decimal)
w3cli convert 255 hex                            # -> 0xff

# Calldata decode / encode
w3cli decode 0xa9059cbb000000000000000000...     # Decode calldata -> method + args
w3cli encode "transfer(address,uint256)" 0xTo 1000000000000000000

# Keccak-256 hashing
w3cli keccak "transfer(address,uint256)"         # Full hash + 4-byte selector
w3cli keccak 0xdeadbeef                          # Hash raw hex bytes

# Function selectors
w3cli selector "transfer(address,uint256)"       # Signature -> 0xa9059cbb
w3cli selector 0xa9059cbb                        # Reverse lookup -> transfer

# Address checksum (EIP-55)
w3cli checksum 0xd8da6bf26964af9d7eed9e03e53415d37aa96045

# Contract inspection
w3cli code 0xUSDC --network ethereum             # Contract or EOA?
w3cli storage 0xContract 0 --network ethereum    # Read raw storage slot
w3cli events 0xContract --network ethereum       # Query event logs (auto-decodes Transfer, Approval, etc.)
```

### Network Management

```bash
w3cli network list                               # List all 26 chains
w3cli network use base                           # Set default network
w3cli network use base --testnet                 # Switch to testnet
w3cli allgas                                     # Gas prices across all chains
w3cli block --network ethereum                   # Latest block details
w3cli faucet                                     # Testnet faucet links
```

### RPC Management

```bash
w3cli rpc add base https://custom.rpc.url        # Add custom RPC
w3cli rpc list base                               # List RPCs for chain
w3cli rpc remove base https://old.rpc.url         # Remove custom RPC
w3cli rpc benchmark base                          # Benchmark all RPCs
w3cli rpc algorithm set fastest                   # fastest | round-robin | failover
```

### Config

```bash
w3cli config list                                 # Show full config
w3cli config set-default-network base             # Set default network
w3cli config set-default-wallet mywallet          # Set default wallet
w3cli config set-network-mode testnet             # Persist testnet mode
w3cli config set-key etherscan <key>              # Add provider API key
w3cli default                                     # Quick overview of active defaults
```

### Sync (Team Deployments)

```bash
w3cli sync set-source https://yourproject.com/deployments.json
w3cli sync run                                    # Fetch latest addresses + ABIs
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
| 25 | Solana | -- | Solana |
| 26 | SUI | -- | SUI |

All EVM chains have testnet support. Use `--testnet` or `w3cli config set-network-mode testnet` to switch.

---

## Smart RPC Selection

w3cli automatically picks the best RPC endpoint using three algorithms:

- **Fastest** (default) -- pings all RPCs in parallel, scores by latency + block recency, caches winner for 5 minutes
- **Round-robin** -- cycles through healthy endpoints evenly
- **Failover** -- always tries primary, falls back on failure

Stale nodes (>3 blocks behind best) are automatically discarded. Add custom RPCs with `w3cli rpc add`.

---

## Transaction History Providers

`w3cli txs` uses a priority-ordered provider chain. The first provider that returns data wins.

```
Etherscan V2 -> Alchemy -> Moralis -> BlockScout (free) -> Ankr (free) -> RPC fallback
```

Free providers work out-of-the-box. Add API keys for richer history:

```bash
w3cli config set-key etherscan <key>   # 14+ EVM chains (etherscan.io/apis)
w3cli config set-key alchemy <key>     # ETH, Base, Polygon, ARB, OP (alchemy.com)
w3cli config set-key moralis <key>     # 14+ chains (moralis.io)
w3cli config set-key ankr <key>        # Higher rate limits (ankr.com)
```

---

## Wallet Security

Private keys are stored in the **OS keychain** (macOS Keychain, Linux Secret Service / KWallet). Keys never touch disk in plaintext.

- `w3cli wallet unlock` caches keys in a session file (`~/Library/Caches/w3cli/session.json`) for the duration of your work so you don't get repeated OS keychain prompts
- `w3cli wallet lock` clears the session cache
- `W3CLI_KEY` env var overrides all key lookups (useful for CI/CD)

---

## Config Directory

```
~/.w3cli/
  config.json      # Networks, defaults, RPC algorithm, API keys
  wallets.json     # Wallet addresses + keychain references
  contracts.json   # Registered contracts + ABIs
  sync.json        # Auto-sync source + timestamp
```

Override with `--config <dir>` or `CHAIN_CONFIG_DIR` env var.

---

## Tech Stack

- **[Cobra](https://github.com/spf13/cobra)** -- CLI command routing
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** -- Interactive TUI (Contract Studio)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** -- Terminal colors & styling
- **[go-ethereum](https://github.com/ethereum/go-ethereum)** -- EVM RPC, ABI encoding, transaction signing
- **[99designs/keyring](https://github.com/99designs/keyring)** -- OS keychain storage

---

## Development

```bash
# Run all tests
go test ./...

# Run with race detector
go test ./... -race

# Build
go build -o w3cli .

# Build + install
go build -o w3cli . && cp w3cli /usr/local/bin/w3cli
```

---

## License

MIT
