# Codebook

<codebook-structure>
## Document Structure

A codebook is a markdown document that defines codes for qualitative analysis. Use standard markdown for organization:

- **Headers** (`##`, `###`) group codes into themes, categories, or hierarchies
- **Prose** between codes explains relationships, provides context, or documents decisions
- **Individual codes** are `json-callout` blocks with `type: "codebook"`

Example structure:
```markdown
# Project Codebook

Introduction and scope...

## User Experience

Context about this theme...

```json-callout
{code definition}
```

```json-callout
{another code}
```

## Technical Issues

...
```
</codebook-structure>

<code-definitions>
## Writing Code Definitions

Each code is a callout block with type `codebook`:

```json-callout
{
  "id": "[uuid-callout]",
  "type": "codebook",
  "title": "Code Name",
  "color": "blue",
  "collapsed": false,
  "content": "Definition and application guidance..."
}
```

### Title
- Short, descriptive label (2-5 words)
- Noun phrases: "User Frustration", "Technical Barrier", "Positive Feedback"
- Consistent naming style across the codebook

### Content
The definition answers: *When does this code apply?*

Include:
- **Definition**: What this code captures (1-2 sentences)
- **Indicators**: Specific signals that suggest this code applies
- **Boundaries**: What this code does NOT include (distinguishes from similar codes)

### Quality Principles

**Clarity over completeness** — A focused definition beats a comprehensive one. Matches should be recognizable instantly.

**Mutual exclusivity** — Overlapping codes create inconsistency. Each code should have clear boundaries.

**Observable over interpretive** — "Mentions waiting time" beats "Feels impatient". Ground codes in what's said, not inferred states.

**Consistent granularity** — Codes at similar abstraction levels. Don't mix "Mentioned product" with "Expressed deep existential frustration".
</code-definitions>

<codebook-colors>
## Color Conventions

Colors group related codes visually. Suggested conventions:

- **blue/indigo/violet** — Core analytical codes
- **green/teal/jade** — Positive valence (satisfaction, success, praise)
- **red/tomato/crimson** — Negative valence (frustration, failure, complaints)
- **yellow/amber/orange** — Neutral observations, process codes
- **purple/plum/pink** — Interpretive or emergent codes
- **gray/slate/mauve** — Administrative, meta-codes

Consistency matters more than the specific scheme.
</codebook-colors>
