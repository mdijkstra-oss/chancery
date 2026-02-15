---
requires:
  - execute_with_plan
---

<execute-with-plan>
# Execute with plan

You received a task with intent and context. Decide how to handle it.

## Call execute_with_plan

This is the default for real work. It splits the task into a planning phase and an execution phase, each in a fresh context. It resolves back to the caller on your behalf. Once you call execute_with_plan, you are done.

The planning phase ensures the right questions get asked, involvement is established, and the user sees structured progress rather than opaque activity.

Use execute_with_plan for any task that produces or modifies artifacts, applies judgment to content, or involves more than a single clear action. Examples: coding a transcript, revising a codebook, analysing a document, restructuring files.

## Resolve it yourself

Only for lightweight tasks that need no structure: answering a question, giving feedback on something the user showed you, making a single small edit, looking something up. If the work involves judgment across multiple parts or the user should have visibility into progress — that's a plan, not a direct resolve.

Do not resolve directly just because the task seems simple. "Code this file" seems simple but involves reading codebook, checking readiness, determining involvement, iterating sections. That needs a plan.

## execute_with_plan(intent, context)

Pass along what you know. Don't just copy the delegation — you've now looked at the task, possibly read files, and understand more than the caller did. Refine:

- **intent** — Be more specific based on what you learned during triage. If you discovered the real shape of the work is different from what was described, say so.
- **context** — Add what you discovered: file paths, dependencies, gotchas. Pass paths, not content — files may change before or during execution. The planner and executor read files themselves to get current state.

The planner will ask the user about involvement, constraints, and anything else it needs. You don't need to guess those.
</execute-with-plan>
