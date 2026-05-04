package cli

import (
	"bytes"
	"context"
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

func TestRunStartsRuntimeWithResolvedInputs(t *testing.T) {
	var out, errOut bytes.Buffer
	var got RuntimeConfig
	process := &fakeRuntimeProcess{}
	options := testOptions(t, "linux", map[string]string{"DELTAOPS_DCACCOUNT_URL": "dcaccount:env-secret"}, &out, &errOut)
	options.RuntimeFactory = func(_ context.Context, config RuntimeConfig) (RuntimeProcess, error) {
		got = config
		return process, nil
	}

	exit := Run([]string{"run"}, options)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, errOut.String())
	}
	if !process.ran || !process.closed {
		t.Fatalf("process ran/closed = %v/%v, want true/true", process.ran, process.closed)
	}
	if got.Provisioning.DCAccountURL != "dcaccount:env-secret" {
		t.Fatalf("provisioning URL = %q", got.Provisioning.DCAccountURL)
	}
	if got.Paths.StateDir == "" || got.Paths.DeltaChatAccountsDir == "" || got.Stdout == nil || got.Stderr == nil {
		t.Fatalf("runtime config missing resolved fields: %#v", got)
	}
	if strings.Contains(out.String()+errOut.String(), "dcaccount:env-secret") {
		t.Fatalf("output leaked provisioning URL: stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

func TestRunAcceptsChatmailProviderURL(t *testing.T) {
	var out, errOut bytes.Buffer
	var got RuntimeConfig
	process := &fakeRuntimeProcess{}
	options := testOptions(t, "linux", nil, &out, &errOut)
	options.RuntimeFactory = func(_ context.Context, config RuntimeConfig) (RuntimeProcess, error) {
		got = config
		return process, nil
	}

	exit := Run([]string{"run", "--dcaccount-url", "https://nine.testrun.org/"}, options)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, errOut.String())
	}
	if got.Provisioning.DCAccountURL != "DCACCOUNT:https://nine.testrun.org/new" {
		t.Fatalf("provisioning URL = %q", got.Provisioning.DCAccountURL)
	}
	if strings.Contains(out.String()+errOut.String(), "nine.testrun.org") {
		t.Fatalf("output leaked provider setup input: stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

func TestRunAcceptsEnvChatmailProviderURL(t *testing.T) {
	var out, errOut bytes.Buffer
	var got RuntimeConfig
	process := &fakeRuntimeProcess{}
	options := testOptions(t, "linux", map[string]string{"DELTAOPS_DCACCOUNT_URL": "https://nine.testrun.org/"}, &out, &errOut)
	options.RuntimeFactory = func(_ context.Context, config RuntimeConfig) (RuntimeProcess, error) {
		got = config
		return process, nil
	}

	exit := Run([]string{"run"}, options)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, errOut.String())
	}
	if got.Provisioning.DCAccountURL != "DCACCOUNT:https://nine.testrun.org/new" {
		t.Fatalf("provisioning URL = %q", got.Provisioning.DCAccountURL)
	}
}

func TestRunReportsRuntimeFactoryErrorsWithoutLeakingProvisioningURL(t *testing.T) {
	var out, errOut bytes.Buffer
	options := testOptions(t, "linux", nil, &out, &errOut)
	options.RuntimeFactory = func(context.Context, RuntimeConfig) (RuntimeProcess, error) {
		return nil, errRuntimeFactorySecret{}
	}

	exit := Run([]string{"run", "--dcaccount-url", "dcaccount:flag-secret"}, options)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	combined := out.String() + errOut.String()
	if strings.Contains(combined, "dcaccount:flag-secret") || strings.Contains(combined, "token=abc") {
		t.Fatalf("output leaked runtime factory secret in %q", combined)
	}
	if !strings.Contains(errOut.String(), "next action") {
		t.Fatalf("stderr %q does not include next action", errOut.String())
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
		Stdout:         &out,
		Stderr:         &errOut,
		Env:            map[string]string{},
		GOOS:           "linux",
		Version:        "dev",
		RuntimeFactory: missingHelperRuntimeFactory,
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

func TestRunReadsProviderURLFromConfigFile(t *testing.T) {
	var out, errOut bytes.Buffer
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("delta_chat.dcaccount_url: https://nine.testrun.org/\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var got RuntimeConfig
	process := &fakeRuntimeProcess{}
	options := testOptions(t, "linux", nil, &out, &errOut)
	options.RuntimeFactory = func(_ context.Context, config RuntimeConfig) (RuntimeProcess, error) {
		got = config
		return process, nil
	}

	exit := Run([]string{"run", "--config", configPath}, options)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, errOut.String())
	}
	if got.Provisioning.DCAccountURL != "DCACCOUNT:https://nine.testrun.org/new" {
		t.Fatalf("provisioning URL = %q", got.Provisioning.DCAccountURL)
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
	exit := Run([]string{"run", "--dcaccount-url", "dcaccount:secret"}, testOptions(t, "windows", nil, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	message := errOut.String()
	for _, want := range []string{"unsupported operating system", "linux", "darwin", "next action"} {
		if !strings.Contains(message, want) {
			t.Fatalf("stderr %q does not include %q", message, want)
		}
	}
}

func TestRunDarwinRequiresProvisioningBeforeRuntime(t *testing.T) {
	var out, errOut bytes.Buffer
	exit := Run([]string{"run"}, testOptions(t, "darwin", nil, &out, &errOut))

	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	message := errOut.String()
	if !strings.Contains(message, "startup configuration is incomplete") || !strings.Contains(message, "--dcaccount-url") {
		t.Fatalf("stderr %q does not include provisioning next action", message)
	}
	if strings.Contains(message, "unsupported operating system") {
		t.Fatalf("stderr %q rejected darwin before provisioning", message)
	}
}

func TestRunDarwinReachesRuntimeFactory(t *testing.T) {
	var out, errOut bytes.Buffer
	var got RuntimeConfig
	process := &fakeRuntimeProcess{}
	options := testOptions(t, "darwin", nil, &out, &errOut)
	options.GOARCH = "arm64"
	options.RuntimeFactory = func(_ context.Context, config RuntimeConfig) (RuntimeProcess, error) {
		got = config
		return process, nil
	}

	exit := Run([]string{"run", "--dcaccount-url", "dcaccount:secret"}, options)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, errOut.String())
	}
	if got.GOOS != "darwin" || got.GOARCH != "arm64" {
		t.Fatalf("runtime platform = %s/%s, want darwin/arm64", got.GOOS, got.GOARCH)
	}
	if !process.ran || !process.closed {
		t.Fatalf("process ran/closed = %v/%v, want true/true", process.ran, process.closed)
	}
}

func TestDefaultRuntimeFactoryRejectsUnsupportedHelperTarget(t *testing.T) {
	var out, errOut bytes.Buffer
	options := testOptions(t, "linux", nil, &out, &errOut)
	options.GOARCH = "riscv64"
	options.RuntimeFactory = defaultRuntimeFactory

	exit := Run([]string{"run", "--dcaccount-url", "dcaccount:secret"}, options)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	message := errOut.String()
	for _, want := range []string{"helper target is unsupported", "linux/amd64", "next action"} {
		if !strings.Contains(message, want) {
			t.Fatalf("stderr %q does not include %q", message, want)
		}
	}
	if strings.Contains(message, "dcaccount:secret") {
		t.Fatalf("stderr %q leaked provisioning URL", message)
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
	return Options{Stdout: out, Stderr: errOut, Env: merged, GOOS: goos, Version: "dev", RuntimeFactory: missingHelperRuntimeFactory}
}

func missingHelperRuntimeFactory(_ context.Context, config RuntimeConfig) (RuntimeProcess, error) {
	return nil, missingRPCHelperError(config.GOOS, config.GOARCH)
}

type fakeRuntimeProcess struct {
	ran    bool
	closed bool
}

func (p *fakeRuntimeProcess) Run(context.Context) error {
	p.ran = true
	return nil
}

func (p *fakeRuntimeProcess) Close() error {
	p.closed = true
	return nil
}

type errRuntimeFactorySecret struct{}

func (errRuntimeFactorySecret) Error() string {
	return "setup failed dcaccount:flag-secret token=abc"
}
