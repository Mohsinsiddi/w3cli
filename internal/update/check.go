package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	npmPackage   = "@siddi_404/w3cli"
	registryURL  = "https://registry.npmjs.org/@siddi_404%2fw3cli/latest"
	cacheTTL     = 24 * time.Hour
	checkTimeout = 2 * time.Second
)

type cacheEntry struct {
	Latest    string `json:"latest"`
	CheckedAt int64  `json:"checked_at"`
}

type npmResponse struct {
	Version string `json:"version"`
}

func cacheFile() string {
	cacheDir, _ := os.UserCacheDir()
	return filepath.Join(cacheDir, "w3cli", "update-check.json")
}

func readCache() *cacheEntry {
	data, err := os.ReadFile(cacheFile())
	if err != nil {
		return nil
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}
	if time.Since(time.Unix(entry.CheckedAt, 0)) > cacheTTL {
		return nil
	}
	return &entry
}

func writeCache(latest string) {
	entry := cacheEntry{Latest: latest, CheckedAt: time.Now().Unix()}
	data, _ := json.Marshal(entry)
	dir := filepath.Dir(cacheFile())
	os.MkdirAll(dir, 0o755)
	os.WriteFile(cacheFile(), data, 0o644)
}

// CheckForUpdate checks if a newer version is available on npm.
// Returns the latest version string if an update is available, or empty string if up to date.
// Designed to be called in a goroutine — never blocks the CLI for more than checkTimeout.
func CheckForUpdate(currentVersion string) string {
	// Check cache first.
	if cached := readCache(); cached != nil {
		if isNewer(cached.Latest, currentVersion) {
			return cached.Latest
		}
		return ""
	}

	// Fetch latest from npm registry.
	client := &http.Client{Timeout: checkTimeout}
	resp, err := client.Get(registryURL)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	var npmResp npmResponse
	if err := json.NewDecoder(resp.Body).Decode(&npmResp); err != nil {
		return ""
	}

	writeCache(npmResp.Version)

	if isNewer(npmResp.Version, currentVersion) {
		return npmResp.Version
	}
	return ""
}

// NotifyMessage returns a formatted update notification string.
func NotifyMessage(latest string) string {
	return fmt.Sprintf(
		"\n  Update available: v%s\n  Run: npm install -g %s\n",
		latest, npmPackage,
	)
}

// isNewer returns true if latest > current using simple semver comparison.
func isNewer(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")
	lp := strings.Split(latest, ".")
	cp := strings.Split(current, ".")
	for i := 0; i < len(lp) && i < len(cp); i++ {
		if lp[i] != cp[i] {
			return lp[i] > cp[i]
		}
	}
	return len(lp) > len(cp)
}
