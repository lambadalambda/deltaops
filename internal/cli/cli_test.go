package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run([]string{"version"}, Options{Stdout: &out, Stderr: &errOut, Version: "1.2.3", Commit: "abc123"})

	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "deltaops 1.2.3") || !strings.Contains(got, "abc123") {
		t.Fatalf("version output = %q", got)
	}
}

func TestRunRequiresProvisioningInput(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run([]string{"run"}, testOptions(t, "linux", nil, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	message := errOut.String()
	for _, want := range []string{"next action", "--dcaccount-url", "DELTAOPS_DCACCOUNT_URL"} {
		if !strings.Contains(message, want) {
			t.Fatalf("stderr %q does not include %q", message, want)
		}
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
}

func TestNoCommandDefaultsToRun(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run(nil, testOptions(t, "linux", nil, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(errOut.String(), "next action") || !strings.Contains(errOut.String(), "--dcaccount-url") {
		t.Fatalf("stderr %q does not include run next action", errOut.String())
	}
}

func TestRunUsesFlagBeforeEnvAndDoesNotLeakProvisioningURLs(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run([]string{"run", "--dcaccount-url", "dcaccount:flag-secret"}, testOptions(t, "linux", map[string]string{
		"DELTAOPS_DCACCOUNT_URL": "dcaccount:env-secret",
	}, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	combined := out.String() + errOut.String()
	for _, secret := range []string{"dcaccount:flag-secret", "dcaccount:env-secret"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("output leaked provisioning URL %q in %q", secret, combined)
		}
	}
	if !strings.Contains(errOut.String(), "Delta Chat RPC helper") || !strings.Contains(errOut.String(), "next action") {
		t.Fatalf("stderr %q does not explain runtime packaging next action", errOut.String())
	}
}

func TestRunDoesNotLeakPositionalProvisioningURL(t *testing.T) {
	tests := [][]string{
		{"dcaccount:command-secret"},
		{"run", "dcaccount:run-secret"},
		{"version", "dcaccount:version-secret"},
	}
	for _, args := range tests {
		var out, errOut bytes.Buffer
		exit := Run(args, testOptions(t, "linux", nil, &out, &errOut))
		if exit != 2 {
			t.Fatalf("Run(%v) exit = %d, want 2", args, exit)
		}
		combined := out.String() + errOut.String()
		if strings.Contains(combined, "dcaccount:") || strings.Contains(combined, "secret") {
			t.Fatalf("Run(%v) leaked positional secret in %q", args, combined)
		}
		if !strings.Contains(errOut.String(), "next action") {
			t.Fatalf("Run(%v) stderr %q does not include next action", args, errOut.String())
		}
	}
}

func TestRunSkipsConfigFileWhenFlagProvisioningIsPresent(t *testing.T) {
	var out, errOut bytes.Buffer
	missingConfig := filepath.Join(t.TempDir(), "missing.yaml")
	exit := Run([]string{"run", "--config", missingConfig, "--dcaccount-url", "dcaccount:flag-secret"}, testOptions(t, "linux", nil, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if strings.Contains(errOut.String(), "cannot read config file") {
		t.Fatalf("stderr %q read lower-precedence config despite flag provisioning", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Delta Chat RPC helper") {
		t.Fatalf("stderr %q does not reach runtime packaging error", errOut.String())
	}
}

func TestRunWithFlagAndStateDirDoesNotRequireHomeForConfigPath(t *testing.T) {
	var out, errOut bytes.Buffer
	stateDir := filepath.Join(t.TempDir(), "state")
	exit := Run([]string{"run", "--state-dir", stateDir, "--dcaccount-url", "dcaccount:flag-secret"}, Options{
		Stdout:  &out,
		Stderr:  &errOut,
		Env:     map[string]string{},
		GOOS:    "linux",
		Version: "dev",
	})

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if strings.Contains(errOut.String(), "HOME") {
		t.Fatalf("stderr %q required HOME despite explicit state dir and flag provisioning", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Delta Chat RPC helper") {
		t.Fatalf("stderr %q does not reach runtime packaging error", errOut.String())
	}
}

func TestRunReadsDCAccountURLFromConfigFile(t *testing.T) {
	var out, errOut bytes.Buffer
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("delta_chat.dcaccount_url: dcaccount:config-secret\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	exit := Run([]string{"run", "--config", configPath}, testOptions(t, "linux", nil, &out, &errOut))
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	combined := out.String() + errOut.String()
	if strings.Contains(combined, "dcaccount:config-secret") {
		t.Fatalf("output leaked config provisioning URL in %q", combined)
	}
	if !strings.Contains(errOut.String(), "Delta Chat RPC helper") {
		t.Fatalf("stderr %q does not reach runtime packaging error", errOut.String())
	}
}

func TestRunReadsNestedConfigFileWithComments(t *testing.T) {
	var out, errOut bytes.Buffer
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("delta_chat: # account settings\n  dcaccount_url: \"dcaccount:config-secret\" # provisioning URL\n")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	exit := Run([]string{"run", "--config", configPath}, testOptions(t, "linux", nil, &out, &errOut))
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	combined := out.String() + errOut.String()
	if strings.Contains(combined, "dcaccount:config-secret") {
		t.Fatalf("output leaked config provisioning URL in %q", combined)
	}
	if !strings.Contains(errOut.String(), "Delta Chat RPC helper") {
		t.Fatalf("stderr %q does not reach runtime packaging error", errOut.String())
	}
}

func TestRunRejectsUnsupportedPlatformWithNextAction(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run([]string{"run", "--dcaccount-url", "dcaccount:secret"}, testOptions(t, "darwin", nil, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	message := errOut.String()
	for _, want := range []string{"unsupported operating system", "linux", "next action"} {
		if !strings.Contains(message, want) {
			t.Fatalf("stderr %q does not include %q", message, want)
		}
	}
}

func TestRunRejectsUnsupportedPlatformBeforeProvisioning(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run([]string{"run"}, testOptions(t, "darwin", nil, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	message := errOut.String()
	if !strings.Contains(message, "unsupported operating system") || !strings.Contains(message, "next action") {
		t.Fatalf("stderr %q does not include unsupported-platform next action", message)
	}
	if strings.Contains(message, "--dcaccount-url") {
		t.Fatalf("stderr %q requested provisioning before rejecting unsupported platform", message)
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run([]string{"wat"}, Options{Stdout: &out, Stderr: &errOut})

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(errOut.String(), "unknown command") || !strings.Contains(errOut.String(), "next action") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func testOptions(t *testing.T, goos string, env map[string]string, out, errOut *bytes.Buffer) Options {
	t.Helper()
	root := t.TempDir()
	merged := map[string]string{
		"HOME": root,
	}
	for key, value := range env {
		merged[key] = value
	}
	return Options{Stdout: out, Stderr: errOut, Env: merged, GOOS: goos, Version: "dev"}
}
