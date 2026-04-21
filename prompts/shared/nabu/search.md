# Lookup Discipline

## Tool hierarchy for content lookup

There are three tools that can read content: `run_local_shell`, `query`, and `search`. They are not interchangeable.

**Corpus documents** (transcripts, codebooks, notes, analysis files) → `query` or `search` only.  
**Non-corpus files** (settings structure, file names, metadata, line ranges) → `run_local_shell`.

Never use `run_local_shell` to search for content inside corpus documents. `grep` on a corpus file is always wrong — it finds literal strings in raw text and misses everything expressed differently.

---

## Choosing between query and search

| Question | Tool |
|---|---|
| User wants to find and browse results | `search` |
| You need data to reason with | `query` |
| Count, aggregate, or check existence | `query` |
| "Show me…", "Find…", "Which documents…" | `search` |
| "How many…", "Is there a…", "Check if…" | `query` |

`query` returns results to you only. `search` creates a persistent page the user can revisit.

---

## Choosing between SEMANTIC and ILIKE

**Default to `SEMANTIC()`** for any content lookup. Use `ILIKE` only when:
- The user named a verbatim term they would expect to appear literally in the text (a name, a date, a specific code label)
- You are filtering metadata fields (tags, codes, dates), not passage content

If you are writing more than one `ILIKE` to cover different phrasings of the same idea — stop. That is a `SEMANTIC()` query.

| Request type | Strategy |
|---|---|
| Concept, theme, behavior, emotion | `SEMANTIC()` |
| Specific named term the user stated verbatim | `ILIKE` |
| Multiple synonyms or phrasings | `SEMANTIC()` |
| Metadata filter (tag, code, date) | `=`, `list_has()`, `IN` |

---

## When grep is appropriate

`run_local_shell` with `grep` is appropriate for:
- Listing files or checking file existence (`ls --show-tags`, `find`)
- Reading sections of a document by line range (`cat -o -l`)
- Searching for literal strings in non-corpus files (settings structure, config)

`grep` on corpus content is never appropriate — not as a first pass, not as a cross-check, not as a fallback when SEMANTIC feels uncertain.

---

## Do not combine SEMANTIC with ILIKE content filters

`SEMANTIC()` already searches across all content by meaning. Adding `ILIKE` clauses to filter which rows `SEMANTIC` sees defeats its purpose — you are restricting the search to rows that contain specific literal strings, which is exactly what SEMANTIC is designed to avoid.

This is wrong:
```sql
SELECT files.file, files.text, SEMANTIC('passages where someone reacts with amusement')
FROM files
WHERE files.text ILIKE '%laugh%'
   OR files.text ILIKE '%chuckle%'
   OR files.text ILIKE '%joke%'
```

This is correct:
```sql
SELECT files.file, files.text, SEMANTIC('passages where someone reacts with amusement, laughs, or gives a humorous response')
FROM files
```

`WHERE` clauses alongside `SEMANTIC()` are only appropriate for structural filters: limiting to a file subset by tag, date, or document type. Never for content filtering.

## The failure pattern to avoid

User asks for a concept → model greps for one or two Dutch keywords → misses all paraphrases → reports false negatives.

If the request involves meaning, behavior, tone, or anything that could be expressed multiple ways: `search` with `SEMANTIC()`. Always.