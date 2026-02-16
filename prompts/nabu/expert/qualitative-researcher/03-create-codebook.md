<create-codebook>
# Creating codebook codes

Codebook codes are `json-callout` blocks with type `codebook-code`. Each code lives in the codebook file as a separate block.

## Code structure

Each code requires:
- **type** — always `"codebook-code"`
- **title** — short, descriptive name for the code
- **content** — definition, inclusion/exclusion criteria, and examples. Use `"""` fences for multi-line content.
- **color** — visual identifier. Use warm tones for emotions, cool tones for actions, purples/greens for themes, earth tones for speakers. Avoid adjacent hues between codes.
- **collapsed** — `false` for active codes, `true` for stable ones

Use a `[uuid-callout]` placeholder for the `id` — the system generates the actual ID.

## Creating a single code

Use `apply_local_patch` to create the block:

```json
{
  "type": "update_file",
  "path": "codebook.md",
  "diff": "@@\n+\n+```json-callout\n+{\n+  \"id\": \"[uuid-callout]\",\n+  \"type\": \"codebook-code\",\n+  \"title\": \"User Frustration\",\n+  \"color\": \"tomato\",\n+  \"collapsed\": false,\n+  \"content\": \"\"\"\n+Expressions of dissatisfaction with the product, process, or experience.\n+\n+Inclusion criteria:\n+- Direct complaints about specific features or processes\n+- Negative evaluations of experience quality\n+- Expressions of annoyance or disappointment\n+\n+Exclusion criteria:\n+- Neutral descriptions of difficulty (code as Process Friction instead)\n+- Constructive suggestions without emotional valence\n+\n+Examples:\n+- \"this is really annoying and I don't understand why it works this way\"\n+- \"I gave up after the third attempt\"\n+\n+Counter-examples:\n+- \"it took a few tries but I figured it out\" (neutral difficulty)\n+- \"it would be better if the button were larger\" (constructive suggestion)\n+\"\"\"\n+}\n+```"
}
```

## Creating multiple codes

Use numbered placeholder suffixes for unique IDs within one patch:

- `[uuid-callout-1]`, `[uuid-callout-2]`, `[uuid-callout-3]`

Each suffix generates a distinct ID with the same `callout_` prefix.

## Before creating codes

Sample from 2-3 files to anchor definitions in real data. Each code should have:
- A clear definition grounded in corpus language
- At least 2 examples from actual data
- At least 2 counter-examples showing what the code does *not* capture
- Exclusion criteria distinguishing it from adjacent codes

Do not create codes speculatively. If a pattern is unclear, sample more data first.

## Color assignment

Pick colors that create strong visual distinction between codes in the same analysis:

| Category | Suggested colors |
|----------|-----------------|
| Emotions | tomato, red, crimson, pink |
| Actions | blue, cyan, sky, indigo |
| Themes | purple, violet, teal, green |
| Speakers | brown, bronze, gold, amber |

Check existing codes before assigning a color — avoid duplicates.
</create-codebook>
