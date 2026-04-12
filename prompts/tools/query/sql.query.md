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

### Array columns

Columns like `tags` are `VARCHAR[]`. Use list functions to filter and inspect them.

| Function | Clause | Purpose | Example |
|----------|--------|---------|---------|
| `list_has(col, val)` | WHERE | Row contains value | `WHERE list_has(attributes.tags, 'memo')` |
| `list_has_any(col, [v1, v2])` | WHERE | Row contains any of | `WHERE list_has_any(attributes.tags, ['memo', 'report'])` |
| `list_has_all(col, [v1, v2])` | WHERE | Row contains all of | `WHERE list_has_all(attributes.tags, ['memo', 'report'])` |
| `len(col)` | WHERE / SELECT | Array length | `WHERE len(attributes.tags) > 0` |
| `unnest(col)` | SELECT / FROM | Expand to rows | `SELECT unnest(attributes.tags) AS tag FROM attributes` |

`unnest()` expands arrays into rows. It belongs in SELECT or FROM, never in WHERE.

### `SEMANTIC()` function

`SEMANTIC()` takes a single description of what to find. Describe the passages you want — the system handles search strategy, scoring, ranking, and limits.

SEMANTIC automatically searches across all languages in the corpus. Write your description in the user's language — it finds matching passages regardless of what language they were written in.

```sql
SELECT files.file, files.text, SEMANTIC('passages where engineers flag structural safety concerns')
FROM files
WHERE files.file IN (SELECT DISTINCT attributes.file FROM attributes WHERE list_has(attributes.tags, 'report'))
```

Rules:
- One `SEMANTIC()` per query, only on `files`
- No `AS`, no `ORDER BY` — ranking is automatic

### Writing SEMANTIC descriptions

Describe the passages you want to find as if briefing a research assistant. Be specific about what the passages say, not just the topic.

Phrase around explicit textual evidence, not interpretive impressions. Prefer "passages where the speaker explicitly says X" over "passages where the speaker seems to X." Vague framing like "appears to" or "suggests" drifts into adjacent behaviors and inflates noise.

Good: `SEMANTIC('passages where defendants argue the court lacks jurisdiction')`
Specific about what the text says.

Weak: `SEMANTIC('jurisdiction')`
Too vague — matches everything tangentially related.

Weak: `SEMANTIC('passages where the speaker seems unable to justify their position')`
"Seems" invites inference — pulls in adjacent behaviors like hedging, deflection, topic avoidance.

Better: `SEMANTIC('passages where the speaker explicitly says they cannot justify, have no justification, or acknowledge lacking a rationale')`
Describes observable language.

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
SELECT DISTINCT attributes.file FROM attributes WHERE list_has(attributes.tags, 'memo')

-- Semantic search with filter
SELECT files.file, files.text, SEMANTIC('passages describing soil or groundwater contamination from industrial waste')
FROM files
WHERE files.file IN (SELECT DISTINCT attributes.file FROM attributes WHERE list_has(attributes.tags, 'environmental'))

-- Keyword match
SELECT attributes_annotations.file, attributes_annotations.id, attributes_annotations.text FROM attributes_annotations WHERE attributes_annotations.text ILIKE '%asbestos%'

-- Entity by code
SELECT attributes_annotations.file, attributes_annotations.id FROM attributes_annotations WHERE attributes_annotations.code = 'callout-abc123'

-- Codebook codes
SELECT callouts.file, callouts.id, callouts.title FROM callouts WHERE callouts.type = 'codebook-code'

-- All distinct tags
SELECT DISTINCT unnest(attributes.tags) AS tag, attributes.file FROM attributes

-- Annotations on a file
SELECT attributes.annotations, attributes.file FROM attributes WHERE attributes.file = 'doc.md'
```

### Page size and pagination

Max page size is 50 rows. No LIMIT means the first 50 rows are returned. LIMIT above 50 is shrunk to 50. Paginate past the first page with `LIMIT 50 OFFSET 50`, then `OFFSET 100`, and so on.

For aggregate queries (`COUNT`, `GROUP BY`) just write the aggregate — no LIMIT needed, and the result is not treated as a page.

`SEMANTIC()` queries are ranked by relevance automatically, so pagination is meaningless — OFFSET is dropped. Showing more results means refining the `SEMANTIC('...')` phrase or tightening the WHERE filter, not paging deeper.

### `query` vs `search`

`query` returns raw results to you for reasoning. The user does not see them. For SEMANTIC queries, results are ranked by embedding similarity but not filtered — you judge relevance yourself. Expect noise among the top results; pick what matters and discard the rest.

`search` creates a persistent results page the user can browse.
</query-sql>