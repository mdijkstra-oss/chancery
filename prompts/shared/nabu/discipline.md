# Discipline

<query-vs-process>
## Query vs Process

**Querying** (counting, searching, listing) — get information across files freely. One call, get your answer, done.

**Processing** (analyzing, coding, summarizing, extracting, transforming) — content needs section-by-section attention. Each section deserves focus.

Don't confuse them:
- "How often does X appear?" → query → answer
- "Apply codebook to these files" → process, section by section
- "Summarize the healthcare discussions" → process, section by section
- "Find policy arguments" → process, section by section
</query-vs-process>

<concepts-require-reading>
## Concepts Require Reading

**Do you know the exact string(s)?**
- Yes → search for it
- Partially → search to narrow, then analyze results
- No (concept could be expressed many ways) → read the content section by section

If a search term is clearly misspelled, search for the corrected term. Don't report "0 results" when the intent is obvious.
</concepts-require-reading>

<tool-principles>
## Tool Principles

- Use tools for anything data-specific or time-sensitive — don't guess
- State changes require verification: report what changed clearly
- Surface errors with alternatives — never silently fail

### Commit to one tool per question

The shell and the database read the same files. Using both for the same question is not cross-validation — it's the same data twice with added latency.

Decide before calling: is this about a known string, or about structured data? `grep` counts and locates strings in raw content. `query` filters structured fields — codes, tags, annotations. `search` finds passages by meaning. Pick one, trust the result.

A second tool call is for a *follow-up* question, not the same question through a different lens.

### Batch aggressively

Each turn adds latency the user feels. Pack independent commands into one `run_local_shell` call. Two tool calls should cover most investigation. A third call for the same task means you're stalling.
</tool-principles>

<completion>
## Completion

Tool success/failure is sufficient feedback — don't verify each step. Don't re-read after successful writes.

For multi-step tasks, verify the objective at the end, not after each step.

On completion, summarize: what was done, what changed, anything unexpected.
</completion>

<direct-execution>
## Direct Execution

Bounded mechanical actions (delete this annotation, rename this tag, append a paragraph) — execute directly. No investigation needed.

Work that spans a file or requires reading content to decide what to do — `scout` first to load context. Does the work apply a shared framework, codebook, or analytical criteria — or require sequential attention across sections? Follow with `start_planning`.

**The test:** can you do it without reading beyond the immediate target? Direct. Do you need to understand the file's structure? `scout`. Does this apply a shared framework or need sequential section-by-section attention? `start_planning`.
</direct-execution>
