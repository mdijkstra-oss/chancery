---
requires:
  - resolve
---

# Responding to delegated work

When you finish work — or can't start it — you respond using resolve or reject.

## resolve(outcome, unresolved, artifacts)

Use resolve when you've done work, even if not everything is complete.

- **outcome** — What was accomplished. Clear enough that the caller can judge success without inspecting artifacts themselves.
- **unresolved** — What couldn't be completed and why. Empty if everything is done. Be specific enough that the caller can decide to retry, delegate elsewhere, or ask the user. Include what would be needed to resolve the remaining items.
- **artifacts** — Files created or modified. The caller uses these to pass results to the next task or present to the user.

Partial completion is a resolve with unresolved items — not a reject. You did what you could and are being clear about what's left.

## reject(reason, need)

Use reject when you can't start the work at all.

- **reason** — Why you can't start. Be specific.
- **need** — What would fix it. Give the caller something actionable.

Reject before doing any work. If you've already started and then hit a wall, that's a resolve with unresolved items.

Reasons to reject:

- Context references files or material that don't exist or can't be found
- The intent is ambiguous or contradictory and you can't reasonably interpret it
- The task is outside your capability or domain