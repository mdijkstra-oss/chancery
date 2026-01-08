# SQL Query Scenarios

<data-freshness>
## Your Knowledge is Stale

You do not know current document content. Users edit documents continuously — what you saw before may have changed. Query the database to get current state. Never assume content from prior context is still accurate.
</data-freshness>

<when-to-query>
## Examples when to Use execute_sql, not limited to:

**Metadata lookups:**
- List all documents in a project
- Find documents by name, title, or description
- Filter documents by time range or pinned status
- Count documents, blocks, or annotations

**Content search:**
- Find blocks containing specific text
- Locate headings with certain words
- Search across multiple documents at once

**Structural queries:**
- Find all blocks of a specific type (code, quote, table)
- Get document structure (headings hierarchy)
- Find child blocks under a parent

**Annotation analysis:**
- Find all annotations by actor
- Query by confidence level or color
- Find annotations linked to specific codes
</when-to-query>

<query-patterns>
## Common Patterns

**Search block content for text:**
```sql
SELECT b.id, d.name, b.content
FROM blocks b
JOIN documents d ON d.id = b.document_id
WHERE b.content::TEXT LIKE '%search term%'
```

**Get all headings in a document:**
```sql
SELECT id, content, props
FROM blocks
WHERE document_id = 'doc-xxx' AND type = 'heading'
ORDER BY position
```

**Find documents updated recently:**
```sql
SELECT id, name, updated_at
FROM documents
WHERE updated_at > NOW() - INTERVAL '7 days'
ORDER BY updated_at DESC
```

**Count blocks by type:**
```sql
SELECT type, COUNT(*) as count
FROM blocks
WHERE document_id = 'doc-xxx'
GROUP BY type
```

**Extract JSON fields:**
```sql
SELECT id, json_extract(props, '$.level') as heading_level
FROM blocks
WHERE type = 'heading'
```
</query-patterns>

<sql-discipline>
## Discipline

- Use JOINs to get context (document name with block content)
- LIMIT results when exploring — start with 10
- Cast JSON columns explicitly: `content::TEXT` or `json_extract()`
- ORDER BY position for block sequence, updated_at for recency
</sql-discipline>
