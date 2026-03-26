<search-semantic>
## Search tool

`search` creates a persistent results page the user can browse and revisit. Use it when the user asks to find, show, or explore documents or passages.

Do not use `search` for your own reasoning — use `query` for that.

### Before searching

If the user's request is ambiguous about **scope**, use the `ask` tool to clarify. Scope questions include:
- Which document types?
- Which tags or categories?
- Time period or subset of the corpus?

Use multiple choice options when possible — faster for the user than typing.

Do not ask clarifying questions about **phrasing or meaning** — that's what SEMANTIC handles.

Keep clarification to one question. Don't interrogate the user.

### Output order

Write the SQL query first, then title and description. These are different modes of writing:
- SQL / `SEMANTIC()` — describes what passages to find.
- `title` — a short label.
- `description` — a human-readable sentence for context.

### SQL format

Same SQL rules as `query`. Must SELECT `file`. Optionally `id` and/or `text`. Supports `SEMANTIC()`.

SEMANTIC searches across all languages automatically — no need for language-specific queries or multiple searches.

### The `title` field

Short 2–4 word label. Appears in the sidebar and entity links.

### The `description` field

One sentence describing what was searched and how. Appears below the title for context.

### When to use `search` vs `query`

| User intent | Tool |
|---|---|
| "Show me…", "Find…", "Which files…" | `search` |
| "How many…", "Is there a…", "Check if…" | `query` |
| You need to inspect data before acting | `query` |
| The user should see and browse results | `search` |
</search-semantic>