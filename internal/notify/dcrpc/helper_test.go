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
