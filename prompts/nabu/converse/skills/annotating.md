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

"Find mentions of healthcare systems", "apply codebook", "highlight frustration", "tag policy discussions"

**Always plan.** You can't grep concepts—they're expressed in many ways. You're an LLM: you're good at reading and understanding, bad at guessing grep patterns.

```
create_plan:
  task: "Highlight mentions of healthcare systems"
  files: ["transcript.md"]
  steps:
    - title: "Define what we're looking for"
      expected: "Clear criteria: what counts as 'healthcare systems' in this context"
    - per_section:
        - title: "Read and identify matches"
          expected: "Passages matching the concept identified"
        - title: "Apply annotations"
          expected: "Annotations added for this section"
    - title: "Summary"
      expected: "Overview of what was found"
```

**The first step matters.** Before reading, define what the concept means. "Healthcare systems" could be:
- Direct references: "zorg", "zorgsysteem", "gezondheidszorg"
- Infrastructure: IC capacity, hospital beds, care facilities
- Organizations: VWS, RIVM, GGD
- Processes: care delivery, treatment protocols

Without this step, you'll either grep randomly or miss things while reading.

## Decision Guide

| Signal | Path |
|--------|------|
| Exact term in quotes: "omikron" | Literal → grep |
| Concept/topic: "healthcare systems", "frustration" | Semantic → plan with per_section |
| "mentions of [exact-word]" | Literal |
| "mentions of [concept]" | Semantic |
| Codebook reference | Semantic |

**Key question:** Do you know the exact string(s) to search? If yes → grep. If the concept could be expressed many ways → read.

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
