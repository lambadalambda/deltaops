# AGENTS.md

Guidance for AI agents and contributors working in this repository.

## Project Rules

- Language: Go.
- Toolchain: use `mise`; keep the Go version pinned in `.mise.toml`.
- Delivery shape: one portable `deltaops` binary.
- Notification transport: Delta Chat only.
- Process: test-driven development using red, green, refactor.
- Commits: small, topical commits only; after a reviewed slice has no blocking findings, commit it without asking again.
- Reviews: after each finished feature, request a review from one of the code reviewer subagents before committing.
- Planning: track work in repository-local issues under `meta/` using the repo-issues workflow.

## Engineering Defaults

- Keep changes small and focused on one issue or feature at a time.
- Write tests before implementation for behavior changes.
- Prefer interfaces at external boundaries such as Delta Chat, host metrics, clock, filesystem, and process signals.
- Keep core alerting behavior independent from the Delta Chat transport so it can be tested with fakes.
- Make alert messages actionable: include host, check, observed value, threshold, and recovery state.
- Avoid noisy repeat alerts; use cooldowns and recovery notifications.
- Treat account credentials, binding state, and local config as sensitive data.
- Do not implement host monitoring features until the Delta Chat single-binary integration and account provisioning spikes have a recorded decision.

## Package Layout

- `cmd/deltaops/`: CLI entrypoint.
- `internal/alert/`: threshold evaluation, alert state, cooldowns, and recovery decisions.
- `internal/binding/`: pairing-code and bound-contact state logic.
- `internal/collector/`: host metric collection.
- `internal/config/`: config parsing, defaults, validation, and state path resolution.
- `internal/notify/`: Delta Chat transport interface and implementation.
- `internal/runtime/`: polling loop, signal handling, and orchestration.

## Reviews

- Before committing a finished feature, invoke a code reviewer subagent with the changed files, issue link, acceptance criteria, and verification results.
- Prefer `code-reviewer` unless there is a concrete reason to choose `code-reviewer-kimi`, `code-reviewer-deepseek`, or another available reviewer.
- Address review findings before committing, or document why a finding is intentionally deferred.

## Verification

- Run `mise exec -- go test ./...` before considering a Go change complete.
- Add narrower package tests for any new behavior.
- If adding CLI behavior, include tests around flags, config resolution, and error paths where practical.
- Keep live Delta Chat or provider-dependent tests out of the default test path unless they are hermetic; use build tags or explicit integration-test commands for external services.
- If verification is skipped or impossible, state that clearly in the final response.

## Repository Issues

- Open issue index: `meta/issues.md`.
- Completed issue index: `meta/issues_archive.md`.
- Detail files: `meta/issues/*.md`.
- Keep index files as checklist links only.
- Do not archive an issue until its acceptance criteria are satisfied.
