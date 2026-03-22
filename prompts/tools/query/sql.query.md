<query-sql>
## SQL reference for project database

### Column contract

Every query must SELECT a column named `file` (VARCHAR). Optionally include:
- `id` (VARCHAR): a specific entity within the document (annotation, callout, etc.)
- `text` (VARCHAR): a text passage to display as a snippet

Extra columns are returned but only `file`, `id`, and `text` drive the UI.

### The `files` table

The `files` table stores text passages from documents — one row per paragraph-sized chunk. Columns: `file` (source document), `text` (passage content). This is the only table that supports `SEMANTIC()`.

### Choosing a matching strategy

For concepts, topics, or meaning — use `SEMANTIC()` on the `files` table. It finds similar content regardless of exact wording.
For a specific known term or exact substring — use a single `ILIKE`. Max 3 ILIKE clauses per query.
For exact values (tags, codes, IDs) — use `=`, `list_has()`, or `IN`.

If you're tempted to write multiple ILIKEs to cover synonyms or variants, that's a `SEMANTIC()` query.

### `SEMANTIC()` pseudo-function

`SEMANTIC('text')` scores each row by similarity to the given text. Write it in the SELECT list — the system handles scoring, ranking, and limits automatically. Only works on the `files` table.

```sql
SELECT f.file, f.text, SEMANTIC('political frustration')
FROM files f
```

Combine with filters:
```sql
SELECT f.file, f.text, SEMANTIC('economic policy')
FROM files f
WHERE f.file IN (SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'interview'))
```

Rules:
- One `SEMANTIC()` per query
- Only on the `files` table
- No operators or `AS` after `SEMANTIC()` — just write `SEMANTIC('text')`
- No `SEMANTIC()` in `ORDER BY` — ranking is automatic

### Query examples

Tag filter (file-only):
```sql
SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'interview')
```

Semantic search (file + text):
```sql
SELECT f.file, f.text, SEMANTIC('working from home policies')
FROM files f
```

Keyword match (file + id + text):
```sql
SELECT a.file, a.id, a.text FROM attributes_annotations a WHERE a.text ILIKE '%frustration%'
```

Entity by code (file + id):
```sql
SELECT a.file, a.id FROM attributes_annotations a WHERE a.code = 'callout-abc123'
```

### Truncation

Results are capped at 50 rows. Strings longer than 200 characters are truncated with `...`. Design queries with appropriate `LIMIT` clauses.

### When to use `query`

Use `query` for your own reasoning: counting, aggregating, checking values, exploring data before deciding what to show the user. Results go to you, not the user.
</query-sql>
