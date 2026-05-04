package dcrpc

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSelectHelperUsesReleaseAssetNames(t *testing.T) {
	assets := []HelperAsset{
		{Filename: "deltachat-rpc-server-aarch64-linux", Data: []byte("arm64")},
		{Filename: "deltachat-rpc-server-x86_64-linux", Data: []byte("amd64")},
		{Filename: "deltachat-rpc-server-aarch64-macos", Data: []byte("darwin-arm64")},
	}

	asset, err := SelectHelper(assets, "linux", "amd64")
	if err != nil {
		t.Fatalf("SelectHelper returned error: %v", err)
	}
	if asset.Filename != "deltachat-rpc-server-x86_64-linux" {
		t.Fatalf("Filename = %q", asset.Filename)
	}

	asset, err = SelectHelper(assets, "linux", "arm64")
	if err != nil {
		t.Fatalf("SelectHelper arm64 returned error: %v", err)
	}
	if asset.Filename != "deltachat-rpc-server-aarch64-linux" {
		t.Fatalf("Filename = %q", asset.Filename)
	}

	asset, err = SelectHelper(assets, "darwin", "arm64")
	if err != nil {
		t.Fatalf("SelectHelper darwin arm64 returned error: %v", err)
	}
	if asset.Filename != "deltachat-rpc-server-aarch64-macos" {
		t.Fatalf("Filename = %q", asset.Filename)
	}
}

func TestHelperReleaseAssetsExposePinnedDownloadMetadata(t *testing.T) {
	assets := HelperReleaseAssets()
	want := map[string]string{
		"linux/amd64":  "deltachat-rpc-server-x86_64-linux",
		"linux/arm64":  "deltachat-rpc-server-aarch64-linux",
		"linux/386":    "deltachat-rpc-server-i686-linux",
		"darwin/arm64": "deltachat-rpc-server-aarch64-macos",
		"darwin/amd64": "deltachat-rpc-server-x86_64-macos",
	}
	seen := make(map[string]bool, len(want))
	for _, asset := range assets {
		key := asset.GOOS + "/" + asset.GOARCH
		if filename, ok := want[key]; ok {
			seen[key] = true
			if asset.Filename != filename {
				t.Fatalf("%s filename = %q, want %q", key, asset.Filename, filename)
			}
			if asset.Version != HelperReleaseVersion {
				t.Fatalf("%s version = %q, want %q", key, asset.Version, HelperReleaseVersion)
			}
			if asset.SHA256 == "" {
				t.Fatalf("%s has empty SHA256", key)
			}
			wantURL := "https://github.com/chatmail/core/releases/download/" + HelperReleaseVersion + "/" + asset.Filename
			if asset.URL != wantURL {
				t.Fatalf("%s URL = %q, want %q", key, asset.URL, wantURL)
			}
		}
	}
	for key := range want {
		if !seen[key] {
			t.Fatalf("missing helper release asset for %s", key)
		}
	}
}

func TestSelectHelperReportsMissingPackagedAsset(t *testing.T) {
	_, err := SelectHelper(nil, "linux", "amd64")
	if err == nil {
		t.Fatal("SelectHelper returned nil error, want missing helper")
	}
	if !errors.Is(err, ErrHelperUnavailable) {
		t.Fatalf("error = %v, want ErrHelperUnavailable", err)
	}
}

func TestSelectHelperRejectsChecksumMismatch(t *testing.T) {
	_, err := SelectHelper([]HelperAsset{{
		Filename:       "deltachat-rpc-server-x86_64-linux",
		Data:           []byte("stale helper"),
		ExpectedSHA256: "28e10b40518f55fa8ce20edd119fa743dd29a22df372b58443ec53eb753cb50c",
	}}, "linux", "amd64")
	if err == nil {
		t.Fatal("SelectHelper returned nil error, want checksum mismatch")
	}
	if !errors.Is(err, ErrHelperChecksumMismatch) {
		t.Fatalf("error = %v, want ErrHelperChecksumMismatch", err)
	}
}

func TestSelectHelperReportsUnsupportedTarget(t *testing.T) {
	_, err := SelectHelper(nil, "linux", "riscv64")
	if err == nil {
		t.Fatal("SelectHelper returned nil error, want unsupported target")
	}
	if !errors.Is(err, ErrUnsupportedHelperTarget) {
		t.Fatalf("error = %v, want ErrUnsupportedHelperTarget", err)
	}
}

func TestExtractHelperWritesPrivateExecutable(t *testing.T) {
	path, err := ExtractHelper(HelperAsset{Filename: "deltachat-rpc-server-x86_64-linux", Data: []byte("helper")}, filepath.Join(t.TempDir(), "helpers"))
	if err != nil {
		t.Fatalf("ExtractHelper returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "helper" {
		t.Fatalf("helper data = %q", data)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat returned error: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("helper mode = %v, want 0700", got)
		}
	}
}

func TestExtractHelperRejectsUnsafeAssetName(t *testing.T) {
	_, err := ExtractHelper(HelperAsset{Filename: "../deltachat-rpc-server", Data: []byte("helper")}, t.TempDir())
	if err == nil {
		t.Fatal("ExtractHelper returned nil error, want unsafe filename error")
	}
}
