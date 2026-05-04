# Add macOS development support

## Summary

Support running and testing DeltaOps on macOS so local development can exercise the same product path as release builds.

## Requirements

- Add `darwin/arm64` support for the embedded `deltachat-rpc-server` helper path.
- Add `darwin/amd64` support if upstream helper assets are available and practical.
- Add macOS metric collection or a clearly documented macOS development mode that can exercise runtime and notification behavior without Linux collectors.
- Preserve existing Linux behavior and tests.
- Keep platform-specific logic behind narrow package boundaries.

## Acceptance Criteria

- `mise exec -- go test ./...` passes on macOS.
- `mise exec -- go build -o bin/deltaops ./cmd/deltaops` succeeds on macOS.
- `bin/deltaops run` on supported macOS architectures reaches the normal provisioning/helper/runtime path instead of failing with unsupported-platform before transport setup.
- macOS support is documented in the README, including any limitations compared to Linux.

## Notes

- Local macOS support is for developer confidence and testing convenience. Linux remains the first deployment target unless a later decision expands the MVP target set.
