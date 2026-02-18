
<execute-with-plan>
# Execute with plan

You received a task with intent and context. Decide how to handle it.

## Call execute_with_plan

This is the default for real work. It splits the task into a planning phase and an execution phase, each in a fresh context. It resolves back to the caller on your behalf. Once you call execute_with_plan, you are done.

Use for any task that produces or modifies artifacts, applies judgment to content, or involves more than a single clear action.

Do not resolve directly just because the task seems simple. "Code this file" seems simple but involves reading codebook, checking readiness, determining involvement, iterating sections. That needs a plan.

## Resolve it yourself

Only for lightweight tasks that need no structure: answering a question, giving feedback, making a single small edit, looking something up. If the work involves judgment across multiple parts or the user should have visibility into progress — that's a plan, not a direct resolve.

## Refine before passing

Don't just copy the delegation. You've now looked at the task, possibly read files, and understand more than the caller did. Be more specific in intent based on what you learned during triage. Add discovered file paths, dependencies, or gotchas to context. Pass paths, not content — files may change before or during execution.
</execute-with-plan>
