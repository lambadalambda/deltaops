# Support chatmail provider URL provisioning

## Summary

Allow operators to provide a chatmail provider URL such as `https://nine.testrun.org/` instead of requiring them to manually extract a full `dcaccount:` setup URL.

## Requirements

- Keep existing `dcaccount:` setup URLs working unchanged.
- Accept HTTPS chatmail provider URLs from the existing flag, environment variable, and config file sources.
- Normalize a provider homepage URL to the Delta Chat account setup form expected by `AddTransportFromQr`.
- Do not implement provider-specific signup scraping or custom credential handling.
- Do not print or log generated account setup inputs.

## Acceptance Criteria

- `https://nine.testrun.org/` normalizes to `DCACCOUNT:https://nine.testrun.org/new` before Delta Chat account setup.
- Existing `dcaccount:` inputs remain accepted.
- Unsupported schemes are rejected without echoing the provided value.
- CLI and config tests cover provider URL input.
- README and the account provisioning decision document the provider URL path.
- `mise exec -- go test ./...` passes.

## Notes

- `https://nine.testrun.org/` publishes `DCACCOUNT:https://nine.testrun.org/new` on its homepage. Delta Chat core should perform the provider account request through the existing QR/account setup path.
