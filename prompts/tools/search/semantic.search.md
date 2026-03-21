
<search-semantic>
## Search tool: when and how

### When to use `search` vs `run_local_shell`

`search` creates a persistent, browsable results page the user can revisit. Use it for display-style queries: "show me all documents tagged X", "find annotations mentioning Y", "which files have codes about Z".

`run_local_shell` is for ephemeral answers: counting, aggregation, checking a single value. "How many files are tagged interview?" → shell. "Show me all files tagged interview" → search.

### Query contract

Queries run against the database tables injected at the start of the conversation. Each query declares its `type`:

- `"file"`: the SQL must SELECT a column named `file` (VARCHAR). Each row represents a matching document.
- `"hit"`: the SQL must SELECT columns named `file` (VARCHAR) and `id` (VARCHAR). Each row represents a specific entity within a document.

Extra columns are ignored. Results are deduplicated automatically.

### Examples

Find all documents tagged "interview":
```json
{
  "type": "file",
  "sql": "SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'interview')"
}
```

Find annotations mentioning a term:
```json
{
  "type": "hit",
  "sql": "SELECT a.file, a.id FROM attributes_annotations a WHERE a.text ILIKE '%frustration%'"
}
```

Find files containing a codebook code:
```json
{
  "type": "hit",
  "sql": "SELECT a.file, a.id FROM attributes_annotations a WHERE a.code = 'callout-abc123'"
}
```

### The `title` field

A short 2-4 word label for the search: "Interview documents", "Frustration mentions", "Policy reviews". This appears in the sidebar and entity links.

### The `description` field

A longer human-readable summary: "All documents tagged interview from the 2024 corpus", "Annotations mentioning frustration or dissatisfaction". This appears below the title for context.
</search-semantic>
