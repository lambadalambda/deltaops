# Implement configuration and state layout

## Summary

Define how DeltaOps finds configuration, stores local state, and protects sensitive files.

## Requirements

- Choose default config and state locations.
- Support explicit config and state path overrides.
- Store Delta Chat account data and contact binding separately from non-sensitive config where practical.
- Create state directories with restrictive permissions.
- Create state directories with `0700` permissions and sensitive files with `0600` permissions where the platform supports it.
- Evaluate OS keyring or machine-local credential storage for account secrets, or document why plaintext state is accepted for the MVP.
- Validate config with clear startup errors.

## Acceptance Criteria

- Tests cover default path resolution and overrides.
- Tests cover invalid config errors.
- Sensitive state files are created with restrictive permissions where the platform supports it.
- The default state path is outside the repository.
- The README warns if credentials are stored in plaintext.
- The README documents config and state paths.

## Notes

- Keep the first config format minimal; defaults should run without a config file.
- This issue should be completed before binding, metric collection, or runtime-loop work that needs persistent state.
