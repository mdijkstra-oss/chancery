<create-codebook>
# Creating codebook codes

## Process

Creating a codebook is iterative. As per Braun & Clarke's reflexive TA:

1. **Familiarize** — sample 2-3 documents to understand the data before writing any codes. Read for patterns, recurring language, surprising moments.
2. **Establish the analytical lens** — what is the research question? What matters in this data? The researcher defines this; you help sharpen it.
3. **Draft initial codes** — grounded in corpus language, not abstract categories. Each code should be recognizable in the data.
4. **Test against data** — apply draft codes to passages. What fits? What doesn't? What's missing?
5. **Refine** — split overlapping codes, tighten definitions, add exclusion criteria. A good codebook emerges from iteration, not from a single draft.

Do not create codes speculatively. If a pattern is unclear, sample more data first.

## Code quality

Each code should have (per Boyatzis):
- A clear definition grounded in the data's own language
- Inclusion criteria — what qualifies
- Exclusion criteria — what looks similar but doesn't belong, and which code captures it instead
- At least 2 examples from actual data
- At least 2 counter-examples showing what the code does *not* capture

## Code structure

Codebook codes are `json-callout` blocks with type `codebook-code`. Each code requires:
- **type** — always `"codebook-code"`
- **title** — short, descriptive name for the code
- **content** — definition, criteria, and examples in `"""` fenced content
- **color** — visual identifier
- **collapsed** — `false` for active codes, `true` for stable ones

## Example

```json-callout
{
  "id": "[uuid-callout]",
  "type": "codebook-code",
  "title": "User Frustration",
  "color": "tomato",
  "collapsed": false,
  "content": """
Expressions of dissatisfaction with the product, process, or experience.

Inclusion criteria:
- Direct complaints about specific features or processes
- Negative evaluations of experience quality
- Expressions of annoyance or disappointment

Exclusion criteria:
- Neutral descriptions of difficulty (code as Process Friction instead)
- Constructive suggestions without emotional valence

Examples:
- "this is really annoying and I don't understand why it works this way"
- "I gave up after the third attempt"

Counter-examples:
- "it took a few tries but I figured it out" (neutral difficulty)
- "it would be better if the button were larger" (constructive suggestion)
"""
}
```

## Color assignment

Pick colors that create strong visual distinction between codes:

| Category | Suggested colors |
|----------|-----------------|
| Emotions | tomato, red, crimson, pink |
| Actions | blue, cyan, sky, indigo |
| Themes | purple, violet, teal, green |
| Speakers | brown, bronze, gold, amber |

Check existing codes before assigning a color — avoid duplicates.
</create-codebook>
