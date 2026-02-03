# Annotation Tasks

<annotation-tasks>
## Two Paths

Annotation tasks split based on whether you're matching literal strings or interpreting meaning.

### Literal String Annotation

"Highlight mentions of X", "annotate where Y appears", "tag occurrences of Z"

1. **One grep** — `grep -n -B1 -A1 "term"` (all files, one call)
2. **Context sufficient?** Use `upsert_annotations` on matched files
3. **Need more?** Widen context (`-B3 -A3`) or read specific sections

Don't grep file-by-file. Don't plan for mechanical find-and-annotate.

### Semantic Annotation

For concepts that can't be grepped ("healthcare systems", "frustration", "policy discussions"), use `per_section` processing. See **skills/coding.md** for the apply codebook workflow.

## Decision Guide

| Signal | Path |
|--------|------|
| Exact term in quotes: "omikron" | Literal → grep |
| Concept/topic: "healthcare systems", "frustration" | per_section |
| "mentions of [exact-word]" | Literal |
| "mentions of [concept]" | per_section |
| Codebook reference | See coding.md |

**Key question:** Do you know the exact string(s) to search? If yes → grep. If the concept could be expressed many ways → use `per_section`.

## Literal Flow

```
grep -n -B1 -A1 "omnikron"
```

Returns:
```
notes.md:12-  The team discussed
notes.md:13:  omnikron integration plans
notes.md:14-  for Q2 rollout
interview.md:45-  mentioned that
interview.md:46:  omnikron was difficult
interview.md:47-  to configure
```

Context sufficient → use `upsert_annotations`:

```
upsert_annotations:
  path: "notes.md"
  annotations:
    - text: "omnikron integration plans"
      reason: "Mentions omnikron"
    - text: "omnikron was difficult to configure"
      reason: "Mentions omnikron"
```

No plan needed. One tool call per file, or batch if same file.

Use `remove_annotations` to delete. Never use `apply_local_patch` for annotations.
</annotation-tasks>
