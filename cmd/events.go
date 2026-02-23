package cmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

var (
	eventsNetwork string
	eventsTopic   string
	eventsFrom    string
	eventsTo      string
	eventsCount   int
)

// Well-known event signatures for auto-decoding.
var knownEventTopics = map[string]string{
	computeEventTopic("Transfer(address,address,uint256)"):           "Transfer",
	computeEventTopic("Approval(address,address,uint256)"):           "Approval",
	computeEventTopic("OwnershipTransferred(address,address)"):       "OwnershipTransferred",
	computeEventTopic("Upgraded(address)"):                           "Upgraded",
	computeEventTopic("AdminChanged(address,address)"):               "AdminChanged",
	computeEventTopic("Initialized(uint8)"):                          "Initialized",
	computeEventTopic("Paused(address)"):                             "Paused",
	computeEventTopic("Unpaused(address)"):                           "Unpaused",
	computeEventTopic("RoleGranted(bytes32,address,address)"):        "RoleGranted",
	computeEventTopic("RoleRevoked(bytes32,address,address)"):        "RoleRevoked",
	computeEventTopic("Deposit(address,uint256)"):                    "Deposit",
	computeEventTopic("Withdrawal(address,uint256)"):                 "Withdrawal",
	computeEventTopic("Swap(address,uint256,uint256,uint256,uint256,address)"): "Swap",
}

func computeEventTopic(sig string) string {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(sig))
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

var eventsCmd = &cobra.Command{
	Use:   "events <contract>",
	Short: "Query event logs from a smart contract",
	Long: `Fetch and decode event logs emitted by a smart contract.

By default queries the last 1000 blocks. Use --from and --to to
specify a custom range.

Common events (Transfer, Approval, etc.) are auto-decoded.

Examples:
  w3cli events 0xUSDC --network ethereum
  w3cli events 0xUSDC --topic 0xddf252... --from 0x100 --to latest
  w3cli events 0xToken --count 20 --testnet`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contractAddr := args[0]

		chainName := eventsNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		client := chain.NewEVMClient(rpcURL)

		// Determine block range.
		fromBlock := normalizeBlockParam(eventsFrom)
		toBlock := normalizeBlockParam(eventsTo)
		if fromBlock == "" {
			// Default: last 1000 blocks.
			latest, err := client.GetBlockNumber()
			if err != nil {
				return fmt.Errorf("getting block number: %w", err)
			}
			start := latest - 1000
			if latest < 1000 {
				start = 0
			}
			fromBlock = fmt.Sprintf("0x%x", start)
		}
		if toBlock == "" {
			toBlock = "latest"
		}

		// Build topic filter.
		var topics []string
		if eventsTopic != "" {
			topics = append(topics, eventsTopic)
		}

		spin := ui.NewSpinner(fmt.Sprintf("Fetching events on %s...", c.DisplayName))
		spin.Start()
		logs, err := client.GetLogs(contractAddr, topics, fromBlock, toBlock)
		spin.Stop()
		if err != nil {
			return fmt.Errorf("querying events: %w", err)
		}

		if len(logs) == 0 {
			fmt.Println(ui.Info(fmt.Sprintf("No events found for %s in the specified range", ui.TruncateAddr(contractAddr))))
			return nil
		}

		// Limit display.
		displayCount := len(logs)
		if eventsCount > 0 && displayCount > eventsCount {
			logs = logs[len(logs)-eventsCount:]
			displayCount = eventsCount
		}

		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Events · %s (%s)", c.DisplayName, cfg.NetworkMode),
			[][2]string{
				{"Contract", ui.Addr(contractAddr)},
				{"Found", fmt.Sprintf("%d events (showing %d)", len(logs)+(displayCount-len(logs)), displayCount)},
				{"Block Range", fmt.Sprintf("%s → %s", fromBlock, toBlock)},
			}))
		fmt.Println()

		// Display each event.
		for i, log := range logs {
			blockNum := ""
			if bn, ok := new(big.Int).SetString(strings.TrimPrefix(log.BlockNumber, "0x"), 16); ok {
				blockNum = fmt.Sprintf("%d", bn.Uint64())
			}

			eventName := "Unknown"
			if len(log.Topics) > 0 {
				if name, ok := knownEventTopics[log.Topics[0]]; ok {
					eventName = name
				} else {
					eventName = ui.TruncateAddr(log.Topics[0])
				}
			}

			pairs := [][2]string{
				{"Event", ui.Val(eventName)},
				{"Block", blockNum},
				{"Tx", ui.Addr(log.TxHash)},
			}

			// Decode indexed topics.
			for j := 1; j < len(log.Topics); j++ {
				topicVal := log.Topics[j]
				// If it looks like an address (12 leading zero bytes).
				clean := strings.TrimPrefix(topicVal, "0x")
				if len(clean) == 64 && strings.HasPrefix(clean, "000000000000000000000000") {
					addr := clean[24:]
					allZero := true
					for _, ch := range addr {
						if ch != '0' {
							allZero = false
							break
						}
					}
					if !allZero {
						topicVal = ui.Addr("0x" + addr)
					}
				}
				pairs = append(pairs, [2]string{fmt.Sprintf("Topic[%d]", j), topicVal})
			}

			// Show data if present.
			if log.Data != "" && log.Data != "0x" {
				dataClean := strings.TrimPrefix(log.Data, "0x")
				if len(dataClean) <= 64 {
					// Single word — show as decimal too.
					if n, ok := new(big.Int).SetString(dataClean, 16); ok {
						pairs = append(pairs, [2]string{"Data", fmt.Sprintf("%s (%s)", log.Data, n.String())})
					} else {
						pairs = append(pairs, [2]string{"Data", log.Data})
					}
				} else {
					pairs = append(pairs, [2]string{"Data", log.Data[:min(74, len(log.Data))] + "..."})
				}
			}

			fmt.Println(ui.KeyValueBlock(fmt.Sprintf("Event #%d", i+1), pairs))
		}

		return nil
	},
}

// normalizeBlockParam converts a block number flag to an RPC-compatible value.
// Accepts hex ("0x1a"), decimal ("100"), named tags ("latest"), or empty string.
func normalizeBlockParam(s string) string {
	if s == "" || s == "latest" || s == "earliest" || s == "pending" || strings.HasPrefix(s, "0x") {
		return s
	}
	// Treat as decimal.
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return s // pass through, RPC will reject if invalid
	}
	return fmt.Sprintf("0x%x", n)
}

func init() {
	eventsCmd.Flags().StringVar(&eventsNetwork, "network", "", "chain (default: config)")
	eventsCmd.Flags().StringVar(&eventsTopic, "topic", "", "filter by event topic hash")
	eventsCmd.Flags().StringVar(&eventsFrom, "from", "", "start block (hex or decimal, default: latest-1000)")
	eventsCmd.Flags().StringVar(&eventsTo, "to", "", "end block (default: latest)")
	eventsCmd.Flags().IntVar(&eventsCount, "count", 10, "max events to display")
}
