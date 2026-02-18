package fixtures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixturesDir returns the absolute path to the fixtures directory.
func fixturesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

// LoadABI loads a fixture ABI JSON file and returns its raw bytes.
func LoadABI(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join(fixturesDir(), "abis", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to load fixture ABI: %s", filename)
	return data
}

// LoadRPCResponse loads a fixture RPC response JSON file.
func LoadRPCResponse(t *testing.T, filename string) map[string]interface{} {
	t.Helper()
	path := filepath.Join(fixturesDir(), "rpc", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to load fixture RPC response: %s", filename)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &resp))
	return resp
}
