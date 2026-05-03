# DeltaOps

DeltaOps is planned as a small Go system monitor that sends alerts through Delta Chat. The target deployment model is one portable `deltaops` binary that can be copied onto a host, run with minimal setup, and paired by messaging the bot from the operator's Delta Chat account.

Status: repository scaffold and implementation plan. The monitor is not runnable yet.

## Goals

- Build one portable Go binary for low-setup host monitoring.
- Use Delta Chat as the only notification transport.
- Create, register, or configure the bot's Delta Chat account on first run with as little operator input as the selected provider flow allows.
- Print the bot contact data and a one-time pairing code during setup.
- Bind alerts to the first contact that sends the valid pairing code, then persist that binding.
- Alert on practical host-health failures such as disk exhaustion, high memory pressure, and other checks selected for the MVP.
- Prefer safe defaults, clear recovery messages, and low alert noise.

## Feasibility Risks

- Delta Chat's mature implementation is not pure Go. The first project spike must prove whether a Go binary can embed, link, or manage the required Delta Chat runtime without breaking the single-binary goal.
- Automatic account registration is provider-specific. The first implementation path may require a chatmail `dcaccount:` URL, existing IMAP/SMTP credentials, or another explicit account provisioning input.
- Monitoring work should wait until the Delta Chat and account-provisioning spikes decide whether the desired setup flow is realistic.

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
2. Register or configure a Delta Chat account using the selected provisioning mechanism.
3. Print contact data and a one-time pairing code for the bot.
4. Wait for the first incoming message that contains the pairing code.
5. Persist that sender as the alert recipient.
6. Start sending alerts and recovery notices for configured checks.

After binding, later messages from other contacts should not receive host alerts unless an explicit reset flow is used.

## Security Considerations

- Local state may contain Delta Chat account credentials, message databases, and the bound contact. It must be stored outside the repository with restrictive permissions.
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

The first implementation step should prove the Delta Chat integration path for Go and the single-binary constraint before committing to the rest of the architecture.
