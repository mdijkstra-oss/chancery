<query-sql>
## SQL reference for project database

### Column contract

Every query must SELECT a column named `file` (VARCHAR). Optionally include:
- `id` (VARCHAR): a specific entity within the document (annotation, callout, etc.)
- `text` (VARCHAR): a text passage to display as a snippet

Extra columns are returned but only `file`, `id`, and `text` drive the UI.

### The `files` table

One row per paragraph-sized chunk. Columns: `file` (source document), `text` (passage content). This is the only table that supports `SEMANTIC()`.

### Choosing a matching strategy

For concepts, topics, or meaning → `SEMANTIC()` on the `files` table.
For a specific known term or exact substring → a single `ILIKE`. Max 3 per query.
For exact values (tags, codes, IDs) → `=`, `list_has()`, or `IN`.

If you're tempted to write multiple ILIKEs to cover synonyms or variants, that's a `SEMANTIC()` query.

### `SEMANTIC()` pseudo-function

`SEMANTIC('text')` scores each row by similarity to the given text. Write it in the SELECT list — the system handles scoring, ranking, and limits automatically.

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
- No operators or `AS` after `SEMANTIC()`
- No `SEMANTIC()` in `ORDER BY` — ranking is automatic

### Writing SEMANTIC queries

Keep SEMANTIC queries short and precise — ideally under 10 words. The embedding model averages the entire string into one vector. Every extra word dilutes the signal.

**Decompose the user's request** into filters and a semantic core:
- Document types, tags, metadata → WHERE clause
- General context or setting → WHERE clause or omit entirely
- The specific concept to find → SEMANTIC()

SEMANTIC should contain only the differentiating idea — the thing that separates matching passages from non-matching ones.

**Strip away:**
- Terms that describe the corpus topic (if the corpus is about remote work, don't put "remote work" in SEMANTIC)
- Terms that restate document type or structure ("interview files", "field reports")
- Synonyms and elaborations you invented — use the user's core phrasing, not your expansion of it

**Do not** rephrase, pad, or add synonyms. The embedding model handles similarity — your job is to give it a clean target, not a thesaurus entry.

**Examples:**

User: "Interview files where people describe feeling always available while working from home"
- "Interview files" → `WHERE list_has(a.tags, 'interview')`
- "working from home" → corpus context, omit
- Core concept → `SEMANTIC('feeling always available, unable to switch off')`

Bad: `SEMANTIC('feeling always on while working from home, always available, unable to switch off, remote work blurs boundaries, pressure to stay online')`
Why bad: "working from home", "remote work", "blurs boundaries", "pressure to stay online" are context and invented synonyms that dilute the core concept.

User: "Field reports documenting habitat loss near river systems"
- "Field reports" → tag/type filter
- Core concept → `SEMANTIC('habitat loss near rivers')`

Bad: `SEMANTIC('habitat loss near river systems, deforestation, biodiversity decline, ecological damage, environmental degradation')`
Why bad: everything after the first phrase is synonyms you added.

User: "Show me passages about productivity"
- No filter needed
- Core concept → `SEMANTIC('productivity')`

Bad: `SEMANTIC('being more productive, getting more done, increased output, higher efficiency, better performance at work')`
Why bad: one word was enough. Five restatements make it worse.

### Query examples

Tag filter (file-only):
```sql
SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'interview')
```

Semantic search (file + text):
```sql
SELECT f.file, f.text, SEMANTIC('economic anxiety')
FROM files f
```

Semantic search with filter:
```sql
SELECT f.file, f.text, SEMANTIC('feeling isolated')
FROM files f
WHERE f.file IN (SELECT DISTINCT a.file FROM attributes a WHERE list_has(a.tags, 'interview'))
```

Keyword match (file + id + text):
```sql
SELECT a.file, a.id, a.text FROM attributes_annotations a WHERE a.text ILIKE '%frustration%'
```

Entity by code (file + id):
```sql
SELECT a.file, a.id FROM attributes_annotations a WHERE a.code = 'callout-abc123'
```

### Limits

Max 50 rows.

### `query` vs `search`

`query` returns results to you for reasoning: counting, aggregating, checking values, exploring data before deciding what to do. The user does not see query results.

`search` creates a persistent results page the user can browse. Use it when the user asks to find, show, or explore. If the user should see the results, use `search`.
</query-sql>