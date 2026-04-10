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

Write the SQL query first, then highlight, then title and description. These are different modes of writing:
- SQL / `SEMANTIC()` — describes what passages to find.
- `highlight` — the filtering instruction sent to a post-retrieval model that decides which sentences within each result chunk to extract and display. Describe what to *show*, not what to *search for* (that's SEMANTIC's job). The more specific the highlight, the tighter the extracted passages — vague highlights return broad chunks, specific highlights return the exact relevant sentences.
- `title` — a short label.
- `description` — a human-readable sentence for context.

### SQL format

Same SQL rules as `query`. Must SELECT `file`. Optionally `id` and/or `text`. Supports `SEMANTIC()`.

SEMANTIC searches across all languages automatically — no need for language-specific queries or multiple searches.

Do not write `LIMIT` or `OFFSET` in search SQL — the results page owns pagination and loads more as the user scrolls. Any paging clauses are stripped before execution.

### The `title` field

Short 2–4 word label. Appears in the sidebar and entity links.

### The `description` field

One sentence describing what was searched and how. Appears below the title for context.

### Result samples

Results returned to you are embedding-ranked, not pre-filtered. Judge relevance yourself.

### When to use `search` vs `query`

| User intent | Tool |
|---|---|
| "Show me…", "Find…", "Which files…" | `search` |
| "How many…", "Is there a…", "Check if…" | `query` |
| You need to inspect data before acting | `query` |
| The user should see and browse results | `search` |

`query` returns results for your reasoning only. `search` saves a persistent results page the user can browse.
</search-semantic>