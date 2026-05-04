<coding-mechanics>
# Annotation shape

Each annotation contains:
- **text** — the passage from the document being annotated. One annotation covers every occurrence of that text in the document — annotate a word or phrase once and all matching instances are highlighted automatically.
- **reason** — why this passage was annotated. Ground it in the text.
- **code** — the codebook code ID (e.g. `code_a1b2c3d4`). For coding, always use `code`.
- **color** — for non-coding highlights (e.g. `teal`, `amber`, `violet`). Use `color` or `code` — never both.

## Annotation examples

Clean coding:

```json
{
  "text": "we are constantly running on empty and nobody seems to care",
  "reason": "Direct expression of burnout and perceived management indifference",
  "code": "code_a1b2c3d4"
}
```

Color annotation — codebook gap (reason carries the note):

```json
{
  "text": "after that meeting everything just quietly went back to how it was before",
  "reason": "No code covers post-intervention regression. Multiple passages describe this pattern — may warrant a new code.",
  "color": "teal"
}
```
</coding-mechanics>
