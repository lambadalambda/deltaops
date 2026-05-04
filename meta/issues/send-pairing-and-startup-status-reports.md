# Send pairing and startup status reports

## Summary

Send an operator-facing Delta Chat confirmation after pairing succeeds, and send a status report on startup when an alert contact is already bound.

## Requirements

- Send a confirmation/status report immediately after a new contact sends the valid pairing code.
- Send a startup status report when DeltaOps starts with an existing bound contact.
- Include all currently collected metric samples in the report.
- Keep normal threshold alerts and recovery notifications unchanged.
- Keep reports free of provisioning URLs, setup codes, and raw inbound message text.

## Acceptance Criteria

- Runtime tests cover a report after first-time pairing.
- Runtime tests cover a report on startup with an existing binding.
- Report message text includes a reason, host, metric names, targets, and observed values.
- Report send failures use bounded retry behavior like alert notifications.
- `mise exec -- go test ./...` passes.

## Notes

- This is a liveness/usability feature: a quiet monitor should still confirm that pairing worked and show what it is currently observing.
