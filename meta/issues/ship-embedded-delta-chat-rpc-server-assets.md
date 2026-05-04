# Ship embedded Delta Chat RPC server assets

## Summary

Make the `chatmail/rpc-client-go/v2` plus embedded `deltachat-rpc-server` path the product track for the MVP release, replacing the current placeholder/helper-missing development behavior with real versioned helper assets.

## Requirements

- Keep `github.com/chatmail/rpc-client-go/v2` as the supported Delta Chat transport client.
- Embed platform-specific `deltachat-rpc-server` assets into release binaries instead of requiring operator-installed sidecars.
- Pin the helper asset version to the Go RPC client version used by the module.
- Do not silently download helper binaries at runtime.
- Extract helper assets to a restrictive local path and execute them as child processes.
- Keep default tests hermetic and independent of live Delta Chat services.

## Acceptance Criteria

- Supported release targets have documented helper asset names and source checksums.
- `deltaops run` uses an embedded helper on supported targets instead of returning the current missing-helper error.
- Extraction permissions and `DC_ACCOUNTS_PATH` handling are covered by tests.
- The README documents how release helpers are sourced, verified, embedded, and updated.
- `mise exec -- go test ./...` passes.

## Notes

- This is the primary product path. C FFI or other Delta Chat bindings are deferred to a separate research issue.
- Runtime helper downloads are intentionally out of scope for the MVP.
