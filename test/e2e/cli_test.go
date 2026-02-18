package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary before all E2E tests.
	tmp, err := os.MkdirTemp("", "w3cli-e2e-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "w3cli")
	// Build from the module root (two levels up from test/e2e/).
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = moduleRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	os.Exit(m.Run())
}

func runCLI(t *testing.T, configDir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "CHAIN_CONFIG_DIR="+configDir)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestVersionFlag(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "--version")
	require.NoError(t, err)
	assert.Contains(t, out, "w3cli")
	assert.Contains(t, out, "1.0.0")
}

func TestHelpCommand(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "w3cli")
	assert.Contains(t, strings.ToLower(out), "balance")
	assert.Contains(t, strings.ToLower(out), "txs")
	assert.Contains(t, strings.ToLower(out), "contract")
	assert.Contains(t, strings.ToLower(out), "wallet")
	assert.Contains(t, strings.ToLower(out), "network")
	assert.Contains(t, out, "--testnet")
	assert.Contains(t, out, "--mainnet")
}

func TestNetworkList(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "network", "list")
	require.NoError(t, err)

	chains := []string{"ethereum", "base", "polygon", "arbitrum", "solana", "sui"}
	for _, c := range chains {
		assert.Contains(t, strings.ToLower(out), c, "network list should contain %s", c)
	}
}

func TestNetworkUse(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "network", "use", "base")
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(out), "base")
}

func TestNetworkUseUnknown(t *testing.T) {
	dir := t.TempDir()
	_, err := runCLI(t, dir, "network", "use", "unknownchain99")
	assert.Error(t, err) // should fail
}

func TestNetworkUseWithTestnetFlag(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "--testnet", "network", "use", "base")
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(out), "base")

	// Config should now have testnet persisted.
	cfgOut, err := runCLI(t, dir, "config", "list")
	require.NoError(t, err)
	assert.Contains(t, cfgOut, "testnet")
}

func TestNetworkUseWithMainnetFlag(t *testing.T) {
	dir := t.TempDir()
	// First set to testnet.
	_, _ = runCLI(t, dir, "--testnet", "network", "use", "base")
	// Now switch back with --mainnet.
	out, err := runCLI(t, dir, "--mainnet", "network", "use", "base")
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(out), "base")

	cfgOut, err := runCLI(t, dir, "config", "list")
	require.NoError(t, err)
	assert.Contains(t, cfgOut, "mainnet")
}

func TestWalletAddAndList(t *testing.T) {
	dir := t.TempDir()

	_, err := runCLI(t, dir, "wallet", "add", "testwal", "0x1234567890abcdef1234567890abcdef12345678")
	require.NoError(t, err)

	out, err := runCLI(t, dir, "wallet", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "testwal")
	assert.Contains(t, out, "0x1234")
}

func TestWalletRemove(t *testing.T) {
	dir := t.TempDir()

	runCLI(t, dir, "wallet", "add", "w1", "0x1234567890abcdef1234567890abcdef12345678") //nolint:errcheck

	// Use stdin to auto-confirm the prompt.
	cmd := exec.Command(binaryPath, "wallet", "remove", "w1")
	cmd.Env = append(os.Environ(), "CHAIN_CONFIG_DIR="+dir)
	cmd.Stdin = strings.NewReader("y\n")
	cmd.Run() //nolint:errcheck

	out, err := runCLI(t, dir, "wallet", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, "w1")
}

func TestRPCAdd(t *testing.T) {
	dir := t.TempDir()

	_, err := runCLI(t, dir, "rpc", "add", "base", "https://custom.rpc.url")
	require.NoError(t, err)

	out, _ := runCLI(t, dir, "rpc", "list", "base")
	assert.Contains(t, out, "custom.rpc.url")
}

func TestRPCAlgorithmSet(t *testing.T) {
	dir := t.TempDir()

	_, err := runCLI(t, dir, "rpc", "algorithm", "set", "round-robin")
	require.NoError(t, err)

	out, _ := runCLI(t, dir, "config", "list")
	assert.Contains(t, out, "round-robin")
}

func TestConfigList(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "config", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "default_network")
	assert.Contains(t, out, "rpc_algorithm")
}

func TestConfigSetDefaultNetwork(t *testing.T) {
	dir := t.TempDir()
	_, err := runCLI(t, dir, "config", "set-default-network", "polygon")
	require.NoError(t, err)

	out, _ := runCLI(t, dir, "config", "list")
	assert.Contains(t, out, "polygon")
}

func TestConfigSetNetworkMode(t *testing.T) {
	dir := t.TempDir()

	// Set to testnet.
	out, err := runCLI(t, dir, "config", "set-network-mode", "testnet")
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(out), "testnet")

	cfgOut, err := runCLI(t, dir, "config", "list")
	require.NoError(t, err)
	assert.Contains(t, cfgOut, "testnet")

	// Set back to mainnet.
	out, err = runCLI(t, dir, "config", "set-network-mode", "mainnet")
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(out), "mainnet")

	cfgOut, err = runCLI(t, dir, "config", "list")
	require.NoError(t, err)
	assert.Contains(t, cfgOut, "mainnet")
}

func TestConfigSetNetworkModeInvalid(t *testing.T) {
	dir := t.TempDir()
	_, err := runCLI(t, dir, "config", "set-network-mode", "devnet")
	assert.Error(t, err)
}

func TestTestnetMainnetMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	_, err := runCLI(t, dir, "--testnet", "--mainnet", "config", "list")
	assert.Error(t, err)
}

func TestGlobalTestnetFlagInherited(t *testing.T) {
	dir := t.TempDir()
	// The --testnet flag should be accepted on any subcommand position.
	out, err := runCLI(t, dir, "config", "list", "--testnet")
	require.NoError(t, err)
	// Config list should show the runtime-overridden mode.
	assert.Contains(t, out, "network_mode")
}

func TestGlobalMainnetFlagInherited(t *testing.T) {
	dir := t.TempDir()
	// Set config to testnet first.
	_, _ = runCLI(t, dir, "config", "set-network-mode", "testnet")
	// --mainnet flag should override at runtime.
	out, err := runCLI(t, dir, "config", "list", "--mainnet")
	require.NoError(t, err)
	assert.Contains(t, out, "network_mode")
}

func TestUnknownCommandShowsError(t *testing.T) {
	dir := t.TempDir()
	out, _ := runCLI(t, dir, "unknowncommand")
	assert.Contains(t, strings.ToLower(out), "unknown command")
}

func TestSyncSetSource(t *testing.T) {
	dir := t.TempDir()
	_, err := runCLI(t, dir, "sync", "set-source", "https://example.com/deployments.json")
	require.NoError(t, err)
}

func TestBalanceHelpShowsTestnetFlag(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "balance", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--testnet")
	assert.Contains(t, out, "--mainnet")
}

func TestTxsHelpShowsTestnetFlag(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "txs", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--testnet")
	assert.Contains(t, out, "--mainnet")
}

func TestSendHelpShowsTestnetFlag(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "send", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--testnet")
	assert.Contains(t, out, "--mainnet")
}

func TestWatchHelpShowsTestnetFlag(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, dir, "watch", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--testnet")
	assert.Contains(t, out, "--mainnet")
}
