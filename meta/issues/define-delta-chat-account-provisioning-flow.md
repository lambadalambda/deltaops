# Define Delta Chat account provisioning flow

## Summary

Define the first-run Delta Chat account setup flow honestly, including whether automatic registration is possible for the MVP.

## Requirements

- Identify the supported provisioning inputs, such as chatmail `dcaccount:` URLs, existing IMAP/SMTP credentials, OAuth tokens, or another Delta Chat-supported mechanism.
- Avoid generic provider web scraping or any flow that depends on bypassing CAPTCHAs, terms of service, phone verification, or rate limits.
- Define how provisioning inputs are supplied: flags, environment variables, stdin prompts, config file, or local setup command.
- Define the failure mode when no supported account input is available.
- Define what contact data is printed after setup.

## Acceptance Criteria

- The README describes the selected provisioning flow and any fallback.
- Tests cover config validation for required provisioning inputs using fakes.
- The decision states whether true automatic account registration is supported, constrained to specific providers, or deferred.
- Error messages give a clear next action when account setup cannot proceed.

## Notes

- Depends on the Delta Chat integration spike.
- Keep the UX low setup, but do not pretend email account creation is provider-neutral.

## Resolution

- Decision recorded in `../decisions/0002-account-provisioning.md`.
- MVP supports explicit operator-provided chatmail `dcaccount:` URLs only.
- Input precedence is CLI flag `--dcaccount-url`, then environment variable `DELTAOPS_DCACCOUNT_URL`, then config key `delta_chat.dcaccount_url`.
- Generic provider signup, existing IMAP/SMTP credentials, and OAuth setup are deferred until a concrete Delta Chat API path is implemented and tested.
- After setup, DeltaOps should print the bot contact or secure-join URI, bot email address if available, and the local pairing code, but not the consumed provisioning URL.
- Validation is implemented in `internal/config` and covered by fake input-source tests.
