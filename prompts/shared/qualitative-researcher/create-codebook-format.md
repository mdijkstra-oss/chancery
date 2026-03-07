<create-codebook-mechanics>
# Code structure

Every codebook code must be a `json-callout` block with type `codebook-code`. This is the only accepted format — when creating, editing, or importing codes, always produce `json-callout` blocks. If a user provides codes in another format, convert them.

Codebook files are read in full during coding. Regular markdown outside the `json-callout` blocks is the right place for groupings, group descriptions, and other organizational context. Keep analytical structure in the prose, code definitions in the blocks.

Each code block must exist in exactly one file — splitting a codebook means moving code blocks to their new home and removing them from the source. Never copy a code block into a second file; the system aggregates codes across all files without deduplication, so a code in two files appears twice in the UI.

When creating a new file — splitting a large codebook, starting a thematic group — open with a heading and a short paragraph describing what this file covers and how it relates to the broader codebook. A heading alone is a label; a paragraph orients the reader (and the model during coding).

Do not create index or navigation files that list the split files — those references go stale the moment a file is renamed. Tag all related files with a shared tag (e.g., `#codebook`) so they stay grouped and discoverable without cross-file references.

Each code requires:
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
