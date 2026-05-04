# Docker End-to-End Harness

This directory contains an opt-in Docker harness for exercising DeltaOps with an embedded Linux `deltachat-rpc-server` helper. It is not part of default `go test ./...`.

## Smoke Test

From the repository root:

```sh
sh scripts/e2e-docker.sh smoke
```

The smoke test:

1. Downloads and checksum-verifies the Linux RPC helper for the selected target.
2. Cross-builds `deltaops` with the helper embedded.
3. Builds a minimal Debian runtime image.
4. Runs DeltaOps with `dcaccount:placeholder`.

The expected result is a startup failure from Delta Chat account configuration, not a missing-helper error. That proves the container ran the embedded helper far enough to reach live transport provisioning.

## Live Run

Use a real chatmail provisioning URL from an ignored env file:

```sh
install -m 600 /dev/null test/e2e/.env
printf 'DELTAOPS_DCACCOUNT_URL=https://nine.testrun.org/\n' > test/e2e/.env
sh scripts/e2e-docker.sh live
```

The container prints the bot contact data and setup code. Pair it from the operator Delta Chat account by sending the setup code to the bot. Docker environment metadata is sensitive while the live container exists; reset the harness when done.

## Reset

Remove disposable Docker state with:

```sh
sh scripts/e2e-docker.sh reset
```

Live state is stored in the `deltaops-e2e_deltaops-state` Docker volume. Smoke state uses `deltaops-e2e-smoke_deltaops-state` and is cleaned automatically. Do not put real `dcaccount:` URLs, provider URLs with tokens, account databases, or binding files in git.

## Target Selection

The script defaults to `linux/arm64` on Apple Silicon and `linux/amd64` on x86_64 hosts. Override with:

```sh
DELTAOPS_E2E_TARGET=linux/amd64 sh scripts/e2e-docker.sh smoke
```

Supported Docker targets are `linux/amd64` and `linux/arm64`.

## Remaining Gap

This harness does not yet automate an alert and recovery notification. A complete end-to-end test still needs either a hermetic Delta Chat peer or a documented live-provider flow that can pair the bot and force metric threshold crossings repeatably.
