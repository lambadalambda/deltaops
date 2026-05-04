package config

import (
	"errors"
	"net/url"
	"strings"
)

const (
	DCAccountURLFlag      = "--dcaccount-url"
	DCAccountURLEnv       = "DELTAOPS_DCACCOUNT_URL"
	DCAccountURLConfigKey = "delta_chat.dcaccount_url"
	dcAccountScheme       = "dcaccount:"
	dcAccountURLPrefix    = "DCACCOUNT:"
)

type ProvisioningSources struct {
	FlagDCAccountURL   string
	EnvDCAccountURL    string
	ConfigDCAccountURL string
}

type DeltaChatProvisioning struct {
	DCAccountURL string
}

func ResolveProvisioning(sources ProvisioningSources) DeltaChatProvisioning {
	return DeltaChatProvisioning{
		DCAccountURL: NormalizeAccountSetupInput(firstNonEmpty(
			sources.FlagDCAccountURL,
			sources.EnvDCAccountURL,
			sources.ConfigDCAccountURL,
		)),
	}
}

func (p DeltaChatProvisioning) Validate() error {
	value := NormalizeAccountSetupInput(p.DCAccountURL)
	if value == "" {
		return errors.New("Delta Chat account setup requires a chatmail dcaccount URL or provider URL; provide --dcaccount-url, DELTAOPS_DCACCOUNT_URL, or delta_chat.dcaccount_url")
	}
	if len(value) < len(dcAccountScheme) || !strings.EqualFold(value[:len(dcAccountScheme)], dcAccountScheme) {
		return errors.New("unsupported Delta Chat account setup input; provide a chatmail URL starting with dcaccount: or https://")
	}
	if strings.TrimSpace(value[len(dcAccountScheme):]) == "" {
		return errors.New("Delta Chat account setup requires a complete chatmail dcaccount URL; provide --dcaccount-url, DELTAOPS_DCACCOUNT_URL, or delta_chat.dcaccount_url")
	}
	return nil
}

func NormalizeAccountSetupInput(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) >= len(dcAccountScheme) && strings.EqualFold(value[:len(dcAccountScheme)], dcAccountScheme) {
		return value
	}
	providerURL, err := url.Parse(value)
	if err != nil || providerURL.Host == "" || !strings.EqualFold(providerURL.Scheme, "https") {
		return value
	}
	if providerURL.Path == "" || providerURL.Path == "/" {
		providerURL.Path = "/new"
		providerURL.RawPath = ""
		providerURL.RawQuery = ""
		providerURL.ForceQuery = false
		providerURL.Fragment = ""
	}
	return dcAccountURLPrefix + providerURL.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
