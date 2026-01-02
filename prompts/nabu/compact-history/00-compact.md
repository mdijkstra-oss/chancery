# Compact Prompt

<task>
Produce a structured summary that allows work to continue without the full transcript.

## Output format

```
## Decisions
[User decisions, interpretations chosen, methodology confirmed]

## State
[What's done, in progress, pending. Include relevant IDs]

## Preferences
[Workflow, tone, methodology preferences expressed]

## Context
[Definitions agreed, patterns observed, domain knowledge established]

## Open
[Unresolved questions, blockers, flagged issues]
```

Omit empty sections.
</task>

<preserve>
- User decisions (authoritative, cannot be inferred later)
- Work state: completions, partial progress, what's queued
- Relevant IDs with brief reference (e.g., "doc_abc (Jones interview)"); use table if more than 5
- Established definitions and criteria
- Preferences that affect future behavior
- Unresolved issues, blockers, questions awaiting answer
- Errors that inform future work (e.g., "X approach failed because Y")
</preserve>

<discard>
- Conversational back-and-forth that led to decisions
- Tool call mechanics (preserve outcomes only: "created code X" not the tool JSON)
- Routine confirmations and acknowledgments
- Explanations already understood
- Failed attempts unless failure is informative
- Narration of process ("I'll now...", "Let me...")
- Restating what's visible in current context
</discard>

<principle>
Someone reading only this summary should understand what happened and what matters going forward, without needing the original transcript. Be concise but complete.
</principle>