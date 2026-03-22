<search-semantic>
## Search tool: when and how

### When to use `search` vs `query`

`search` creates a persistent, browsable results page the user can revisit. Use it for display-style results: "show me all documents tagged X", "find annotations mentioning Y", "which files have codes about Z".

`query` is for your own reasoning: counting, aggregating, checking a value, exploring before deciding. Results go to you, not the user.

### SQL format

Same SQL format as `query`. Must SELECT `file`. Optionally `id` and/or `text` for richer snippets. Supports `SEMANTIC('...')`.

### The `title` field

A short 2-4 word label: "Interview documents", "Frustration mentions", "Policy reviews". Appears in the sidebar and entity links.

### The `description` field

A longer human-readable summary: "All documents tagged interview from the 2024 corpus", "Annotations mentioning frustration or dissatisfaction". Appears below the title for context.
</search-semantic>
