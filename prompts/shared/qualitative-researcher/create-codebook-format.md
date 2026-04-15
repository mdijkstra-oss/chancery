<create-codebook-mechanics>
# Code structure

Every codebook code must be a `json-callout` block with type `codebook-code`. This is the only accepted format — when creating, editing, or importing codes, always produce `json-callout` blocks. If a user provides codes in another format, convert them.

Codebook files are read in full during coding. Regular markdown outside the `json-callout` blocks is the right place for groupings, group descriptions, and other organizational context. Keep analytical structure in the prose, code definitions in the blocks.

A file can hold multiple codes — the file is the grouping mechanism. Codes in the same file form a thematic group; the filename and heading describe the group.

| File | Codes |
|------|-------|
| `economic_factors.md` | Cost Concern, Resource Scarcity, Financial Barrier |
| `emotional_responses.md` | User Frustration, Satisfaction, Anxiety |
| `process_issues.md` | Workaround, Process Friction, Compliance |

Never duplicate a code across files — the system aggregates codes from all files without deduplication, so a code in two files appears twice in the UI. Splitting a codebook means moving code blocks to their new home and removing them from the source.

When creating a new file — splitting a large codebook, starting a thematic group — open with a heading and a short paragraph describing what this file covers and how it relates to the broader codebook. A heading alone is a label; a paragraph orients the reader (and the model during coding).

Do not create index or navigation files that list the split files — those references go stale the moment a file is renamed. Tag all related files with a shared tag (e.g., `#codebook`) so they stay grouped and discoverable without cross-file references.

Each code requires:
- **type** — always `"codebook-code"`
- **title** — short, descriptive name for the code
- **content** — definition, criteria, and examples as a JSON string (use `\n` for newlines)
- **color** — visual identifier
- **collapsed** — `false` for active codes, `true` for stable ones

## Creating codes

1. Write the document prose first — heading, group description, any organizational context
2. `add_callout` to place an empty block after the right prose anchor
3. `patch_callout` to populate the block's fields using the returned `block_id`

## Example value shape

```json-callout
{
  "type": "codebook-code",
  "title": "User Frustration",
  "color": "tomato",
  "collapsed": false,
  "content": "Expressions of dissatisfaction with the product, process, or experience.\n\nInclusion criteria:\n- Direct complaints about specific features or processes\n- Negative evaluations of experience quality\n- Expressions of annoyance or disappointment\n\nExclusion criteria:\n- Neutral descriptions of difficulty (code as Process Friction instead)\n- Constructive suggestions without emotional valence\n\nExamples:\n- \"this is really annoying and I don't understand why it works this way\"\n- \"I gave up after the third attempt\"\n\nCounter-examples:\n- \"it took a few tries but I figured it out\" (neutral difficulty)\n- \"it would be better if the button were larger\" (constructive suggestion)"
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
