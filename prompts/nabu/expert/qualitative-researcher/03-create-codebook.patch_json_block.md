<create-codebook-mechanics>
# Code structure

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
</create-codebook-mechanics>
