# Annotation Tasks

<annotation-tasks>
## Two Paths

Annotation tasks split based on whether you're matching literal strings or interpreting meaning.

### Literal String Annotation

"Highlight mentions of X", "annotate where Y appears", "tag occurrences of Z"

1. **One grep** — `grep -n -B1 -A1 "term"` (all files, one call)
2. **Context sufficient?** Direct execute: patch annotations into matched files
3. **Need more?** Widen context (`-B3 -A3`) or read specific sections

Don't grep file-by-file. Don't plan for mechanical find-and-annotate.

### Semantic Annotation

"Apply codebook", "find themes", "identify frustration", "code for user pain points"

**Always plan with `files` + `per_section`.**

Why: Large files lose information in the context window middle (lost-in-middle problem). Section-by-section processing keeps each chunk in focus. Grepping synonyms misses things—you need to read and interpret.

## Decision Guide

| Signal | Path |
|--------|------|
| Exact term, quoted string | Literal → grep + direct execute |
| Concept, emotion, theme | Semantic → plan with per_section |
| "mentions of X" | Literal |
| "examples of X" | Semantic (requires judgment) |
| Codebook reference | Semantic |

When uncertain: if you could write a regex for it, it's literal. If you need to understand meaning, it's semantic.

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

Context sufficient → patch annotations directly:
- `text`: the matching line or relevant phrase
- `reason`: derived from surrounding context
- Target files: those with matches

No plan needed. Batch all patches in one response.

## Semantic Flow

```
create_plan:
  task: "Apply codebook to interview transcripts"
  files: ["interview-1.md", "interview-2.md"]
  steps:
    - per_section:
        - title: "Identify codeable passages"
          expected: "Relevant passages identified with applicable codes"
        - title: "Apply annotations"
          expected: "Annotations added with code references and reasons"
    - title: "Review coverage"
      expected: "Summary of codes applied across documents"
```

The `per_section` steps receive content chunk by chunk—each section gets full attention, avoiding context window degradation.

## Common Mistakes

**File-by-file grep** — `grep x file1`, `grep x file2`, ... (N tool calls)
→ Fix: `grep x` with no path searches all files (1 tool call)

**Plan without files** — Finding matches via grep, then planning without passing `files`
→ Fix: If you need `per_section` processing, pass `files` even if you "already found" them

**Grep for concepts** — `grep frustrated`, `grep angry`, `grep upset`, ...
→ Fix: You can't grep emotions. Plan with `per_section` and interpret.

**Over-planning literals** — Full explore/plan cycle for "highlight term X"
→ Fix: One grep with context, then direct execute patches
</annotation-tasks>
