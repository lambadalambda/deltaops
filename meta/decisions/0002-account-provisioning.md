# 0002 Account provisioning flow

## Status

Accepted for MVP planning.

## Context

DeltaOps should be low setup, but provider-neutral arbitrary email account creation is not a stable Delta Chat protocol feature. The integration decision selected `chatmail/rpc-client-go/v2`, where account transport setup can consume a chatmail account setup URL through `AddTransportFromQr`.

Chatmail provider homepages can publish Delta Chat account links. For example, `https://nine.testrun.org/` links to `DCACCOUNT:https://nine.testrun.org/new`.

## Decision

The MVP account provisioning input is either a chatmail provider URL supplied explicitly by the operator, or a full chatmail `dcaccount:` setup URL.

Provider homepage URLs are normalized to the Delta Chat account setup form before transport setup. For example, `https://nine.testrun.org/` is passed to Delta Chat core as `DCACCOUNT:https://nine.testrun.org/new`.

Supported input sources, in precedence order:

1. CLI flag: `--dcaccount-url https://nine.testrun.org/` or `--dcaccount-url dcaccount:...`
2. Environment variable: `DELTAOPS_DCACCOUNT_URL=https://nine.testrun.org/` or `DELTAOPS_DCACCOUNT_URL=dcaccount:...`
3. Config file key: `delta_chat.dcaccount_url: https://nine.testrun.org/` or `delta_chat.dcaccount_url: dcaccount:...`

If no input is provided, startup should fail with a clear error telling the operator to provide one of those inputs. If the input is neither a supported `dcaccount:` setup URL nor an HTTPS chatmail provider URL, startup should reject it without echoing the supplied value.

DeltaOps does not implement provider-specific signup scraping or direct IMAP/SMTP credential generation. Existing IMAP/SMTP credentials and OAuth-based setup are deferred until there is a concrete need and a tested Delta Chat API path.

After successful setup, DeltaOps should print only operator-facing contact data: the bot Delta Chat contact or secure-join URI, the bot email address if available, and the local pairing code. It should not print the account setup input after consuming it.

## Consequences

- The first-run UX remains low setup for operators who know a chatmail provider URL.
- DeltaOps does not try to automate arbitrary email provider signup flows.
- The CLI and config implementation must preserve the input precedence selected here.
- Provisioning validation can be tested without live Delta Chat by validating source resolution and supported URL schemes.
