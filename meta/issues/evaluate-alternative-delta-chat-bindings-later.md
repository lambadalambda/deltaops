# Evaluate alternative Delta Chat bindings later

## Summary

Defer research into non-RPC Delta Chat bindings until the embedded `deltachat-rpc-server` product path is working end-to-end.

## Requirements

- Do not block MVP product work on C FFI or other alternative bindings.
- Evaluate `https://github.com/hugot/go-deltachat` and any maintained alternatives only as a later spike.
- Compare alternatives against the embedded RPC helper path using packaging, maintenance, API coverage, provisioning support, cross-compilation, and release operational complexity.
- Record any future decision in `meta/decisions/`.

## Acceptance Criteria

- The spike verifies maintenance status and Delta Chat core version freshness for each alternative considered.
- The spike confirms whether current `dcaccount:` provisioning is supported.
- Build feasibility is tested for at least Linux and macOS target architectures relevant to DeltaOps.
- The decision records whether to keep the RPC helper path or switch, with explicit tradeoffs.

## Notes

- This issue intentionally stays behind the embedded RPC server work so the project remains focused on a shippable product.
