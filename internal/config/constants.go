package config

import "time"

// Gas limits used as EstimateGas fallbacks when the node cannot simulate the tx.
// These are conservative upper bounds; actual gas used will be lower.
const (
	GasLimitETHTransfer   = uint64(21_000)    // native ETH / SOL / SUI transfer
	GasLimitERC20Transfer = uint64(60_000)    // ERC-20 transfer or burn
	GasLimitERC20Mint     = uint64(80_000)    // ERC-20 mint
	GasLimitContractCall  = uint64(200_000)   // generic contract state-change call
	GasLimitTokenDeploy   = uint64(1_500_000) // full ERC-20 contract deployment
)

// Timeout constants used across cmd and future server packages.
const (
	RPCSelectTimeout = 10 * time.Second // BestEVM benchmark / RPC selection
	TxConfirmTimeout = 3 * time.Minute  // standard transaction confirmation wait
	TxDeployTimeout  = 5 * time.Minute  // contract deployment confirmation wait
)
