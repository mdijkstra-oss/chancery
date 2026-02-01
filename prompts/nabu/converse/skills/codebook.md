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

Each code is a `json-callout` block with `type: "codebook"`:

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
</codebook>
