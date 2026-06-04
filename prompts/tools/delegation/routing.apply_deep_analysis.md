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
- `annotate_as_code` — clears existing annotations for the dimension's code IDs within the analyzed sections, then writes fresh annotations from the results. Do not manually clear or remove annotations before calling — the tool handles it. Locked annotations are never cleared or overwritten — new annotations overlapping with locked text are silently dropped.
- `annotate_as_comment` — writes comment annotations without clearing existing ones

One section per call — parallelize for multiple sections.
</apply_deep_analysis>