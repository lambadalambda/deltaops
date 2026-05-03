# Document operation, security, and hardening

## Summary

Document how to run DeltaOps safely and harden behavior for real hosts.

## Requirements

- Document setup, pairing, config, state, and reset behavior.
- Document service manager examples for the supported MVP operating system.
- Document the accepted security model for local state, account credentials, pairing, and alert content.
- Add structured logs that avoid leaking credentials or message contents.
- Handle shutdown cleanly.
- Describe backup and rotation considerations for local state.

## Acceptance Criteria

- README contains an end-to-end quickstart after the CLI exists.
- Operational docs include a service example.
- Security considerations explain plaintext credential risks or the selected credential storage mechanism.
- Tests or manual verification cover graceful shutdown.
- Logs do not include secrets in normal operation.

## Notes

- Keep this issue open until there is real behavior to document.
