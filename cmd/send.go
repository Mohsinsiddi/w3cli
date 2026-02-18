package cmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if sendTo == "" {
			return fmt.Errorf("--to is required")
		}
		if sendValue == "" {
			return fmt.Errorf("--value is required")
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
		store := wallet.NewJSONStore(filepath.Join(cfg.Dir(), "wallets.json"))
		mgr := wallet.NewManager(wallet.WithStore(store))
		w, err := mgr.Get(walletName)
		if err != nil {
			return fmt.Errorf("wallet %q not found", walletName)
		}
		if w.Type != wallet.TypeSigning {
			return fmt.Errorf("wallet %q is watch-only — add it with --key to sign transactions", walletName)
		}

		// Resolve chain.
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

		// Parse value (ETH float → wei).
		valueWei, err := ethToWei(sendValue)
		if err != nil {
			return fmt.Errorf("invalid value %q: %w", sendValue, err)
		}

		// Gas speed multiplier.
		gasMult := big.NewInt(1)
		switch strings.ToLower(sendGas) {
		case "fast":
			gasMult = big.NewInt(2)
		case "slow":
			gasMult = big.NewInt(1)
		default: // standard
			gasMult = big.NewInt(1)
		}

		gasPrice, err := client.GasPrice()
		if err != nil {
			return err
		}
		gasPrice.Mul(gasPrice, gasMult)

		gasLimit, err := client.EstimateGas(w.Address, sendTo, "", valueWei)
		if err != nil {
			gasLimit = 21000
		}

		nonce, err := client.GetNonce(w.Address)
		if err != nil {
			return err
		}

		chainID, err := client.ChainID()
		if err != nil {
			return err
		}

		// Preview screen.
		usdCost := gasPrice.Uint64() * gasLimit
		fmt.Println(ui.KeyValueBlock("Transaction Preview", [][2]string{
			{"From", ui.Addr(w.Address)},
			{"To", ui.Addr(sendTo)},
			{"Value", sendValue + " " + c.NativeCurrency},
			{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
			{"Gas Price", fmt.Sprintf("%d Gwei", toGwei(gasPrice))},
			{"Est. Fee", fmt.Sprintf("~%d Wei", usdCost)},
			{"Network", c.DisplayName},
		}))

		if !ui.Confirm("Broadcast this transaction?") {
			fmt.Println(ui.Meta("Cancelled."))
			return nil
		}

		spin := ui.NewSpinner("Broadcasting transaction...")
		spin.Start()

		toAddr := common.HexToAddress(sendTo)
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &toAddr,
			Value:    valueWei,
		})

		ks := wallet.DefaultKeystore()
		signer := wallet.NewSigner(w, ks)
		raw, err := signer.SignTx(tx, big.NewInt(chainID))
		spin.Stop()
		if err != nil {
			return err
		}

		spin = ui.NewSpinner("Sending...")
		spin.Start()
		hash, err := client.SendRawTransaction("0x" + hex.EncodeToString(raw))
		spin.Stop()
		if err != nil {
			return err
		}

		explorer := c.Explorer(cfg.NetworkMode)
		fmt.Println(ui.Success("Transaction sent!"))
		fmt.Println(ui.Addr("Hash: " + hash))
		fmt.Println(ui.Meta(explorer + "/tx/" + hash))
		return nil
	},
}

func init() {
	sendCmd.Flags().StringVar(&sendTo, "to", "", "recipient address (required)")
	sendCmd.Flags().StringVar(&sendValue, "value", "", "amount to send (required)")
	sendCmd.Flags().StringVar(&sendToken, "token", "", "ERC-20 token contract address")
	sendCmd.Flags().StringVar(&sendGas, "gas", "standard", "gas speed: slow|standard|fast")
	sendCmd.Flags().StringVar(&sendNetwork, "network", "", "chain (default: config)")
	sendCmd.Flags().StringVar(&sendWallet, "wallet", "", "wallet name (default: config)")
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
