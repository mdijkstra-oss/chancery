---
requires:
  - delegate
---

# Delegation of work

You can delegate tasks to expert agents when a task requires specialised knowledge or can be handled more effectively by a narrower expert. This doesn't replace doing work yourself — delegate when it makes sense, handle things directly when it doesn't.

## Available action

### delegate(intent, outcome, context, involvement, constraints)

Send a task to an expert agent. The expert decides how to approach it — whether that's doing it directly or planning first.

- **who** — Who to delegate to.
- **intent** — What needs to be accomplished. Include enough detail that the expert can start working without needing to re-ask the user. If the user explained their goal across multiple messages, capture the full picture here — don't compress it into a vague summary.
- **outcome** — What done looks like. Be specific enough that both you and the expert agree when it's met.
- **context** — What the expert needs to know or read before starting. File paths, background, domain info, references. The expert decides whether and how to load any referenced material.
- **involvement** — How autonomous the expert is. When should it pause for the user? Examples: "fully autonomous", "check in if anything is ambiguous", "show me each step before proceeding".
- **constraints** — Rules the expert must follow. What to do, what not to do, and any quality standards or conventions to respect.

## When to delegate

- The task requires domain expertise you don't have
- The task involves domain-specific judgment (qualitative coding, legal review, etc.)
- A subtask is distinct enough that a narrower expert would handle it better

## When to handle it yourself

- Simple file operations — creating, renaming, moving, deleting files
- Straightforward reading, formatting, or restructuring content
- Answering questions from what you already know or can look up
- The user is asking a clarifying question, not requesting work
- Anything that doesn't require specialised knowledge

## When to do neither

- The request is ambiguous — gather what you need first, then decide

## Filling in the fields

Your primary job before delegating is gathering enough information to fill the fields well. Don't delegate the moment you recognise the task — make sure the expert can succeed without re-asking the user.

Ask the user when:

- **intent** is vague — "summarize these files" — summarize how? For who? What level of detail?
- **outcome** is unclear — "code this transcript" — what does a well-coded transcript look like to you?
- **context** is missing — are there reference files, conventions, prior work the expert should know about?
- **involvement** is worth asking about when the expert will be making judgment calls — summarizing, analysing, reviewing, coding. For mechanical tasks with one right answer (rename files, convert formats), skip it and let the expert run. For destructive or irreversible tasks (rewriting, restructuring, deleting), default to high involvement even if the user didn't specify.
- **constraints** are likely but unstated — are there things to avoid, formats to follow, conventions to respect?

You don't need to ask about every field every time. Use judgment:

- If the task is unambiguous, delegate immediately.
- If there's one unclear thing, ask about that one thing.
- If the request is broad or subjective (summarize, analyse, review), ask enough to give the expert a direction.

Constraints and involvement can be empty — that's fine. But intent and outcome should always be clear enough that you'd feel confident handing this to a colleague and walking away.

The expert has no access to your conversation with the user. Everything that matters must be in the five fields. If you summarise too aggressively, the expert either guesses wrong or has to ask the user questions they already answered.

## Handling responses

### resolve(outcome, unresolved, artifacts)

The expert is done. Read the response:

- **outcome** — What was accomplished. Check this against the original outcome you specified.
- **unresolved** — What couldn't be completed and why. If empty, the task is fully done. If not, decide: retry, delegate to a different expert, or ask the user.
- **artifacts** — Files created or modified. Present these to the user or pass them to the next task.

If the outcome matches what you asked for and unresolved is empty, the task succeeded. Tell the user.

If there are unresolved items, assess: is the result good enough to present, or does more work need to happen? You can delegate again with a narrower intent focused on the unresolved items.

### reject(reason, need)

The expert can't even start. Read the response:

- **reason** — Why the task can't be done.
- **need** — What would fix it.

Decide: can you provide what's needed? Can a different expert handle it? Or do you need to ask the user?

Do not silently retry the same delegate after a reject. Either fix what the expert said was missing, or tell the user what's needed.

## General rules

- One task per delegate. Don't bundle unrelated work.
- Don't delegate back and forth — if an expert resolves with unresolved items, decide what to do rather than bouncing it back immediately.
- Keep the user informed. When you delegate, briefly say what you're doing. When you get a result, present it clearly.