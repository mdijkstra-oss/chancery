<coding-mechanics>
# Annotation shape

Each annotation contains:
- **text** — the passage from the document being annotated
- **reason** — why this passage was annotated. Ground it in the text.
- **code** — the codebook code ID (e.g. `code_a1b2c3d4`). For coding, always use `code`.
- **color** — for non-coding highlights (e.g. `teal`, `amber`, `violet`). Use `color` or `code` — never both.
- **review** — optional. Flags for human review — explain what needs attention.

## Annotation examples

Clean coding — no review:

```json
{
  "text": "we are constantly running on empty and nobody seems to care",
  "reason": "Direct expression of burnout and perceived management indifference",
  "code": "code_a1b2c3d4"
}
```

Coded annotation with review — codebook definition stretch:

```json
{
  "text": "the training was inadequate and the timeline was unrealistic",
  "reason": "Training gaps expressed alongside systemic frustration",
  "code": "code_k9m2n4p6",
  "review": "The training-gaps code covers learning deficits but this passage also expresses frustration with structural constraints (timeline). The definition may need broadening to include systemic context around training failures."
}
```

Color annotation with review — uncertain relevance:

```json
{
  "text": "I remember thinking at the time that something felt off",
  "reason": "Retrospective sense-making that may signal an unprocessed critical incident",
  "color": "amber",
  "review": "Ambiguous whether this is analytically relevant or casual reflection. Flagging for researcher judgment."
}
```
</coding-mechanics>
