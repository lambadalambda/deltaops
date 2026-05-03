# DeltaOps

DeltaOps is planned as a small Go system monitor that sends alerts through Delta Chat. The target deployment model is one portable `deltaops` binary that can be copied onto a host, run with minimal setup, and paired by messaging the bot from the operator's Delta Chat account.

Status: Delta Chat integration, account provisioning, state layout, pairing logic, and MVP metric-source decisions are in place. The monitor is not runnable yet.

## Goals

- Build one portable Go binary for low-setup host monitoring.
- Use Delta Chat as the only notification transport.
- Create, register, or configure the bot's Delta Chat account on first run with as little operator input as the selected provider flow allows.
- Print the bot contact data and a one-time pairing code during setup.
- Bind alerts to the first contact that sends the valid pairing code, then persist that binding.
- Alert on practical host-health failures such as disk exhaustion, high memory pressure, and other checks selected for the MVP.
- Prefer safe defaults, clear recovery messages, and low alert noise.

## Delta Chat Integration

- DeltaOps will use `github.com/chatmail/rpc-client-go/v2` and a managed `deltachat-rpc-server` subprocess for the MVP path.
- Release builds should keep the operator experience to one copied `deltaops` file by embedding the matching platform-specific RPC server helper and extracting it at runtime.
- This is not a pure-Go binary internally. Each supported OS and architecture needs a matching Delta Chat RPC server asset.
- The MVP-supported account setup path is explicit operator input via a chatmail `dcaccount:` URL. Provider-neutral email account auto-registration is out of scope unless a provider offers a documented automation flow.
- The full decision is recorded in `meta/decisions/0001-delta-chat-integration.md`.

## Account Provisioning

The MVP setup input is a chatmail `dcaccount:` URL. DeltaOps will accept it from these sources, in order:

1. `--dcaccount-url dcaccount:...`
2. `DELTAOPS_DCACCOUNT_URL=dcaccount:...`
3. `delta_chat.dcaccount_url` in the config file

If none is provided, startup should fail with a next action telling the operator to provide one of those inputs. Existing IMAP/SMTP credentials and OAuth setup are deferred, not fallback behavior for the MVP.

After account setup, DeltaOps should print the bot Delta Chat contact or secure-join URI, the bot email address if available, and the local pairing code. It should not print the consumed `dcaccount:` URL.

The provisioning decision is recorded in `meta/decisions/0002-account-provisioning.md`.

## Config And State

DeltaOps should run without a config file when required setup input is supplied by flag or environment.

Default paths, using absolute XDG locations only:

1. Config file: `$XDG_CONFIG_HOME/deltaops/config.yaml`, or `$HOME/.config/deltaops/config.yaml` when `XDG_CONFIG_HOME` is unset.
2. State directory: `$XDG_STATE_HOME/deltaops`, or `$HOME/.local/state/deltaops` when `XDG_STATE_HOME` is unset.
3. Delta Chat accounts: `<state>/deltachat-accounts`.
4. Bound contact: `<state>/binding.json`.

State paths should be overrideable by future CLI/config wiring. State directories are created with `0700` permissions and sensitive files with `0600` permissions on POSIX-style filesystems where supported.

MVP warning: local Delta Chat state is accepted as plaintext on disk behind filesystem permissions. That state can include account credentials, message databases, and contact binding data. Protect and back up the state directory accordingly.

The state layout decision is recorded in `meta/decisions/0003-config-and-state-layout.md`.

## Pairing And Reset

When no contact is bound, DeltaOps should show a one-time setup code alongside the bot contact data. The first inbound message containing that setup code becomes the alert recipient. Messages without the setup code are ignored while unbound, and messages from later contacts cannot replace an existing binding.

The binding is stored at `<state>/binding.json`. The MVP reset path is deliberate local access: stop DeltaOps, delete `<state>/binding.json`, then restart with a new setup code. Future CLI wiring can expose the same reset operation as a command.

## Platform And Metrics

The MVP collector target is Linux only. Collector plan creation fails at runtime with a clear error on unsupported operating systems before collector startup.

Default metrics planned for the MVP:

1. `disk.used_percent`: `100 * (Blocks - Bfree) / Blocks` from `statfs`; unavailable when `Blocks == 0`.
2. `disk.inodes_used_percent`: `100 * (Files - Ffree) / Files` from `statfs`; unavailable when `Files == 0`.
3. `memory.pressure_percent`: `100 * (MemTotal - MemAvailable) / MemTotal` from `/proc/meminfo`; unavailable when `MemTotal == 0` or `MemAvailable` is missing.
4. `load.1m`: first field of `/proc/loadavg`; unavailable when the file is missing or unparsable.

CPU utilization is deferred for the MVP; load average is the initial CPU-pressure signal. Linux release builds must include matching embedded Delta Chat RPC server assets for their target architecture.

The `internal/collector` package implements these Linux samples behind fakeable filesystem and `/proc` interfaces. Unavailable metric sources return clear unavailable errors instead of panics.

The platform and metric-source decision is recorded in `meta/decisions/0004-mvp-platform-and-metrics.md`.

## Non-Goals

- Replacing full observability stacks such as Prometheus, Grafana, or agent-based SaaS platforms.
- Supporting many notification transports.
- Managing multiple operators in the first version.
- Shipping a daemon supervisor. Service manager examples can be documented later.

## Intended UX

```sh
deltaops
```

On first run, DeltaOps should:

1. Create or load its local state directory.
2. Configure a Delta Chat account using a supplied chatmail `dcaccount:` URL from flag, environment, or config.
3. Print contact data and a one-time pairing code for the bot.
4. Wait for the first incoming message that contains the pairing code.
5. Persist that sender as the alert recipient.
6. Start sending alerts and recovery notices for configured checks.

After binding, later messages from other contacts should not receive host alerts unless the local reset flow is used.

## Security Considerations

- Local state may contain Delta Chat account credentials, message databases, and the bound contact. It must be stored outside the repository with restrictive permissions.
- The MVP accepts plaintext local state protected by filesystem permissions; OS keyring integration is deferred.
- The pairing code prevents a random first sender from taking over alert delivery during setup.
- Alert messages can reveal hostnames, resource pressure, and operational state. The monitor should send them only to the persisted bound contact.
- If Delta Chat delivery is unavailable, DeltaOps should log locally, retry with backoff, and avoid unbounded queues.

## Development

This repository uses `mise` to pin the Go toolchain.

```sh
mise install
mise exec -- go test ./...
```

If `mise` reports that `.mise.toml` is not trusted, review the file and run:

```sh
mise trust .mise.toml
```

Build command once the CLI exists:

```sh
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

## Planning

The local issue plan lives in `meta/issues.md`, with issue details in `meta/issues/`.

The first implementation steps recorded the Delta Chat integration path, single-binary constraint, account provisioning flow, config/state layout, pairing-code contact binding, and MVP metric-source decision. The next open issue is collecting disk, memory, and load metrics.
