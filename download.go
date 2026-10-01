package reparos

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// releaseBaseURL hosts the prebuilt bridge libraries (see build-native.yml).
const releaseBaseURL = "https://github.com/Hieu31/reparos-go/releases/latest/download/"

func nativePlatform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

// downloadNativeLib fetches the prebuilt bridge for this platform into the user
// cache, verifying its SHA-256, and returns the cached path. Set
// REPAROS_NO_DOWNLOAD=1 to disable.
func downloadNativeLib() (string, error) {
	if os.Getenv("REPAROS_NO_DOWNLOAD") != "" {
		return "", errNativeNotFound
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "reparos-go", "native")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	platform := nativePlatform()
	ext := filepath.Ext(nativeLibName())
	asset := "reparos_bridge-" + platform + ext
	client := &http.Client{Timeout: 5 * time.Minute}

	sumLine, err := httpGetString(client, releaseBaseURL+"SHA256SUMS-"+platform+".txt")
	if err != nil {
		return "", fmt.Errorf("download checksum: %w", err)
	}
	fields := strings.Fields(sumLine)
	if len(fields) < 1 {
		return "", fmt.Errorf("download checksum: empty checksum file")
	}
	want := strings.ToLower(fields[0])

	resp, err := client.Get(releaseBaseURL + asset)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", asset, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: %s", asset, resp.Status)
	}

	tmp, err := os.CreateTemp(dir, "download-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		tmp.Close()
		return "", fmt.Errorf("download %s: %w", asset, err)
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return "", fmt.Errorf("download %s: checksum mismatch (got %s, want %s)", asset, got, want)
	}

	dest := filepath.Join(dir, nativeLibName())
	if err := os.Rename(tmp.Name(), dest); err != nil {
		// Another process may have installed it already (and, on Windows, holds it open).
		if _, statErr := os.Stat(dest); statErr != nil {
			return "", err
		}
	}
	return dest, nil
}

func httpGetString(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", url, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return string(b), err
}
