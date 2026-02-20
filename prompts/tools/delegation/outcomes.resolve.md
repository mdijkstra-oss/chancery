
<resolve-cancel>
# resolve and cancel

`resolve` = the plan is complete. Call from execution mode when all steps are done. Submits the outcome.

`cancel` = abandon the current mode and return to chat. Use when:

- Context references files or material that don't exist or can't be found
- The intent is ambiguous or contradictory and you can't reasonably interpret it
- The task is outside your capability or domain
- Critical files are missing or the plan is fundamentally blocked
- The user wants to stop the current plan or execution

`cancel` is not failure — it's a mode transition back to conversation.
</resolve-cancel>
