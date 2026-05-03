# 0002 Account provisioning flow

## Status

Accepted for MVP planning.

## Context

DeltaOps should be low setup, but provider-neutral email account creation is not a stable Delta Chat protocol feature. The integration decision selected `chatmail/rpc-client-go/v2`, where account transport setup can consume a chatmail `dcaccount:` URL.

## Decision

The MVP account provisioning input is a chatmail `dcaccount:` URL supplied explicitly by the operator.

Supported input sources, in precedence order:

1. CLI flag: `--dcaccount-url dcaccount:...`
2. Environment variable: `DELTAOPS_DCACCOUNT_URL=dcaccount:...`
3. Config file key: `delta_chat.dcaccount_url: dcaccount:...`

If no input is provided, startup should fail with a clear error telling the operator to provide one of those inputs. If the input does not start with `dcaccount:`, startup should reject it without echoing the supplied value.

True automatic account registration is not supported for the MVP. Existing IMAP/SMTP credentials and OAuth-based setup are deferred until there is a concrete need and a tested Delta Chat API path.

After successful setup, DeltaOps should print only operator-facing contact data: the bot Delta Chat contact or secure-join URI, the bot email address if available, and the local pairing code. It should not print the `dcaccount:` URL after consuming it.

## Consequences

- The first-run UX remains low setup for operators who have a chatmail provisioning URL.
- DeltaOps does not try to automate arbitrary email provider signup flows.
- The CLI and config implementation must preserve the input precedence selected here.
- Provisioning validation can be tested without live Delta Chat by validating source resolution and supported URL schemes.
