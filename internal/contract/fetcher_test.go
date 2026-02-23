package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseABIValid(t *testing.T) {
	data := `[
		{"name":"balanceOf","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}],"stateMutability":"view"},
		{"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable"}
	]`

	abi, err := parseABI([]byte(data))
	require.NoError(t, err)
	assert.Len(t, abi, 2)
	assert.Equal(t, "balanceOf", abi[0].Name)
	assert.Equal(t, "function", abi[0].Type)
	assert.Len(t, abi[0].Inputs, 1)
	assert.Len(t, abi[0].Outputs, 1)
	assert.Equal(t, "view", abi[0].StateMutability)
	assert.Equal(t, "transfer", abi[1].Name)
	assert.Len(t, abi[1].Inputs, 2)
}

func TestParseABIInvalidJSON(t *testing.T) {
	_, err := parseABI([]byte("{not valid json"))
	assert.Error(t, err)
}

func TestParseABIEmptyArray(t *testing.T) {
	abi, err := parseABI([]byte("[]"))
	require.NoError(t, err)
	assert.Empty(t, abi)
}

func TestParseABIWithEvents(t *testing.T) {
	data := `[
		{"name":"Transfer","type":"event","inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}]},
		{"name":"balanceOf","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}],"stateMutability":"view"}
	]`

	abi, err := parseABI([]byte(data))
	require.NoError(t, err)
	assert.Len(t, abi, 2)
	assert.Equal(t, "event", abi[0].Type)
	assert.Equal(t, "function", abi[1].Type)
}

func TestParseABIMinimalEntry(t *testing.T) {
	data := `[{"name":"foo","type":"function"}]`

	abi, err := parseABI([]byte(data))
	require.NoError(t, err)
	assert.Len(t, abi, 1)
	assert.Equal(t, "foo", abi[0].Name)
	assert.Empty(t, abi[0].Inputs)
	assert.Empty(t, abi[0].Outputs)
}

func TestParseABINullInputs(t *testing.T) {
	data := `[{"name":"foo","type":"function","inputs":null,"outputs":null,"stateMutability":"view"}]`

	abi, err := parseABI([]byte(data))
	require.NoError(t, err)
	assert.Len(t, abi, 1)
	assert.Nil(t, abi[0].Inputs)
	assert.Nil(t, abi[0].Outputs)
}

func TestParseABINotArray(t *testing.T) {
	_, err := parseABI([]byte(`{"name":"foo"}`))
	assert.Error(t, err)
}

func TestLoadFromFileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abi.json")

	abiJSON := `[
		{"name":"name","type":"function","inputs":[],"outputs":[{"name":"","type":"string"}],"stateMutability":"view"},
		{"name":"symbol","type":"function","inputs":[],"outputs":[{"name":"","type":"string"}],"stateMutability":"view"}
	]`
	require.NoError(t, os.WriteFile(path, []byte(abiJSON), 0o644))

	abi, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Len(t, abi, 2)
	assert.Equal(t, "name", abi[0].Name)
	assert.Equal(t, "symbol", abi[1].Name)
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/abi.json")
	assert.Error(t, err)
}

func TestLoadFromFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	_, err := LoadFromFile(path)
	assert.Error(t, err)
}

func TestLoadFromFileEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o644))

	abi, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Empty(t, abi)
}

func TestLoadFromFileERC20Fixture(t *testing.T) {
	// Test with a realistic ERC20 ABI.
	dir := t.TempDir()
	path := filepath.Join(dir, "erc20.json")

	abi := []ABIEntry{
		{Name: "name", Type: "function", Inputs: []ABIParam{}, Outputs: []ABIParam{{Type: "string"}}, StateMutability: "view"},
		{Name: "symbol", Type: "function", Inputs: []ABIParam{}, Outputs: []ABIParam{{Type: "string"}}, StateMutability: "view"},
		{Name: "decimals", Type: "function", Inputs: []ABIParam{}, Outputs: []ABIParam{{Type: "uint8"}}, StateMutability: "view"},
		{Name: "balanceOf", Type: "function", Inputs: []ABIParam{{Name: "account", Type: "address"}}, Outputs: []ABIParam{{Type: "uint256"}}, StateMutability: "view"},
		{Name: "transfer", Type: "function", Inputs: []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}}, Outputs: []ABIParam{{Type: "bool"}}, StateMutability: "nonpayable"},
	}

	data, err := json.Marshal(abi)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	loaded, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Len(t, loaded, 5)

	// Verify read/write classification.
	assert.True(t, loaded[0].IsReadFunction())  // name
	assert.True(t, loaded[3].IsReadFunction())  // balanceOf
	assert.True(t, loaded[4].IsWriteFunction()) // transfer
}

func TestNewFetcher(t *testing.T) {
	f := NewFetcher("test-api-key")
	assert.NotNil(t, f)
	assert.NotNil(t, f.client)
	assert.Equal(t, "test-api-key", f.apiKey)
}

func TestNewFetcherEmptyAPIKey(t *testing.T) {
	f := NewFetcher("")
	assert.NotNil(t, f)
	assert.Equal(t, "", f.apiKey)
}

func TestFetchFromExplorerSuccess(t *testing.T) {
	abiJSON := `[{"name":"balanceOf","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}],"stateMutability":"view"}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api", r.URL.Path)
		assert.Equal(t, "contract", r.URL.Query().Get("module"))
		assert.Equal(t, "getabi", r.URL.Query().Get("action"))
		assert.Equal(t, "0xContractAddr", r.URL.Query().Get("address"))
		assert.Equal(t, "test-key", r.URL.Query().Get("apikey"))

		resp := map[string]string{
			"status":  "1",
			"message": "OK",
			"result":  abiJSON,
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("test-key")
	f.client = server.Client()

	abi, err := f.FetchFromExplorer(server.URL, "0xContractAddr")
	require.NoError(t, err)
	assert.Len(t, abi, 1)
	assert.Equal(t, "balanceOf", abi[0].Name)
}

func TestFetchFromExplorerErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"status":  "0",
			"message": "NOTOK",
			"result":  "Max rate limit reached",
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("test-key")
	f.client = server.Client()

	_, err := f.FetchFromExplorer(server.URL, "0xContractAddr")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOTOK")
}

func TestFetchFromExplorerInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json")) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("test-key")
	f.client = server.Client()

	_, err := f.FetchFromExplorer(server.URL, "0xContractAddr")
	assert.Error(t, err)
}

func TestFetchFromExplorerServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	f := NewFetcher("test-key")
	f.client = server.Client()

	_, err := f.FetchFromExplorer(server.URL, "0xContractAddr")
	assert.Error(t, err)
}

func TestFetchFromExplorerInvalidABIInResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"status":  "1",
			"message": "OK",
			"result":  "not valid abi json",
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("test-key")
	f.client = server.Client()

	_, err := f.FetchFromExplorer(server.URL, "0xContractAddr")
	assert.Error(t, err)
}

func TestFetchFromURLSuccess(t *testing.T) {
	abiJSON := `[{"name":"name","type":"function","inputs":[],"outputs":[{"name":"","type":"string"}],"stateMutability":"view"}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(abiJSON)) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("")
	f.client = server.Client()

	abi, err := f.FetchFromURL(server.URL)
	require.NoError(t, err)
	assert.Len(t, abi, 1)
	assert.Equal(t, "name", abi[0].Name)
}

func TestFetchFromURLInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json")) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("")
	f.client = server.Client()

	_, err := f.FetchFromURL(server.URL)
	assert.Error(t, err)
}

func TestFetchFromURLEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]")) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("")
	f.client = server.Client()

	abi, err := f.FetchFromURL(server.URL)
	require.NoError(t, err)
	assert.Empty(t, abi)
}

func TestFetchFromURLMultipleEntries(t *testing.T) {
	abiJSON := `[
		{"name":"name","type":"function","stateMutability":"view"},
		{"name":"symbol","type":"function","stateMutability":"view"},
		{"name":"decimals","type":"function","stateMutability":"view"},
		{"name":"transfer","type":"function","stateMutability":"nonpayable"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(abiJSON)) //nolint:errcheck
	}))
	defer server.Close()

	f := NewFetcher("")
	f.client = server.Client()

	abi, err := f.FetchFromURL(server.URL)
	require.NoError(t, err)
	assert.Len(t, abi, 4)
}

func TestFetchFromExplorerConnectionRefused(t *testing.T) {
	f := NewFetcher("test-key")

	_, err := f.FetchFromExplorer("http://127.0.0.1:1", "0xAddr")
	assert.Error(t, err)
}

func TestFetchFromURLConnectionRefused(t *testing.T) {
	f := NewFetcher("")

	_, err := f.FetchFromURL("http://127.0.0.1:1")
	assert.Error(t, err)
}
