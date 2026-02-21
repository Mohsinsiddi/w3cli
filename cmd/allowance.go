package cmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/config"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/cobra"
)

var (
	allowanceToken   string
	allowanceOwner   string
	allowanceSpender string
	allowanceNetwork string

	approveToken   string
	approveSpender string
	approveAmount  string
	approveWallet  string
	approveNetwork string
)

var allowanceCmd = &cobra.Command{
	Use:   "allowance",
	Short: "Check ERC-20 token allowance (owner → spender)",
	Long: `Query how many tokens an owner has approved a spender to use.

Examples:
  w3cli allowance --token 0xUSDC --owner 0xOwner --spender 0xDEX
  w3cli allowance --token 0xUSDC --owner myWallet --spender 0xRouter --network ethereum`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if allowanceToken == "" {
			return fmt.Errorf("--token is required — provide the ERC-20 contract address")
		}
		if allowanceSpender == "" {
			return fmt.Errorf("--spender is required — provide the spender address")
		}

		// Resolve owner.
		owner := allowanceOwner
		chainName := allowanceNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}
		if owner == "" {
			addr, _, err := resolveWalletAndChain("", chainName)
			if err != nil {
				return fmt.Errorf("--owner is required or set a default wallet")
			}
			owner = addr
		} else if !strings.HasPrefix(owner, "0x") {
			mgr := newWalletManager()
			w, err := mgr.Get(owner)
			if err != nil {
				return fmt.Errorf("wallet %q not found", owner)
			}
			owner = w.Address
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

		spin := ui.NewSpinner("Querying allowance...")
		spin.Start()

		allowance, err := client.GetAllowance(allowanceToken, owner, allowanceSpender)
		if err != nil {
			spin.Stop()
			return fmt.Errorf("querying allowance: %w", err)
		}

		// Fetch decimals for formatting.
		decimals := 18
		if raw, err := client.CallContract(allowanceToken, "0x313ce567"); err == nil && len(raw) >= 66 {
			if d, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16); ok {
				decimals = int(d.Int64())
			}
		}
		spin.Stop()

		formatted := formatTokenAmount(allowance, decimals)

		fmt.Println(ui.KeyValueBlock("ERC-20 Allowance", [][2]string{
			{"Token", ui.Addr(allowanceToken)},
			{"Owner", ui.Addr(owner)},
			{"Spender", ui.Addr(allowanceSpender)},
			{"Allowance", ui.Val(formatted)},
			{"Raw", allowance.String()},
			{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
		}))
		return nil
	},
}

var approveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve ERC-20 token spending for a spender",
	Long: `Approve a spender to use a specific amount of your ERC-20 tokens.

Examples:
  w3cli approve --token 0xUSDC --spender 0xDEX --amount 1000
  w3cli approve --token 0xUSDC --spender 0xRouter --amount 1000 --wallet myWallet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if approveToken == "" {
			return fmt.Errorf("--token is required")
		}
		if approveSpender == "" {
			return fmt.Errorf("--spender is required")
		}
		if approveAmount == "" {
			return fmt.Errorf("--amount is required")
		}

		chainName := approveNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}
		walletName := approveWallet
		if walletName == "" {
			walletName = cfg.DefaultWallet
		}

		w, _, err := loadSigningWallet(walletName)
		if err != nil {
			return err
		}

		warnIfNoSession()

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

		spin := ui.NewSpinner(fmt.Sprintf("Preparing approve on %s...", c.DisplayName))
		spin.Start()

		// Fetch decimals.
		decimals := 18
		if raw, err := client.CallContract(approveToken, "0x313ce567"); err == nil && len(raw) >= 66 {
			if d, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16); ok {
				decimals = int(d.Int64())
			}
		}

		// Scale amount by decimals.
		amt, ok := new(big.Float).SetString(approveAmount)
		if !ok {
			spin.Stop()
			return fmt.Errorf("invalid amount %q", approveAmount)
		}
		scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		amountScaled, _ := new(big.Float).Mul(amt, scale).Int(nil)

		// Build approve(address,uint256) calldata.
		// Selector: 0x095ea7b3
		spenderBytes, _ := hex.DecodeString(strings.TrimPrefix(approveSpender, "0x"))
		addrWord := make([]byte, 32)
		copy(addrWord[12:], spenderBytes)
		amtWord := make([]byte, 32)
		ab := amountScaled.Bytes()
		copy(amtWord[32-len(ab):], ab)

		calldata := make([]byte, 0, 68)
		calldata = append(calldata, 0x09, 0x5e, 0xa7, 0xb3)
		calldata = append(calldata, addrWord...)
		calldata = append(calldata, amtWord...)
		calldataHex := "0x" + hex.EncodeToString(calldata)

		gasPrice, err := client.GasPrice()
		if err != nil {
			spin.Stop()
			return err
		}

		chainID, err := client.ChainID()
		if err != nil {
			spin.Stop()
			return err
		}

		nonce, err := client.GetPendingNonce(w.Address)
		if err != nil {
			spin.Stop()
			return err
		}

		gasLimit, err := client.EstimateGas(w.Address, approveToken, calldataHex, nil)
		if err != nil {
			gasLimit = config.GasLimitERC20Transfer
		}
		spin.Stop()

		explorer := c.Explorer(cfg.NetworkMode)

		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Approve Preview · %s (%s)", c.DisplayName, cfg.NetworkMode),
			[][2]string{
				{"From", ui.Addr(w.Address)},
				{"Token", ui.Addr(approveToken)},
				{"Spender", ui.Addr(approveSpender)},
				{"Amount", fmt.Sprintf("%s (decimals: %d)", approveAmount, decimals)},
				{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
				{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
			}))

		if !ui.Confirm("Broadcast this approve transaction?") {
			fmt.Println(ui.Meta("Cancelled."))
			return nil
		}

		spin = ui.NewSpinner("Broadcasting approve...")
		spin.Start()

		tokenAddr := common.HexToAddress(approveToken)
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(chainID),
			Nonce:     nonce,
			GasTipCap: gasPrice,
			GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
			Gas:       gasLimit,
			To:        &tokenAddr,
			Value:     big.NewInt(0),
			Data:      calldata,
		})

		ks := wallet.DefaultKeystore()
		signer := wallet.NewSigner(w, ks)
		raw, err := signer.SignTx(tx, big.NewInt(chainID))
		if err != nil {
			spin.Stop()
			return err
		}

		hash, err := client.SendRawTransaction("0x" + hex.EncodeToString(raw))
		spin.Stop()
		if err != nil {
			return err
		}

		spin = ui.NewSpinner("Waiting for confirmation...")
		spin.Start()
		receipt, err := client.WaitForReceipt(hash, config.TxConfirmTimeout)
		spin.Stop()
		if err != nil {
			return fmt.Errorf("tx %s: %w", hash, err)
		}

		fmt.Println()
		fmt.Println(ui.KeyValueBlock("Approve Confirmed ✓", [][2]string{
			{"Hash", ui.Addr(hash)},
			{"Block", fmt.Sprintf("%d", receipt.BlockNumber)},
			{"Gas Used", fmt.Sprintf("%d", receipt.GasUsed)},
			{"Explorer", explorer + "/tx/" + hash},
		}))
		ui.OpenURL(explorer + "/tx/" + hash)
		return nil
	},
}

// formatTokenAmount formats a raw token amount with the given decimals.
func formatTokenAmount(raw *big.Int, decimals int) string {
	if decimals <= 0 {
		return raw.String()
	}
	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	f := new(big.Float).SetInt(raw)
	f.Quo(f, new(big.Float).SetInt(div))
	return f.Text('f', decimals)
}

func init() {
	allowanceCmd.Flags().StringVar(&allowanceToken, "token", "", "ERC-20 token address (required)")
	allowanceCmd.Flags().StringVar(&allowanceOwner, "owner", "", "owner address or wallet name")
	allowanceCmd.Flags().StringVar(&allowanceSpender, "spender", "", "spender address (required)")
	allowanceCmd.Flags().StringVar(&allowanceNetwork, "network", "", "chain (default: config)")

	approveCmd.Flags().StringVar(&approveToken, "token", "", "ERC-20 token address (required)")
	approveCmd.Flags().StringVar(&approveSpender, "spender", "", "spender address (required)")
	approveCmd.Flags().StringVar(&approveAmount, "amount", "", "amount to approve (required)")
	approveCmd.Flags().StringVar(&approveWallet, "wallet", "", "wallet name (default: config)")
	approveCmd.Flags().StringVar(&approveNetwork, "network", "", "chain (default: config)")
}
