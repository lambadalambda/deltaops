# 0001 Delta Chat integration path

## Status

Accepted for MVP planning.

## Context

DeltaOps needs Delta Chat notifications from Go while preserving a one-file, low-setup operator experience. The main risks are Delta Chat runtime availability, account provisioning, and cross-platform packaging.

## Findings

- The maintained Go path is `github.com/chatmail/rpc-client-go/v2`.
- The Go client talks JSON-RPC over standard I/O to a separate `deltachat-rpc-server` executable.
- The RPC server is provided by Delta Chat core releases as platform-specific binaries.
- The RPC server stores account data in the current directory unless `DC_ACCOUNTS_PATH` is set.
- The Go client example creates an account with RPC calls, then configures transport from a supplied `dcaccount:` URL using `AddTransportFromQr`.
- Provider-neutral email account auto-registration is not a Delta Chat protocol feature and should not be implemented by scraping provider signup flows.
- A Docker spike ran the upstream `deltachat-rpc-server-aarch64-linux` release asset and verified `deltachat-rpc-server --version` returned `2.49.0`.
- Account configuration was not exercised end-to-end because this spike did not have a usable `dcaccount:` provisioning URL. The blocker is external account provisioning input, not the local RPC server process.

## Decision

Use `chatmail/rpc-client-go/v2` with a managed `deltachat-rpc-server` subprocess for the MVP path.

DeltaOps should keep the user-facing distribution as one `deltaops` binary by embedding the matching platform-specific RPC server asset into each release binary, extracting it to a restrictive state or cache path at runtime, and executing it as a child process. This keeps the operator workflow to one copied file, but the implementation is not a pure-Go binary.

The MVP account provisioning path is an explicit operator-provided chatmail `dcaccount:` URL. Existing IMAP/SMTP credentials can be evaluated later as a fallback, but they are not the first supported setup path. Generic automatic provider signup is out of scope unless a supported provider offers a documented automation flow.

The monitoring core must depend only on a small notification interface. Live Delta Chat tests should be explicit integration tests; default tests should use fakes.

## Consequences

- Each supported OS and architecture needs a matching embedded RPC server asset and release build.
- Cross-compilation requires fetching or vendoring the matching RPC server artifact for the target, not just running `GOOS` and `GOARCH`.
- The Go client and RPC server versions must be pinned together.
- Binary size will include the Go program plus a roughly 10 to 30 MiB RPC server helper.
- Runtime state must include Delta Chat account data under a controlled `DC_ACCOUNTS_PATH` with restrictive permissions.
- If embedding the helper is rejected later, requiring `deltachat-rpc-server` in `PATH` remains a developer fallback but does not satisfy the one-file operator goal.
- The next account provisioning issue should exercise `AddTransportFromQr` with a real or hermetic `dcaccount:` URL.

## Alternatives Considered

- Pure-Go SMTP and IMAP implementation: rejected for MVP because it would not provide the tested Delta Chat core behavior and would risk incompatibility with encryption, contact, and message semantics.
- cgo or Rust library embedding: deferred because it adds cross-compilation, linking, and toolchain complexity without a clearer path to a portable Go release.
- External sidecar installed by the operator: useful for development and integration tests, but it fails the low-setup single-file requirement.
