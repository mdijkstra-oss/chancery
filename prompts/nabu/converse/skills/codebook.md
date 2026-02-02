# Codebook

<codebook>
A codebook defines codes for qualitative analysis. This skill covers what codebooks are and how to create them.

## Trigger

User asks to:
- "Create a codebook"
- "Make codes for..."
- "Help me define codes"
- "Set up coding scheme"
- "Convert these codes to proper format"

## Structure

A codebook is a markdown document with:

- **Headers** (`##`, `###`) group codes into themes, categories, or hierarchies
- **Prose** between codes explains relationships, provides context, or documents decisions
- **Individual codes** as structured blocks

Example structure:
```markdown
# Project Codebook

Introduction and scope...

## User Experience

Context about this theme...

[code definition]

[another code]

## Technical Issues

...
```

## Writing Code Definitions

Each code is a `json-callout` block with `type: "codebook-code"`:

```json-callout
{
  "id": "[uuid-callout]",
  "type": "codebook-code",
  "title": "Code Name",
  "color": "blue",
  "collapsed": false,
  "content": """
Definition: What this code captures (1-2 sentences).

Inclusion criteria:
- Specific signal or marker
- Another indicator

Exclusion criteria:
- What this does NOT include

Examples:
- "Illustrative quote or phrase"

Counter examples:
- "Phrase that seems to fit but doesn't"
"""
}
```

### Title

- Short, descriptive label (2-5 words)
- Noun phrases: "User Frustration", "Technical Barrier", "Positive Feedback"
- Consistent naming style across the codebook

### Content

The definition answers: *When does this code apply?*

Content uses `"""` fences and supports markdown formatting: **bold**, *italic*, bullet lists, headings, etc. The content appears as regular file lines — patch it like normal markdown.

Structure:

**Definition** — Single paragraph. What this code captures (1-2 sentences). The conceptual essence.

**Inclusion Criteria** — Bulleted list. Specific signals, markers, and patterns that indicate this code applies. Linguistic markers, speech acts, observable features.

**Exclusion Criteria** — Bulleted list. What this code does NOT include. Boundary conditions distinguishing from similar codes.

**Examples** — Bulleted list of brief illustrative phrases or quotes that fit.

**Counter Examples** — Bulleted list of phrases that might seem to fit but should NOT be coded.

## Quality Principles

**Clarity over completeness** — A focused definition beats a comprehensive one. Matches should be recognizable instantly.

**Mutual exclusivity** — Overlapping codes create inconsistency. Each code should have clear boundaries.

**Observable over interpretive** — "Mentions waiting time" beats "Feels impatient". Ground codes in what's said, not inferred states.

**Consistent granularity** — Codes at similar abstraction levels. Don't mix "Mentioned product" with "Expressed deep existential frustration".

## Colors

Colors group related codes visually. Available:

- **Reds**: tomato, red, ruby, crimson
- **Pinks**: pink, plum, purple, violet
- **Blues**: iris, indigo, blue, sky, cyan
- **Greens**: teal, jade, green, grass, mint, lime
- **Yellows**: yellow, amber, orange
- **Browns**: brown, bronze, gold, sand
- **Neutrals**: olive, sage, mauve, slate, gray

Use consistent colors within themes.

## Workflow: Creating a Codebook

### Fast Path: Format Conversion

If the user has existing code definitions and wants them **reformatted** (not reconceptualized), skip to step 2b. Read the source, convert each code to `json-callout` format, preserve all content. Don't ask about research questions or methodology — the codes already exist.

### 1. Understand the Goal

Ask if unclear:
- What's the research question?
- Deductive (predefined codes) or inductive (emerge from data)?
- Any existing codes, frameworks, or literature to draw from?

### 2. If Converting Existing Codes

User may have codes in plain text, a list, or wrong format:

a) Read the source material
b) For each code/theme found:
   - Create proper code definition
   - Preserve their intent, improve structure
   - Add inclusion/exclusion criteria if missing
c) Write incrementally—one code at a time, not all at end

### 3. If Building From Scratch

a) **Gather input** — What themes matter? Any initial ideas?
b) **Propose structure** — Suggest top-level themes/categories
c) **Draft codes** — Start with a few, get feedback
d) **Iterate** — Refine definitions, add codes as needed

### 4. Review

After initial codebook exists:
- Check for overlapping codes (merge or clarify boundaries)
- Check for gaps (themes in data not covered)
- Verify granularity is consistent

## Edge Cases

**User has vague codes** ("stuff about politics")
→ Help sharpen: "What specific political aspects? Mentions of parties? Policy positions? Electoral rhetoric?"

**Codes overlap**
→ Clarify boundaries or merge: "These two codes seem to cover similar ground—should we combine them or specify what distinguishes them?"

**User wants too many codes**
→ Gently push back: "30 codes may be hard to apply consistently. Could we group some under broader themes?"

**User wants codes that require inference**
→ Suggest observable alternatives: "Instead of 'feels angry', what about 'uses aggressive language' or 'makes accusations'?"

## Codebook Operations

These are semantic tasks that require judgment. Explore first, then plan.

### Trigger

User asks to:
- "Restructure/refactor my codebook"
- "Merge these codebooks"
- "Clean up/improve the codebook"
- "Combine these code lists"
- "Review codebook for issues"

### Codebook Refactor

Explore the existing codebook first (read it, understand the codes, identify issues). Then plan — write to a new file, don't patch the original:

```
create_plan:
  task: "Refactor codebook"
  files: ["codebook.md"]
  steps:
    - title: "Create Codebook (new).md with target structure skeleton"
      expected: "New file with proposed headings/hierarchy, no content yet"
    - per_section:
        - title: "Write codes from this section into new structure"
          expected: "Codes written to correct location in Codebook (new).md"
    - title: "Review and clean up"
      expected: "Codebook (new).md coherent, no orphans, no duplicates"
    - title: "Report to user"
      expected: "User informed: new version in Codebook (new).md, original unchanged"
```

### Codebook Merge

Explore both codebooks first (read them, understand overlap, identify conflicts). Then plan — write the merged result to a new file:

```
create_plan:
  task: "Merge codebooks"
  files: ["codebook-a.md", "codebook-b.md"]
  steps:
    - title: "Normalize both codebooks"
      expected: "Each codebook as flat list: code title, definition, parent group"
    - title: "Identify overlaps and conflicts"
      expected: "Duplicates, near-duplicates, unique to each, conflicting definitions"
    - title: "Resolve with user"
      expected: "User decisions on conflicts, or 'no conflicts'"
    - title: "Write merged codebook to Codebook (new).md"
      expected: "Single codebook written, conflicts resolved, originals unchanged"
    - title: "Report to user"
      expected: "User informed: merged version in Codebook (new).md, originals unchanged"
```

### Decision Guide

| Signal | Path |
|--------|------|
| "Merge codebooks" (just concatenate) | Direct execution |
| "Merge and deduplicate/reconcile" | Explore, then plan (merge template) |
| "Restructure/refactor codebook" | Explore, then plan (refactor template) |
| "Add these codes to codebook" | Direct execution |
| "Review codebook for issues" | Explore, then answer or plan |
</codebook>
