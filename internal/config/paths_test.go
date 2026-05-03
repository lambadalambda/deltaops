package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolvePathsUsesDefaultsOutsideWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	work := filepath.Join(root, "repo")
	mustMkdir(t, home)
	mustMkdir(t, work)

	paths, err := ResolvePaths(PathEnv{HomeDir: home}, PathOverrides{})
	if err != nil {
		t.Fatalf("ResolvePaths returned error: %v", err)
	}

	wantConfig := filepath.Join(home, ".config", "deltaops", "config.yaml")
	if paths.ConfigPath != wantConfig {
		t.Fatalf("ConfigPath = %q, want %q", paths.ConfigPath, wantConfig)
	}
	wantState := filepath.Join(home, ".local", "state", "deltaops")
	if paths.StateDir != wantState {
		t.Fatalf("StateDir = %q, want %q", paths.StateDir, wantState)
	}
	if strings.HasPrefix(paths.StateDir, work+string(os.PathSeparator)) || paths.StateDir == work {
		t.Fatalf("StateDir %q is inside working directory %q", paths.StateDir, work)
	}
	if paths.DeltaChatAccountsDir != filepath.Join(wantState, "deltachat-accounts") {
		t.Fatalf("DeltaChatAccountsDir = %q", paths.DeltaChatAccountsDir)
	}
	if paths.BindingPath != filepath.Join(wantState, "binding.json") {
		t.Fatalf("BindingPath = %q", paths.BindingPath)
	}
}

func TestResolvePathsUsesXDGAndOverrides(t *testing.T) {
	root := t.TempDir()
	xdgConfig := filepath.Join(root, "xdg-config")
	xdgState := filepath.Join(root, "xdg-state")
	overrideConfig := filepath.Join(root, "custom", "deltaops.yaml")
	overrideState := filepath.Join(root, "custom-state")

	t.Run("xdg defaults", func(t *testing.T) {
		paths, err := ResolvePaths(PathEnv{XDGConfigHome: xdgConfig, XDGStateHome: xdgState}, PathOverrides{})
		if err != nil {
			t.Fatalf("ResolvePaths returned error: %v", err)
		}
		if paths.ConfigPath != filepath.Join(xdgConfig, "deltaops", "config.yaml") {
			t.Fatalf("ConfigPath = %q", paths.ConfigPath)
		}
		if paths.StateDir != filepath.Join(xdgState, "deltaops") {
			t.Fatalf("StateDir = %q", paths.StateDir)
		}
	})

	t.Run("explicit overrides", func(t *testing.T) {
		paths, err := ResolvePaths(PathEnv{XDGConfigHome: xdgConfig, XDGStateHome: xdgState}, PathOverrides{
			ConfigPath: overrideConfig,
			StateDir:   overrideState,
		})
		if err != nil {
			t.Fatalf("ResolvePaths returned error: %v", err)
		}
		if paths.ConfigPath != overrideConfig {
			t.Fatalf("ConfigPath = %q, want %q", paths.ConfigPath, overrideConfig)
		}
		if paths.StateDir != overrideState {
			t.Fatalf("StateDir = %q, want %q", paths.StateDir, overrideState)
		}
	})
}

func TestResolvePathsRequiresHomeForDefaults(t *testing.T) {
	_, err := ResolvePaths(PathEnv{}, PathOverrides{})
	if err == nil {
		t.Fatal("ResolvePaths returned nil error, want missing home error")
	}
	if !strings.Contains(err.Error(), "HOME") {
		t.Fatalf("error %q does not mention HOME", err)
	}
}

func TestResolvePathsRejectsRelativeXDGPaths(t *testing.T) {
	absolute := t.TempDir()
	tests := []struct {
		name string
		env  PathEnv
		want string
	}{
		{
			name: "config",
			env:  PathEnv{XDGConfigHome: "relative-config", XDGStateHome: absolute},
			want: "XDG_CONFIG_HOME",
		},
		{
			name: "state",
			env:  PathEnv{XDGConfigHome: absolute, XDGStateHome: "relative-state"},
			want: "XDG_STATE_HOME",
		},
		{
			name: "home fallback",
			env:  PathEnv{HomeDir: "relative-home"},
			want: "HOME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolvePaths(tt.env, PathOverrides{})
			if err == nil {
				t.Fatal("ResolvePaths returned nil error, want relative path error")
			}
			if !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "absolute") {
				t.Fatalf("error %q does not mention %s absolute path", err, tt.want)
			}
		})
	}
}

func TestStartupConfigValidateReportsProvisioningErrors(t *testing.T) {
	err := (StartupConfig{Provisioning: DeltaChatProvisioning{DCAccountURL: "https://provider.example/signup"}}).Validate()
	if err == nil {
		t.Fatal("Validate returned nil, want invalid provisioning error")
	}
	if !strings.Contains(err.Error(), "Delta Chat provisioning") {
		t.Fatalf("error %q does not include config area", err)
	}
}

func TestEnsureStateLayoutCreatesRestrictedState(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	paths := Paths{
		StateDir:             stateDir,
		DeltaChatAccountsDir: filepath.Join(stateDir, "deltachat-accounts"),
		BindingPath:          filepath.Join(stateDir, "binding.json"),
	}

	if err := EnsureStateLayout(paths); err != nil {
		t.Fatalf("EnsureStateLayout returned error: %v", err)
	}
	assertPerm(t, paths.StateDir, 0o700)
	assertPerm(t, paths.DeltaChatAccountsDir, 0o700)

	if err := WriteSensitiveFile(paths.BindingPath, []byte(`{"contact_id":"operator"}`)); err != nil {
		t.Fatalf("WriteSensitiveFile returned error: %v", err)
	}
	assertPerm(t, paths.BindingPath, 0o600)
	data, err := os.ReadFile(paths.BindingPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) == "" {
		t.Fatal("sensitive file was empty")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", path, err)
	}
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("exact POSIX permission assertions are not supported on Windows")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) returned error: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("permissions for %q = %v, want %v", path, got, want)
	}
}
