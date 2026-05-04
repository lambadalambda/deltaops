# Build GitHub release binaries

## Summary

Create a public GitHub repository and a GitHub Actions workflow that builds DeltaOps release binaries for Linux and macOS.

## Requirements

- Create a public GitHub repository for this project.
- Add a release workflow that runs tests before building release artifacts.
- Build release binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`.
- Embed the matching `deltachat-rpc-server` helper in each release binary.
- Upload build artifacts for branch/manual runs.
- Publish a GitHub Release with artifacts when a `v*` tag is pushed.

## Acceptance Criteria

- The repository exists publicly on GitHub.
- The default branch is pushed to GitHub.
- A `v*` tag triggers the release workflow.
- The workflow publishes Linux and macOS release artifacts with checksums.
- The workflow result is checked after pushing.

## Notes

- Helper binaries are downloaded and checksum-verified during the workflow; they are not committed to the repository.
