# Implement pairing-code contact binding

## Summary

Bind DeltaOps to the first Delta Chat contact that proves possession of the setup code printed by the bot, then send all future alerts to that contact.

## Requirements

- When no recipient is bound, print or expose a one-time setup code.
- Bind only to the first inbound message from a contact that contains the valid setup code.
- Persist the selected contact in local state.
- Reuse the persisted binding on restart.
- Ignore or safely reject messages without the setup code before binding.
- Ignore or safely reject later contacts that are not bound.
- Provide a deliberate reset path for rebinding.

## Acceptance Criteria

- Tests prove first valid setup-code message wins.
- Tests prove messages without the setup code cannot bind.
- Tests prove the binding survives restart.
- Tests prove messages from later contacts cannot steal the binding.
- The reset behavior is documented and tested.

## Notes

- Depends on the Delta Chat integration, account provisioning, and state layout decisions.
- Keep binding logic independent from the Delta Chat adapter so race cases are testable.
