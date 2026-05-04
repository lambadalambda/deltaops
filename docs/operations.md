# Operations Guide

This guide covers running DeltaOps on a host after you have downloaded an official release binary.

## Account Setup

DeltaOps needs a Delta Chat account setup source on startup. You can provide either a chatmail provider URL or a full `dcaccount:` setup URL.

Supported sources, in order:

1. `--dcaccount-url https://nine.testrun.org/`
2. `DELTAOPS_DCACCOUNT_URL=https://nine.testrun.org/`
3. `delta_chat.dcaccount_url` in the config file

Provider homepage URLs with an empty or `/` path are normalized to the provider's account creation endpoint. For example, `https://nine.testrun.org/` becomes `DCACCOUNT:https://nine.testrun.org/new` internally.

Full `dcaccount:` setup URLs can contain account setup secrets. Do not put them in shell history, world-readable unit files, tickets, or logs.

The current CLI still requires the setup input on restarts, even when the local Delta Chat account already exists. Keep the value available in the service environment or config file.

## First Run Flow

1. Install the official `deltaops` release binary for your OS and architecture.
2. Create a private state directory, such as `/var/lib/deltaops`, owned by the service user with mode `0700`.
3. Start `deltaops run --state-dir /var/lib/deltaops` with a provider URL from flag, environment, or config.
4. Read the startup output for the bot contact details and one-time pairing code.
5. Send the pairing code to the bot from the Delta Chat account that should receive alerts.
6. Leave DeltaOps running under a service manager.

After pairing, DeltaOps sends a `reason=paired` status report. On later restarts with the same binding, it sends a `reason=startup` status report.

## Config And State

DeltaOps can run without a config file when required setup input is supplied by flag or environment.

Default paths:

| Purpose | Path |
| --- | --- |
| Config file | `$XDG_CONFIG_HOME/deltaops/config.yaml`, or `$HOME/.config/deltaops/config.yaml` |
| State directory | `$XDG_STATE_HOME/deltaops`, or `$HOME/.local/state/deltaops` |
| Delta Chat accounts | `<state>/deltachat-accounts` |
| Extracted RPC helper | `<state>/deltachat-rpc-helper` |
| Bound alert recipient | `<state>/binding.json` |

The CLI supports `--config` and `--state-dir` overrides. State directories are created with `0700` permissions and sensitive files with `0600` permissions where the filesystem supports it.

Local Delta Chat state can include account credentials, message databases, and contact binding data. Treat the whole state directory as sensitive.

## Pairing And Reset

When no contact is bound, DeltaOps waits for the first inbound message containing the one-time pairing code. That sender becomes the alert recipient. Other messages are ignored while unbound, and later contacts cannot replace the saved binding.

To reset only the alert recipient:

1. Stop DeltaOps.
2. Delete `<state>/binding.json`.
3. Start DeltaOps again.
4. Send the new pairing code from the new recipient account.

Do not delete the whole state directory unless you also want to remove the Delta Chat bot account and provision a new one.

## Status Reports And Alerts

Status reports include host, reason, metric names, targets, and observed values for the samples collected at that moment. Status reports are separate from threshold alerts and do not suppress later alert or recovery messages.

Alert and recovery messages include host, check, target, observed value, threshold, severity, and state.

Default thresholds:

| Metric | Warning | Critical |
| --- | ---: | ---: |
| `disk.used_percent` | `85` | `95` |
| `disk.inodes_used_percent` | `85` | `95` |
| `memory.pressure_percent` | `80` | `90` |
| `load.1m` | `1` | `2` |

The default repeat cooldown is `30m`. The load defaults are conservative absolute-load values and should be tuned for larger multi-core hosts when configuration support grows.

## Linux Service Example

Use a dedicated service user and a private environment file rather than putting setup values directly in the unit.

Example `/etc/deltaops/deltaops.env`:

```sh
DELTAOPS_DCACCOUNT_URL=https://nine.testrun.org/
```

Restrict the file to the service user with mode `0600`.

Example systemd unit:

```ini
[Unit]
Description=DeltaOps host monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=deltaops
Group=deltaops
EnvironmentFile=/etc/deltaops/deltaops.env
ExecStart=/usr/local/bin/deltaops run --state-dir /var/lib/deltaops
Restart=on-failure
RestartSec=30s
StateDirectory=deltaops
StateDirectoryMode=0700
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/deltaops

[Install]
WantedBy=multi-user.target
```

`SIGINT` and `SIGTERM` cancel the runtime loop cleanly.

## Backups And Rotation

Back up the state directory only to encrypted storage. Stop DeltaOps before file-level backups so Delta Chat account databases and binding files are consistent. Restore the directory with the original owner and restrictive permissions before restarting the service.

To rotate the Delta Chat bot account, stop DeltaOps, back up or delete the existing state directory, provision with a new provider or `dcaccount:` URL, and pair the new bot contact.

To rotate only the alert recipient, delete `binding.json` and pair a new contact.

If logs go to journald, log rotation is handled by the service manager. If logs are redirected to files, rotate them normally and restrict file permissions because alert metadata can reveal operational state.

## Troubleshooting

If startup says no account setup URL is available, provide `--dcaccount-url`, `DELTAOPS_DCACCOUNT_URL`, or `delta_chat.dcaccount_url`.

If startup says the Delta Chat RPC helper is not packaged, you are likely running a local build that did not prepare helper assets. Official release binaries include the matching helper. For local builds, see [development](development.md).

If delivery fails, check local logs, network reachability, and provider availability. DeltaOps retries with bounded backoff and avoids unbounded notification queues.
