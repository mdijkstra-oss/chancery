<query-sql>
## SQL reference for project database

### Column contract

Every query must SELECT `file` (VARCHAR). Optionally include:
- `id` (VARCHAR): a specific entity within the document
- `text` (VARCHAR): a text passage to display as a snippet

Extra columns are returned but only `file`, `id`, and `text` drive the UI.

### The `files` table

One row per paragraph-sized chunk. Columns: `file` (source document), `text` (passage content). Only table that supports `SEMANTIC()`.

### Choosing a matching strategy

For concepts, topics, or meaning → `SEMANTIC()` on the `files` table.
For a specific known term or exact substring → `ILIKE`. Up to 3 per query.
For exact values (tags, codes, IDs) → `=`, `list_has()`, or `IN`.

If you're tempted to write multiple ILIKEs to cover synonyms, that's a `SEMANTIC()` query.

Do not use `SEMANTIC()` when a simpler strategy works:
- User names a specific term or phrase → `ILIKE '%asbestos%'`, not `SEMANTIC('asbestos')`
- Filtering by tag, code, or metadata → `=`, `list_has()`, `IN`
- Counting or checking existence → `query` with exact match

Reserve `SEMANTIC()` for when the meaning matters more than the wording — when the corpus may express the idea in different words than the request uses.

### `SEMANTIC()` function

`SEMANTIC()` takes a single description of what to find. Describe the passages you want — the system handles search strategy, scoring, ranking, and limits.

```sql
SELECT f.file, f.text, SEMANTIC('passages where engineers flag structural safety concerns')
FROM files f
WHERE f.file IN (SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'report'))
```

Rules:
- One `SEMANTIC()` per query, only on `files`
- No `AS`, no `ORDER BY` — ranking is automatic

### Writing SEMANTIC descriptions

Describe the passages you want to find as if briefing a research assistant. Be specific about what the passages say, not just the topic.

Good: `SEMANTIC('passages where defendants argue the court lacks jurisdiction')`
Specific about what the text says.

Weak: `SEMANTIC('jurisdiction')`
Too vague — matches everything tangentially related.

Include stance or direction when relevant:
- `SEMANTIC('passages praising the new policy')` — not just `'new policy'`
- `SEMANTIC('complaints about slow delivery')` — not just `'delivery'`

Decompose user requests:
- Document types, tags, metadata → WHERE clause
- What the passages say → SEMANTIC()

### SEMANTIC vs description

SEMANTIC describes what to search for. The `description` field in `search` calls is a label for the user. Don't mix them.

User: "Find court filings where defendants challenged jurisdiction"
→ `SEMANTIC('passages where defendants argue the court lacks jurisdiction or is the wrong venue')`
→ title: "Jurisdiction challenges"
→ description: "Court filings where defendants challenged the jurisdiction or venue of the court"

### Query examples

```sql
-- Tag filter
SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'memo')

-- Semantic search with filter
SELECT f.file, f.text, SEMANTIC('passages describing soil or groundwater contamination from industrial waste')
FROM files f
WHERE f.file IN (SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'environmental'))

-- Keyword match
SELECT a.file, a.id, a.text FROM attributes_annotations a WHERE a.text ILIKE '%asbestos%'

-- Entity by code
SELECT a.file, a.id FROM attributes_annotations a WHERE a.code = 'callout-abc123'
```

### Limits

Max 50 rows.

### `query` vs `search`

`query` returns results to you for reasoning. The user does not see them.
`search` creates a persistent results page the user can browse.
</query-sql>