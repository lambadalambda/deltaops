# Package the CLI as a portable binary

## Summary

Provide the command-line entrypoint and build path for copying DeltaOps onto a monitored host.

## Requirements

- Add a `cmd/deltaops` entrypoint.
- Provide `run`, `version`, and useful startup error behavior.
- Build a single binary for the supported MVP platforms.
- Include version metadata when practical.
- Keep runtime dependencies explicit and minimal.
- If Delta Chat requires cgo, native libraries, or helper binaries, document the packaging impact and revisit the single-binary requirement before implementation proceeds.

## Acceptance Criteria

- `mise exec -- go test ./...` passes.
- `mise exec -- go build -o bin/deltaops ./cmd/deltaops` produces the binary.
- The binary starts with safe defaults or prints a clear next action.
- Packaging instructions are documented.
- Cross-compilation constraints are documented for every supported platform.

## Notes

- Revisit this issue after the Delta Chat integration spike decides whether cgo or native linking is involved.
