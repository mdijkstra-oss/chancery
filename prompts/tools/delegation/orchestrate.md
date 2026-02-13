---
requires:
  - orchestrate
---

# Orchestrate

You received a task via delegate with intent, outcome, context, involvement, and constraints. Decide how to handle it.

## Resolve it yourself

This is the default. Do the work and resolve. Searching, reading, analysing, writing files, making edits, running commands — all normal work that does not require orchestration, even if it takes many tool calls.

## Call orchestrate

Orchestrate splits the task into a planning phase and an execution phase, each in a fresh context. It resolves back to the caller on your behalf. Once you call orchestrate, you are done.

Call orchestrate when any of these apply:

- The task requires changes to more than 3 files that reference or depend on each other
- You'd need to read more than 3 files or documents before you can start the actual work, and the work itself is also substantial
- The intent describes a workflow with explicit phases — analyse then transform then validate, or similar
- The user described a multi-step process with decision points between steps
- The task involves a loop over items where each iteration requires judgment and may surface ambiguity (e.g. coding a transcript section by section, reviewing documents one by one with user check-ins)

Do not call orchestrate when:

- The steps are independent — just do them in sequence
- The task is repetitive without judgment — doing the same operation across many files doesn't need a plan, it needs a loop
- There are fewer than 3 distinct steps

## orchestrate(intent, outcome, context, involvement, constraints)

The fields are the same ones you received via delegate. Don't just copy them — you've now looked at the task, possibly read files, and understand more than the caller did. Refine them:

- **intent** — Be more specific based on what you learned during triage. If you discovered the real shape of the work is different from what was described, say so.
- **outcome** — Sharpen it. "Auth module refactored with all tests passing" is better than "auth module improved."
- **context** — Add what you discovered. File paths you found, dependencies you noticed, patterns in the codebase, gotchas to watch for.
- **involvement** — Pass through unless you have reason to change it. If the caller said "fully autonomous" but you discovered the task involves destructive or ambiguous changes, consider raising it.
- **constraints** — Pass through and add any you discovered. Conventions in the codebase, patterns to follow, things to avoid that aren't obvious from the intent alone.