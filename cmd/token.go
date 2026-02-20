package cmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/cobra"
)

// ── flag vars ─────────────────────────────────────────────────────────────────

var (
	tokenName     string
	tokenSymbol   string
	tokenDecimals uint8
	tokenSupply   string
	tokenNetwork  string
	tokenWallet   string

	// mint / burn
	tokenContract string
	tokenTo       string
	tokenAmount   string
)

// ── root token command ────────────────────────────────────────────────────────

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Deploy and manage ERC-20 tokens",
	Long: `Deploy, mint, and burn ERC-20 tokens.

Sub-commands:
  w3cli token create   — deploy a new mintable + burnable ERC-20
  w3cli token mint     — mint additional tokens (owner only)
  w3cli token burn     — burn tokens from your wallet`,
}

// ── token create ──────────────────────────────────────────────────────────────

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Deploy a new mintable + burnable ERC-20 token",
	Long: `Deploy a new ERC-20 token using OpenZeppelin v5.5 (Mintable + Burnable).

The deployer wallet becomes the owner and receives the entire initial supply.
Owner can mint more later with: w3cli token mint

Examples:
  w3cli token create --name "MyToken" --symbol MTK --decimals 18 --supply 1000000 --network base
  w3cli token create   (interactive wizard)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// ── Interactive prompts for missing flags ──────────────────────────────
		if tokenName == "" {
			tokenName = ui.PromptInput("Token name (e.g. MyToken)")
			if tokenName == "" {
				return fmt.Errorf("token name is required")
			}
		}
		if tokenSymbol == "" {
			tokenSymbol = ui.PromptInput("Token symbol (e.g. MTK)")
			if tokenSymbol == "" {
				return fmt.Errorf("token symbol is required")
			}
		}
		if !cmd.Flags().Changed("decimals") {
			raw := ui.PromptInput("Decimals [18]")
			if raw != "" {
				d, err := strconv.ParseUint(raw, 10, 8)
				if err != nil {
					return fmt.Errorf("invalid decimals: %w", err)
				}
				tokenDecimals = uint8(d)
			}
		}
		if tokenSupply == "" {
			tokenSupply = ui.PromptInput("Initial supply (token units, e.g. 1000000)")
			if tokenSupply == "" {
				return fmt.Errorf("supply is required")
			}
		}

		// ── Parse supply ──────────────────────────────────────────────────────
		supplyUnits, ok := new(big.Float).SetString(tokenSupply)
		if !ok {
			return fmt.Errorf("invalid supply %q", tokenSupply)
		}
		scale := new(big.Float).SetInt(
			new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tokenDecimals)), nil))
		supplyF := new(big.Float).Mul(supplyUnits, scale)
		supplyWei, _ := supplyF.Int(nil)

		// ── Resolve chain + wallet ────────────────────────────────────────────
		chainName := tokenNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}
		walletName := tokenWallet
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
			return fmt.Errorf("unknown chain %q — run `w3cli network list`", chainName)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}
		client := chain.NewEVMClient(rpcURL)

		// ── Build deploy data ─────────────────────────────────────────────────
		deployData, err := chain.BuildERC20DeployData(tokenName, tokenSymbol, tokenDecimals, supplyWei)
		if err != nil {
			return fmt.Errorf("building deploy data: %w", err)
		}
		deployHex := "0x" + hex.EncodeToString(deployData)

		// ── Fetch on-chain data for preview ──────────────────────────────────
		spin := ui.NewSpinner(fmt.Sprintf("Preparing deployment on %s...", c.DisplayName))
		spin.Start()

		gasPrice, err := client.GasPrice()
		if err != nil {
			spin.Stop()
			return err
		}
		gasLimit, err := client.EstimateGas(w.Address, "", deployHex, nil)
		if err != nil {
			gasLimit = 1_500_000
		}

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

		// ── Preview ───────────────────────────────────────────────────────────
		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Token Deploy Preview · %s (%s)", c.DisplayName, cfg.NetworkMode),
			[][2]string{
				{"Deployer", ui.Addr(w.Address)},
				{"Name", tokenName},
				{"Symbol", tokenSymbol},
				{"Decimals", fmt.Sprintf("%d", tokenDecimals)},
				{"Supply", fmt.Sprintf("%s %s", tokenSupply, tokenSymbol)},
				{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
				{"Gas Price", fmt.Sprintf("%d Gwei", toGwei(gasPrice))},
				{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
			}))

		if !ui.Confirm("Deploy this token?") {
			fmt.Println(ui.Meta("Cancelled."))
			return nil
		}

		// ── Sign + broadcast ──────────────────────────────────────────────────
		spin = ui.NewSpinner("Deploying token...")
		spin.Start()

		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(chainID),
			Nonce:     nonce,
			GasTipCap: gasPrice,
			GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
			Gas:       gasLimit,
			To:        nil, // contract creation
			Value:     big.NewInt(0),
			Data:      deployData,
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

		// ── Wait for receipt ──────────────────────────────────────────────────
		spin = ui.NewSpinner("Waiting for deployment confirmation...")
		spin.Start()
		receipt, err := client.WaitForReceipt(hash, 5*time.Minute)
		spin.Stop()
		if err != nil {
			return fmt.Errorf("deploy tx %s: %w", hash, err)
		}

		explorer := c.Explorer(cfg.NetworkMode)
		fmt.Println()
		fmt.Println(ui.KeyValueBlock("Token Deployed ✓", [][2]string{
			{"Contract", ui.Addr(receipt.ContractAddress)},
			{"Tx Hash", ui.Addr(hash)},
			{"Block", fmt.Sprintf("%d", receipt.BlockNumber)},
			{"Gas Used", fmt.Sprintf("%d", receipt.GasUsed)},
			{"Name", tokenName},
			{"Symbol", tokenSymbol},
			{"Decimals", fmt.Sprintf("%d", tokenDecimals)},
			{"Supply", tokenSupply + " " + tokenSymbol},
			{"Owner/Minter", ui.Addr(w.Address)},
			{"Explorer", explorer + "/address/" + receipt.ContractAddress},
		}))
		fmt.Println(ui.Hint(fmt.Sprintf(
			"Mint more: w3cli token mint --contract %s --to <addr> --amount <n> --network %s",
			receipt.ContractAddress, chainName)))

		// ── Auto-register in contract studio ──────────────────────────────────
		contractReg := newContractRegistry()
		if loadErr := contractReg.Load(); loadErr == nil {
			contractReg.Add(&contract.Entry{
				Name:       tokenSymbol,
				Network:    chainName,
				Address:    receipt.ContractAddress,
				ABI:        contract.GetBuiltinABI("w3token"),
				Kind:       "builtin",
				BuiltinID:  "w3token",
				Deployer:   w.Address,
				TxHash:     hash,
				DeployedAt: time.Now().UTC().Format(time.RFC3339),
			})
			if saveErr := contractReg.Save(); saveErr == nil {
				fmt.Println(ui.Success(fmt.Sprintf(
					"Registered in contract studio as %q — use: w3cli contract studio %s",
					tokenSymbol, tokenSymbol)))
			}
		}

		ui.OpenURL(explorer + "/address/" + receipt.ContractAddress)
		return nil
	},
}

// ── token mint ────────────────────────────────────────────────────────────────

var tokenMintCmd = &cobra.Command{
	Use:   "mint",
	Short: "Mint additional tokens (owner only)",
	Long: `Mint new tokens to an address. Caller must be the contract owner.

Examples:
  w3cli token mint --contract 0x... --to 0xRecipient --amount 5000 --network base`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if tokenContract == "" {
			tokenContract = ui.PromptInput("Token contract address")
		}
		if tokenTo == "" {
			tokenTo = ui.PromptInput("Mint to address or wallet name")
		}
		if tokenAmount == "" {
			tokenAmount = ui.PromptInput("Amount (token units)")
		}

		// Resolve chain + wallet.
		chainName := tokenNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}
		walletName := tokenWallet
		if walletName == "" {
			walletName = cfg.DefaultWallet
		}

		w, mgr, err := loadSigningWallet(walletName)
		if err != nil {
			return err
		}

		warnIfNoSession()

		toAddress, err := resolveToAddress(tokenTo, mgr)
		if err != nil {
			return err
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

		// ── Fetch on-chain data for preview ──────────────────────────────────
		spin := ui.NewSpinner(fmt.Sprintf("Preparing mint on %s...", c.DisplayName))
		spin.Start()

		decimals := 18
		if raw, err := client.CallContract(tokenContract, "0x313ce567"); err == nil && len(raw) >= 66 {
			if d, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16); ok {
				decimals = int(d.Int64())
			}
		}

		// Scale amount.
		amtF, ok := new(big.Float).SetString(tokenAmount)
		if !ok {
			spin.Stop()
			return fmt.Errorf("invalid amount %q", tokenAmount)
		}
		scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		amountWei, _ := new(big.Float).Mul(amtF, scale).Int(nil)

		calldata := chain.W3TokenMintCalldata(toAddress, amountWei)
		calldataHex := "0x" + hex.EncodeToString(calldata)

		gasPrice, err := client.GasPrice()
		if err != nil {
			spin.Stop()
			return err
		}
		gasLimit, err := client.EstimateGas(w.Address, tokenContract, calldataHex, nil)
		if err != nil {
			gasLimit = 80_000
		}
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

		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Mint Preview · %s (%s)", c.DisplayName, cfg.NetworkMode),
			[][2]string{
				{"Contract", ui.Addr(tokenContract)},
				{"Mint To", ui.Addr(toAddress)},
				{"Amount", tokenAmount + " tokens"},
				{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
				{"Gas Price", fmt.Sprintf("%d Gwei", toGwei(gasPrice))},
			}))

		if !ui.Confirm("Broadcast mint transaction?") {
			fmt.Println(ui.Meta("Cancelled."))
			return nil
		}

		contractAddr, err := parseAddress(tokenContract)
		if err != nil {
			return err
		}
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(chainID),
			Nonce:     nonce,
			GasTipCap: gasPrice,
			GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
			Gas:       gasLimit,
			To:        &contractAddr,
			Value:     big.NewInt(0),
			Data:      calldata,
		})

		ks := wallet.DefaultKeystore()
		signer := wallet.NewSigner(w, ks)

		spin = ui.NewSpinner("Minting tokens...")
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
		receipt, err := client.WaitForReceipt(hash, 3*time.Minute)
		spin.Stop()
		if err != nil {
			return fmt.Errorf("tx %s: %w", hash, err)
		}

		explorer := c.Explorer(cfg.NetworkMode)
		fmt.Println()
		fmt.Println(ui.KeyValueBlock("Mint Confirmed ✓", [][2]string{
			{"Hash", ui.Addr(hash)},
			{"Block", fmt.Sprintf("%d", receipt.BlockNumber)},
			{"Minted To", ui.Addr(toAddress)},
			{"Amount", tokenAmount + " tokens"},
			{"Explorer", explorer + "/tx/" + hash},
		}))
		ui.OpenURL(explorer + "/tx/" + hash)
		return nil
	},
}

// ── token burn ────────────────────────────────────────────────────────────────

var tokenBurnCmd = &cobra.Command{
	Use:   "burn",
	Short: "Burn tokens from your wallet",
	Long: `Burn (permanently destroy) tokens from your own wallet.

Examples:
  w3cli token burn --contract 0x... --amount 100 --network base`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if tokenContract == "" {
			tokenContract = ui.PromptInput("Token contract address")
		}
		if tokenAmount == "" {
			tokenAmount = ui.PromptInput("Amount to burn (token units)")
		}

		chainName := tokenNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}
		walletName := tokenWallet
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
			return fmt.Errorf("unknown chain %q — run `w3cli network list`", chainName)
		}
		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}
		client := chain.NewEVMClient(rpcURL)

		// ── Fetch on-chain data for preview ──────────────────────────────────
		spin := ui.NewSpinner(fmt.Sprintf("Preparing burn on %s...", c.DisplayName))
		spin.Start()

		decimals := 18
		if raw, err := client.CallContract(tokenContract, "0x313ce567"); err == nil && len(raw) >= 66 {
			if d, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16); ok {
				decimals = int(d.Int64())
			}
		}

		// Scale amount.
		amtF, ok := new(big.Float).SetString(tokenAmount)
		if !ok {
			spin.Stop()
			return fmt.Errorf("invalid amount %q", tokenAmount)
		}
		scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		amountWei, _ := new(big.Float).Mul(amtF, scale).Int(nil)

		calldata := chain.W3TokenBurnCalldata(amountWei)
		calldataHex := "0x" + hex.EncodeToString(calldata)

		gasPrice, err := client.GasPrice()
		if err != nil {
			spin.Stop()
			return err
		}
		gasLimit, err := client.EstimateGas(w.Address, tokenContract, calldataHex, nil)
		if err != nil {
			gasLimit = 60_000
		}
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

		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Burn Preview · %s (%s)", c.DisplayName, cfg.NetworkMode),
			[][2]string{
				{"Contract", ui.Addr(tokenContract)},
				{"Burn From", ui.Addr(w.Address)},
				{"Amount", tokenAmount + " tokens"},
				{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
				{"Gas Price", fmt.Sprintf("%d Gwei", toGwei(gasPrice))},
			}))

		if !ui.ConfirmDanger(fmt.Sprintf("Burn %s tokens? This is irreversible.", tokenAmount)) {
			fmt.Println(ui.Meta("Cancelled."))
			return nil
		}

		contractAddr, err := parseAddress(tokenContract)
		if err != nil {
			return err
		}
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(chainID),
			Nonce:     nonce,
			GasTipCap: gasPrice,
			GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
			Gas:       gasLimit,
			To:        &contractAddr,
			Value:     big.NewInt(0),
			Data:      calldata,
		})

		ks := wallet.DefaultKeystore()
		signer := wallet.NewSigner(w, ks)

		spin = ui.NewSpinner("Burning tokens...")
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
		receipt, err := client.WaitForReceipt(hash, 3*time.Minute)
		spin.Stop()
		if err != nil {
			return fmt.Errorf("tx %s: %w", hash, err)
		}

		explorer := c.Explorer(cfg.NetworkMode)
		fmt.Println()
		fmt.Println(ui.KeyValueBlock("Burn Confirmed ✓", [][2]string{
			{"Hash", ui.Addr(hash)},
			{"Block", fmt.Sprintf("%d", receipt.BlockNumber)},
			{"Burned", tokenAmount + " tokens"},
			{"Explorer", explorer + "/tx/" + hash},
		}))
		ui.OpenURL(explorer + "/tx/" + hash)
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseAddress(s string) (addr common.Address, err error) {
	s = strings.TrimPrefix(s, "0x")
	b, e := hex.DecodeString(s)
	if e != nil || len(b) != 20 {
		return addr, fmt.Errorf("invalid address %q", s)
	}
	copy(addr[:], b)
	return addr, nil
}

func init() {
	// create
	tokenCreateCmd.Flags().StringVar(&tokenName, "name", "", "token name (e.g. MyToken)")
	tokenCreateCmd.Flags().StringVar(&tokenSymbol, "symbol", "", "token symbol (e.g. MTK)")
	tokenCreateCmd.Flags().Uint8Var(&tokenDecimals, "decimals", 18, "decimal places (default 18)")
	tokenCreateCmd.Flags().StringVar(&tokenSupply, "supply", "", "initial supply in token units")
	tokenCreateCmd.Flags().StringVar(&tokenNetwork, "network", "", "chain (default: config)")
	tokenCreateCmd.Flags().StringVar(&tokenWallet, "wallet", "", "signing wallet (default: config)")

	// mint
	tokenMintCmd.Flags().StringVar(&tokenContract, "contract", "", "token contract address")
	tokenMintCmd.Flags().StringVar(&tokenTo, "to", "", "recipient address or wallet name")
	tokenMintCmd.Flags().StringVar(&tokenAmount, "amount", "", "amount to mint (token units)")
	tokenMintCmd.Flags().StringVar(&tokenNetwork, "network", "", "chain (default: config)")
	tokenMintCmd.Flags().StringVar(&tokenWallet, "wallet", "", "signing wallet (default: config)")

	// burn
	tokenBurnCmd.Flags().StringVar(&tokenContract, "contract", "", "token contract address")
	tokenBurnCmd.Flags().StringVar(&tokenAmount, "amount", "", "amount to burn (token units)")
	tokenBurnCmd.Flags().StringVar(&tokenNetwork, "network", "", "chain (default: config)")
	tokenBurnCmd.Flags().StringVar(&tokenWallet, "wallet", "", "signing wallet (default: config)")

	tokenCmd.AddCommand(tokenCreateCmd, tokenMintCmd, tokenBurnCmd)
}
