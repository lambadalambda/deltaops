package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PathEnv struct {
	HomeDir       string
	XDGConfigHome string
	XDGStateHome  string
}

type PathOverrides struct {
	ConfigPath string
	StateDir   string
}

type Paths struct {
	ConfigPath           string
	StateDir             string
	DeltaChatAccountsDir string
	BindingPath          string
}

type StartupConfig struct {
	Provisioning DeltaChatProvisioning
}

func ResolvePaths(env PathEnv, overrides PathOverrides) (Paths, error) {
	configPath, err := resolveConfigPath(env, overrides.ConfigPath)
	if err != nil {
		return Paths{}, err
	}
	stateDir, err := resolveStateDir(env, overrides.StateDir)
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		ConfigPath:           configPath,
		StateDir:             stateDir,
		DeltaChatAccountsDir: filepath.Join(stateDir, "deltachat-accounts"),
		BindingPath:          filepath.Join(stateDir, "binding.json"),
	}, nil
}

func (c StartupConfig) Validate() error {
	if err := c.Provisioning.Validate(); err != nil {
		return fmt.Errorf("invalid Delta Chat provisioning config: %w", err)
	}
	return nil
}

func EnsureStateLayout(paths Paths) error {
	if strings.TrimSpace(paths.StateDir) == "" {
		return errors.New("state directory is required")
	}
	if strings.TrimSpace(paths.DeltaChatAccountsDir) == "" {
		return errors.New("Delta Chat accounts directory is required")
	}

	for _, dir := range []string{paths.StateDir, paths.DeltaChatAccountsDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create state directory %q: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("restrict state directory %q: %w", dir, err)
		}
	}
	return nil
}

func WriteSensitiveFile(path string, data []byte) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("sensitive file path is required")
	}

	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-tmp-")
	if err != nil {
		return fmt.Errorf("create sensitive file %q: %w", path, err)
	}
	tmpPath := file.Name()
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		_ = os.Remove(tmpPath)
	}()

	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("restrict sensitive file %q: %w", path, err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write sensitive file %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		closed = true
		return fmt.Errorf("close sensitive file %q: %w", path, err)
	}
	closed = true
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace sensitive file %q: %w", path, err)
	}
	return nil
}

func resolveConfigPath(env PathEnv, override string) (string, error) {
	if path := strings.TrimSpace(override); path != "" {
		return filepath.Clean(path), nil
	}
	base, err := xdgBase("XDG_CONFIG_HOME", env.XDGConfigHome, env.HomeDir, ".config")
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	return filepath.Join(base, "deltaops", "config.yaml"), nil
}

func resolveStateDir(env PathEnv, override string) (string, error) {
	if dir := strings.TrimSpace(override); dir != "" {
		return filepath.Clean(dir), nil
	}
	base, err := xdgBase("XDG_STATE_HOME", env.XDGStateHome, env.HomeDir, filepath.Join(".local", "state"))
	if err != nil {
		return "", fmt.Errorf("resolve state directory: %w", err)
	}
	return filepath.Join(base, "deltaops"), nil
}

func xdgBase(name, xdgHome, homeDir, fallback string) (string, error) {
	if base := strings.TrimSpace(xdgHome); base != "" {
		if !filepath.IsAbs(base) {
			return "", fmt.Errorf("%s must be an absolute path", name)
		}
		return filepath.Clean(base), nil
	}
	if home := strings.TrimSpace(homeDir); home != "" {
		if !filepath.IsAbs(home) {
			return "", errors.New("HOME must be an absolute path")
		}
		return filepath.Join(home, fallback), nil
	}
	return "", errors.New("HOME is required when XDG paths are not set")
}
