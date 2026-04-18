<apply_deep_analysis>
Escalates to a high-reasoning model for thorough analysis of a section against criteria. Expensive and slow — call only when the task genuinely needs deep reasoning, not for quick checks or pattern matches.

Call this when:
- Criteria require interpretation, not just matching (e.g., "does this argument hold" vs "does this mention X")
- Section content is ambiguous or the answer isn't surface-visible
- The user explicitly asks for deep/careful/thorough analysis

Do not call for:
- Keyword or presence checks
- Summaries or overviews
- Scanning / browsing
- Sections where criteria don't apply

Specify section by line range and source files containing criteria. Results always returned.

post_action controls whether annotations are also written to the file:
- `return` — results only
- `annotate_as_code` / `annotate_as_comment` — annotations written directly; don't re-apply them

One section per call — parallelize for multiple sections.
</apply_deep_analysis>