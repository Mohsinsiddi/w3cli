package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/spf13/cobra"
)

var (
	signWallet string

	verifySig     string
	verifyAddress string
)

var signCmd = &cobra.Command{
	Use:   "sign <message>",
	Short: "Sign a message with EIP-191 (personal_sign)",
	Long: `Sign a plaintext message using EIP-191 personal_sign.

The message is prefixed with "\x19Ethereum Signed Message:\n<len>"
before being hashed and signed.

Examples:
  w3cli sign "hello world"
  w3cli sign "login nonce: 12345" --wallet myWallet`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[0]

		walletName := signWallet
		if walletName == "" {
			walletName = cfg.DefaultWallet
		}

		w, _, err := loadSigningWallet(walletName)
		if err != nil {
			return err
		}

		warnIfNoSession()

		ks := wallet.DefaultKeystore()
		sig, err := wallet.SignMessage(w, ks, []byte(message))
		if err != nil {
			return fmt.Errorf("signing failed: %w", err)
		}

		sigHex := "0x" + hex.EncodeToString(sig)

		fmt.Println(ui.KeyValueBlock("Message Signed", [][2]string{
			{"Signer", ui.Addr(w.Address)},
			{"Message", message},
			{"Signature", sigHex},
		}))
		fmt.Println(ui.Hint("Verify: w3cli verify \"" + message + "\" --sig " + sigHex + " --address " + w.Address))
		return nil
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify <message>",
	Short: "Verify an EIP-191 signed message",
	Long: `Verify who signed a message using EIP-191 personal_sign.

Recovers the signer address from the signature and compares it to
the expected address (if provided).

Examples:
  w3cli verify "hello world" --sig 0x... --address 0x...
  w3cli verify "hello world" --sig 0x...`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[0]

		if verifySig == "" {
			return fmt.Errorf("--sig is required — provide the hex signature")
		}

		sigBytes, err := hex.DecodeString(strings.TrimPrefix(verifySig, "0x"))
		if err != nil {
			return fmt.Errorf("invalid signature hex: %w", err)
		}

		recovered, err := wallet.VerifyMessage([]byte(message), sigBytes)
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		recoveredAddr := recovered.Hex()

		pairs := [][2]string{
			{"Message", message},
			{"Recovered Signer", ui.Addr(recoveredAddr)},
		}

		if verifyAddress != "" {
			if strings.EqualFold(recoveredAddr, verifyAddress) {
				pairs = append(pairs, [2]string{"Match", ui.Success("signature is valid — signer matches")})
			} else {
				pairs = append(pairs, [2]string{"Expected", ui.Addr(verifyAddress)})
				pairs = append(pairs, [2]string{"Match", ui.Err("signature does NOT match expected address")})
			}
		}

		fmt.Println(ui.KeyValueBlock("Signature Verification", pairs))
		return nil
	},
}

func init() {
	signCmd.Flags().StringVar(&signWallet, "wallet", "", "wallet name (default: config)")

	verifyCmd.Flags().StringVar(&verifySig, "sig", "", "hex signature to verify (required)")
	verifyCmd.Flags().StringVar(&verifyAddress, "address", "", "expected signer address (optional)")
}
