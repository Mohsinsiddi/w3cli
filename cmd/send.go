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
	sendTo      string
	sendValue   string
	sendToken   string
	sendGas     string
	sendNetwork string
	sendWallet  string
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send native tokens or ERC-20 tokens",
	Long: `Send native tokens or ERC-20 tokens to an address or wallet name.

Uses the configured network mode (mainnet/testnet) by default.
Override per-call with --testnet or --mainnet.

Examples:
  w3cli send --to 0x... --value 0.1
  w3cli send --to myOtherWallet --value 0.001 --network ethereum
  w3cli send --to 0x... --value 100 --token 0xUSDC --network base
  w3cli send --to 0x... --value 0.1 --testnet --gas fast`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if sendTo == "" {
			return fmt.Errorf("--to is required — specify a recipient address or wallet name")
		}
		if sendValue == "" {
			return fmt.Errorf("--value is required — specify the amount to send (e.g. --value 0.1)")
		}

		chainName := sendNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}
		walletName := sendWallet
		if walletName == "" {
			walletName = cfg.DefaultWallet
		}

		// Load the signing wallet.
		w, mgr, err := loadSigningWallet(walletName)
		if err != nil {
			return err
		}

		warnIfNoSession()

		// Resolve --to as wallet name or raw address.
		toAddress, err := resolveToAddress(sendTo, mgr)
		if err != nil {
			return err
		}

		// Resolve chain.
		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		client := chain.NewEVMClient(rpcURL)

		// Gas speed multiplier.
		gasMult := big.NewInt(1)
		if strings.ToLower(sendGas) == "fast" {
			gasMult = big.NewInt(2)
		}

		// ── Fetch on-chain data needed for the preview ────────────────────────
		spin := ui.NewSpinner(fmt.Sprintf("Preparing transaction on %s...", c.DisplayName))
		spin.Start()

		gasPrice, err := client.GasPrice()
		if err != nil {
			spin.Stop()
			return err
		}
		gasPrice.Mul(gasPrice, gasMult)

		chainID, err := client.ChainID()
		if err != nil {
			spin.Stop()
			return err
		}

		nonce, err := client.GetNonce(w.Address)
		if err != nil {
			spin.Stop()
			return err
		}
		spin.Stop()

		ks := wallet.DefaultKeystore()
		signer := wallet.NewSigner(w, ks)
		explorer := c.Explorer(cfg.NetworkMode)

		// ── ERC-20 send ──────────────────────────────────────────────────────
		if sendToken != "" {
			return runTokenSend(client, signer, w.Address, toAddress, sendToken,
				sendValue, gasPrice, nonce, chainID, c, explorer)
		}

		// ── Native ETH send (EIP-1559) ────────────────────────────────────────
		valueWei, err := ethToWei(sendValue)
		if err != nil {
			return fmt.Errorf("invalid value %q: %w", sendValue, err)
		}

		gasLimit, err := client.EstimateGas(w.Address, toAddress, "", valueWei)
		if err != nil {
			gasLimit = config.GasLimitETHTransfer
		}

		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Transaction Preview · %s (%s)", c.DisplayName, cfg.NetworkMode),
			[][2]string{
				{"From", ui.Addr(w.Address)},
				{"To", ui.Addr(toAddress)},
				{"Value", sendValue + " " + c.NativeCurrency},
				{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
				{"Gas Price", fmt.Sprintf("%d Gwei", toGwei(gasPrice))},
				{"Gas Speed", sendGas},
				{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
			}))

		if !ui.Confirm("Broadcast this transaction?") {
			fmt.Println(ui.Meta("Cancelled."))
			return nil
		}

		spin = ui.NewSpinner(fmt.Sprintf("Broadcasting on %s...", c.DisplayName))
		spin.Start()

		toAddr := common.HexToAddress(toAddress)
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(chainID),
			Nonce:     nonce,
			GasTipCap: gasPrice,
			GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
			Gas:       gasLimit,
			To:        &toAddr,
			Value:     valueWei,
		})

		raw, err := signer.SignTx(tx, big.NewInt(chainID))
		spin.Stop()
		if err != nil {
			return err
		}

		spin = ui.NewSpinner("Sending transaction...")
		spin.Start()
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
		fmt.Println(ui.KeyValueBlock("Transaction Confirmed ✓", [][2]string{
			{"Hash", ui.Addr(hash)},
			{"Block", fmt.Sprintf("%d", receipt.BlockNumber)},
			{"Gas Used", fmt.Sprintf("%d", receipt.GasUsed)},
			{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
			{"Explorer", explorer + "/tx/" + hash},
		}))
		fmt.Println(ui.Hint("Track: w3cli tx " + hash + " --network " + chainName))
		ui.OpenURL(explorer + "/tx/" + hash)
		return nil
	},
}

// runTokenSend handles ERC-20 token sends.
func runTokenSend(client *chain.EVMClient, signer *wallet.Signer, from, to, tokenAddr, valueStr string,
	gasPrice *big.Int, nonce uint64, chainID int64, c *chain.Chain, explorer string) error {

	// ── Prepare: fetch token info + estimate gas ─────────────────────────
	spin := ui.NewSpinner(fmt.Sprintf("Preparing token transfer on %s...", c.DisplayName))
	spin.Start()

	decimals := 18
	if raw, err := client.CallContract(tokenAddr, "0x313ce567"); err == nil && len(raw) >= 66 {
		if d, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16); ok {
			decimals = int(d.Int64())
		}
	}

	// Scale value by 10^decimals.
	amt, ok := new(big.Float).SetString(valueStr)
	if !ok {
		spin.Stop()
		return fmt.Errorf("invalid value %q", valueStr)
	}
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountF := new(big.Float).Mul(amt, scale)
	amountWei, _ := amountF.Int(nil)

	// Build transfer(address,uint256) calldata.
	// selector: 0xa9059cbb
	toBytes, _ := hex.DecodeString(strings.TrimPrefix(to, "0x"))
	addrWord := make([]byte, 32)
	copy(addrWord[12:], toBytes)
	amtWord := make([]byte, 32)
	ab := amountWei.Bytes()
	copy(amtWord[32-len(ab):], ab)

	calldata := make([]byte, 0, 68)
	calldata = append(calldata, 0xa9, 0x05, 0x9c, 0xbb)
	calldata = append(calldata, addrWord...)
	calldata = append(calldata, amtWord...)
	calldataHex := "0x" + hex.EncodeToString(calldata)

	gasLimit, err := client.EstimateGas(from, tokenAddr, calldataHex, nil)
	if err != nil {
		gasLimit = config.GasLimitERC20Transfer
	}
	spin.Stop()

	fmt.Println(ui.KeyValueBlock(
		fmt.Sprintf("ERC-20 Send Preview · %s (%s)", c.DisplayName, cfg.NetworkMode),
		[][2]string{
			{"From", ui.Addr(from)},
			{"To", ui.Addr(to)},
			{"Token", ui.Addr(tokenAddr)},
			{"Amount", fmt.Sprintf("%s (decimals: %d)", valueStr, decimals)},
			{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
			{"Gas Price", fmt.Sprintf("%d Gwei", toGwei(gasPrice))},
			{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
		}))

	if !ui.Confirm("Broadcast this token transfer?") {
		fmt.Println(ui.Meta("Cancelled."))
		return nil
	}

	tokenAddrCommon := common.HexToAddress(tokenAddr)
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(chainID),
		Nonce:     nonce,
		GasTipCap: gasPrice,
		GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
		Gas:       gasLimit,
		To:        &tokenAddrCommon,
		Value:     big.NewInt(0),
		Data:      calldata,
	})

	spin = ui.NewSpinner("Signing & sending token transfer...")
	spin.Start()
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
	fmt.Println(ui.KeyValueBlock("Token Transfer Confirmed ✓", [][2]string{
		{"Hash", ui.Addr(hash)},
		{"Block", fmt.Sprintf("%d", receipt.BlockNumber)},
		{"Gas Used", fmt.Sprintf("%d", receipt.GasUsed)},
		{"Explorer", explorer + "/tx/" + hash},
	}))
	ui.OpenURL(explorer + "/tx/" + hash)
	return nil
}

func init() {
	sendCmd.Flags().StringVar(&sendTo, "to", "", "recipient address or wallet name (required)")
	sendCmd.Flags().StringVar(&sendValue, "value", "", "amount to send (required)")
	sendCmd.Flags().StringVar(&sendToken, "token", "", "ERC-20 token contract address")
	sendCmd.Flags().StringVar(&sendGas, "gas", "standard", "gas speed: slow|standard|fast")
	sendCmd.Flags().StringVar(&sendNetwork, "network", "", "chain (default: config)")
	sendCmd.Flags().StringVar(&sendWallet, "wallet", "", "wallet name (default: config)")
}

// resolveToAddress returns a hex address from a wallet name or raw 0x address.
func resolveToAddress(s string, mgr *wallet.Manager) (string, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return s, nil
	}
	w, err := mgr.Get(s)
	if err != nil {
		return "", fmt.Errorf("recipient %q is not an 0x address and no wallet named %q was found", s, s)
	}
	return w.Address, nil
}

func ethToWei(ethStr string) (*big.Int, error) {
	f, ok := new(big.Float).SetString(ethStr)
	if !ok {
		return nil, fmt.Errorf("invalid ETH value: %s", ethStr)
	}
	weiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	weiFloat := new(big.Float).Mul(f, weiPerETH)
	wei, _ := weiFloat.Int(nil)
	return wei, nil
}

func toGwei(wei *big.Int) uint64 {
	gwei := new(big.Int).Div(wei, big.NewInt(1e9))
	return gwei.Uint64()
}
