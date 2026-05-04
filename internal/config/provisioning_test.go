package config

import (
	"strings"
	"testing"
)

func TestResolveProvisioningPrefersFlag(t *testing.T) {
	provisioning := ResolveProvisioning(ProvisioningSources{
		FlagDCAccountURL:   " dcaccount:flag.example ",
		EnvDCAccountURL:    "dcaccount:env.example",
		ConfigDCAccountURL: "dcaccount:config.example",
	})

	if provisioning.DCAccountURL != "dcaccount:flag.example" {
		t.Fatalf("DCAccountURL = %q, want %q", provisioning.DCAccountURL, "dcaccount:flag.example")
	}
}

func TestResolveProvisioningNormalizesChatmailProviderURL(t *testing.T) {
	provisioning := ResolveProvisioning(ProvisioningSources{FlagDCAccountURL: " https://nine.testrun.org/ "})

	if provisioning.DCAccountURL != "DCACCOUNT:https://nine.testrun.org/new" {
		t.Fatalf("DCAccountURL = %q", provisioning.DCAccountURL)
	}
}

func TestNormalizeAccountSetupInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "dcaccount", input: " dcaccount:secret ", want: "dcaccount:secret"},
		{name: "provider home", input: "https://nine.testrun.org/", want: "DCACCOUNT:https://nine.testrun.org/new"},
		{name: "provider home with tracking", input: "https://nine.testrun.org/?utm=x#top", want: "DCACCOUNT:https://nine.testrun.org/new"},
		{name: "provider host", input: "https://nine.testrun.org", want: "DCACCOUNT:https://nine.testrun.org/new"},
		{name: "provider endpoint", input: "https://nine.testrun.org/new", want: "DCACCOUNT:https://nine.testrun.org/new"},
		{name: "unsupported", input: "mailto:bot@example.test", want: "mailto:bot@example.test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeAccountSetupInput(tt.input); got != tt.want {
				t.Fatalf("NormalizeAccountSetupInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveProvisioningFallsBackToEnvThenConfig(t *testing.T) {
	tests := []struct {
		name    string
		sources ProvisioningSources
		want    string
	}{
		{
			name: "env",
			sources: ProvisioningSources{
				EnvDCAccountURL:    " dcaccount:env.example ",
				ConfigDCAccountURL: "dcaccount:config.example",
			},
			want: "dcaccount:env.example",
		},
		{
			name: "config",
			sources: ProvisioningSources{
				ConfigDCAccountURL: " dcaccount:config.example ",
			},
			want: "dcaccount:config.example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provisioning := ResolveProvisioning(tt.sources)
			if provisioning.DCAccountURL != tt.want {
				t.Fatalf("DCAccountURL = %q, want %q", provisioning.DCAccountURL, tt.want)
			}
		})
	}
}

func TestProvisioningValidateRequiresDCAccountURL(t *testing.T) {
	err := (DeltaChatProvisioning{}).Validate()
	if err == nil {
		t.Fatal("Validate returned nil, want missing input error")
	}

	message := err.Error()
	for _, want := range []string{"--dcaccount-url", "DELTAOPS_DCACCOUNT_URL", "delta_chat.dcaccount_url"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not include next action %q", message, want)
		}
	}
}

func TestProvisioningValidateAcceptsDCAccountURL(t *testing.T) {
	provisioning := DeltaChatProvisioning{DCAccountURL: "dcaccount:nine.testrun.org"}
	if err := provisioning.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestProvisioningValidateAcceptsProviderURL(t *testing.T) {
	provisioning := DeltaChatProvisioning{DCAccountURL: "https://nine.testrun.org/"}
	if err := provisioning.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestProvisioningValidateRejectsEmptyDCAccountURLPayload(t *testing.T) {
	err := (DeltaChatProvisioning{DCAccountURL: "dcaccount:   "}).Validate()
	if err == nil {
		t.Fatal("Validate returned nil, want incomplete dcaccount URL error")
	}

	message := err.Error()
	if !strings.Contains(message, "complete") {
		t.Fatalf("error %q does not explain the URL is incomplete", message)
	}
	if !strings.Contains(message, DCAccountURLFlag) {
		t.Fatalf("error %q does not include next action %q", message, DCAccountURLFlag)
	}
}

func TestProvisioningValidateRejectsUnsupportedURLWithoutLeakingValue(t *testing.T) {
	const unsupported = "smtp://provider.example/signup?token=secret"
	err := (DeltaChatProvisioning{DCAccountURL: unsupported}).Validate()
	if err == nil {
		t.Fatal("Validate returned nil, want unsupported input error")
	}

	message := err.Error()
	if !strings.Contains(message, "dcaccount:") || !strings.Contains(message, "https://") {
		t.Fatalf("error %q does not explain supported URL scheme", message)
	}
	if strings.Contains(message, unsupported) || strings.Contains(message, "secret") {
		t.Fatalf("error %q leaks provisioning input", message)
	}
}
