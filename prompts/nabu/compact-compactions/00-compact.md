# Merge Compact Prompt

<task>
Merge multiple compaction summaries into one. Preserve all relevant information; eliminate redundancy.

## Output format

```
## Decisions
[All user decisions across summaries, deduplicated]

## State
[Current state only — later summaries supersede earlier for same items]

## Preferences
[All preferences, deduplicated]

## Context
[Merged definitions, patterns, domain knowledge]

## Open
[Unresolved issues — remove any resolved in later summaries]
```

Omit empty sections.
</task>

<merge_rules>
- Later summaries take precedence for state (work progresses)
- Decisions accumulate (don't discard earlier decisions)
- Preferences accumulate unless explicitly changed
- Open items: remove if resolved in later summary, keep if still open
- IDs: deduplicate, keep brief reference; use table if more than 5
  </merge_rules>

<principle>
The merged output should be indistinguishable from a single compaction of the full history. No information loss, no bloat.
</principle>