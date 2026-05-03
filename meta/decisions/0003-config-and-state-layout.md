# 0003 Configuration and state layout

## Status

Accepted for MVP planning.

## Context

DeltaOps needs defaults that work without a config file while keeping account credentials, Delta Chat account databases, and bound-contact state out of the repository.

## Decision

Use absolute XDG-style per-user paths by default:

- Config file: `$XDG_CONFIG_HOME/deltaops/config.yaml`, or `$HOME/.config/deltaops/config.yaml` when `XDG_CONFIG_HOME` is unset.
- State directory: `$XDG_STATE_HOME/deltaops`, or `$HOME/.local/state/deltaops` when `XDG_STATE_HOME` is unset.
- Delta Chat accounts directory: `<state>/deltachat-accounts`.
- Contact binding file: `<state>/binding.json`.

Support explicit config and state path overrides from future CLI/config wiring. Relative `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, and `HOME` values are rejected so defaults do not accidentally resolve inside the repository or current working directory. State directories are created with `0700` permissions and sensitive files with `0600` permissions where the platform supports POSIX permissions.

Accept plaintext local Delta Chat state for the MVP, protected by filesystem permissions. OS keyring storage is deferred because Delta Chat core still needs a local accounts database and message state on disk; a keyring would not remove the need to protect that directory. This decision should be revisited before storing additional secrets outside Delta Chat core state.

## Consequences

- Defaults run without a config file when required setup input is supplied by flag or environment.
- The default state path is outside the repository in normal project checkouts.
- Backups of the state directory can include account credentials and message data.
- Operators should protect host filesystem access and treat the state directory as sensitive.
