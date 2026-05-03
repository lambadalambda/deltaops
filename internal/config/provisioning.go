package config

import (
	"errors"
	"strings"
)

const (
	DCAccountURLFlag      = "--dcaccount-url"
	DCAccountURLEnv       = "DELTAOPS_DCACCOUNT_URL"
	DCAccountURLConfigKey = "delta_chat.dcaccount_url"
	dcAccountScheme       = "dcaccount:"
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
		DCAccountURL: firstNonEmpty(
			sources.FlagDCAccountURL,
			sources.EnvDCAccountURL,
			sources.ConfigDCAccountURL,
		),
	}
}

func (p DeltaChatProvisioning) Validate() error {
	value := strings.TrimSpace(p.DCAccountURL)
	if value == "" {
		return errors.New("Delta Chat account setup requires a chatmail dcaccount URL; provide --dcaccount-url, DELTAOPS_DCACCOUNT_URL, or delta_chat.dcaccount_url")
	}
	if len(value) < len(dcAccountScheme) || !strings.EqualFold(value[:len(dcAccountScheme)], dcAccountScheme) {
		return errors.New("unsupported Delta Chat account setup input; provide a chatmail URL starting with dcaccount:")
	}
	if strings.TrimSpace(value[len(dcAccountScheme):]) == "" {
		return errors.New("Delta Chat account setup requires a complete chatmail dcaccount URL; provide --dcaccount-url, DELTAOPS_DCACCOUNT_URL, or delta_chat.dcaccount_url")
	}
	return nil
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
