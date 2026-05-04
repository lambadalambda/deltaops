package dcrpc

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrHelperUnavailable       = errors.New("Delta Chat RPC helper is not packaged")
	ErrUnsupportedHelperTarget = errors.New("Delta Chat RPC helper target is unsupported")
	ErrHelperChecksumMismatch  = errors.New("Delta Chat RPC helper checksum mismatch")
)

//go:embed assets/*
var embeddedHelperFS embed.FS

type HelperAsset struct {
	Filename       string
	Data           []byte
	ExpectedSHA256 string
}

func EmbeddedHelpers() []HelperAsset {
	entries, err := embeddedHelperFS.ReadDir("assets")
	if err != nil {
		return nil
	}
	assets := make([]HelperAsset, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "deltachat-rpc-server-") {
			continue
		}
		data, err := fs.ReadFile(embeddedHelperFS, filepath.ToSlash(filepath.Join("assets", entry.Name())))
		if err != nil || len(data) == 0 {
			continue
		}
		assets = append(assets, HelperAsset{Filename: entry.Name(), Data: data, ExpectedSHA256: helperSHA256(entry.Name())})
	}
	return assets
}

func SelectHelper(assets []HelperAsset, goos, goarch string) (HelperAsset, error) {
	want, ok := HelperFilename(goos, goarch)
	if !ok {
		return HelperAsset{}, fmt.Errorf("%w: %s/%s", ErrUnsupportedHelperTarget, goos, goarch)
	}
	for _, asset := range assets {
		if asset.Filename == want && len(asset.Data) > 0 {
			if err := ValidateHelperAsset(asset); err != nil {
				return HelperAsset{}, err
			}
			return asset, nil
		}
	}
	return HelperAsset{}, fmt.Errorf("%w: missing %s", ErrHelperUnavailable, want)
}

func ValidateHelperAsset(asset HelperAsset) error {
	if asset.ExpectedSHA256 == "" {
		return nil
	}
	sum := sha256.Sum256(asset.Data)
	actual := hex.EncodeToString(sum[:])
	if actual != asset.ExpectedSHA256 {
		return fmt.Errorf("%w: %s", ErrHelperChecksumMismatch, asset.Filename)
	}
	return nil
}

func HelperFilename(goos, goarch string) (string, bool) {
	for _, asset := range helperReleaseAssets {
		if asset.GOOS == goos && asset.GOARCH == goarch {
			return asset.Filename, true
		}
	}
	return "", false
}

func ExtractHelper(asset HelperAsset, dir string) (string, error) {
	if strings.TrimSpace(asset.Filename) == "" {
		return "", errors.New("helper asset filename is required")
	}
	if filepath.Base(asset.Filename) != asset.Filename {
		return "", fmt.Errorf("unsafe helper asset filename %q", asset.Filename)
	}
	if len(asset.Data) == 0 {
		return "", errors.New("helper asset data is empty")
	}
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("helper extraction directory is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create helper directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("restrict helper directory: %w", err)
	}

	path := filepath.Join(dir, asset.Filename)
	tmp, err := os.CreateTemp(dir, "."+asset.Filename+"-tmp-")
	if err != nil {
		return "", fmt.Errorf("create helper file: %w", err)
	}
	tmpPath := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o700); err != nil {
		return "", fmt.Errorf("make helper executable: %w", err)
	}
	if _, err := tmp.Write(asset.Data); err != nil {
		return "", fmt.Errorf("write helper: %w", err)
	}
	if err := tmp.Close(); err != nil {
		closed = true
		return "", fmt.Errorf("close helper: %w", err)
	}
	closed = true
	if err := os.Rename(tmpPath, path); err != nil {
		return "", fmt.Errorf("install helper: %w", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", fmt.Errorf("restrict helper executable: %w", err)
	}
	return path, nil
}
