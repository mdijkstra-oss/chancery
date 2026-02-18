<coding-mechanics>
# Annotation shape

Each annotation contains:
- **text** — the passage from the document being annotated
- **reason** — why this passage was annotated. Ground it in the text and the code definition.
- **code** — the codebook code ID (e.g. `code_a1b2c3d4`). For coding, always use `code`.
- **ambiguity** — always present:
  - **confidence** — `"high"`, `"medium"`, or `"low"`
  - **description** — explain the uncertainty. Required when confidence is not `"high"`.
  - **user_feedback** — the researcher's note on how ambiguity was resolved
  - **merged** — whether the resolution was absorbed into the codebook

The `color` field exists for non-coding highlights (e.g. `teal`, `amber`, `violet`). When coding, use `code` — never both.

## Annotation examples

When confidence is high and unambiguous:

```json
{
  "text": "we are constantly running on empty and nobody seems to care",
  "reason": "Direct expression of burnout and perceived management indifference",
  "code": "code_a1b2c3d4",
  "ambiguity": { "confidence": "high" }
}
```

When a passage genuinely fits multiple codes, annotate with the strongest fit, lower the confidence, and describe why:

```json
{
  "text": "the training was inadequate and the timeline was unrealistic",
  "reason": "Could indicate training gaps or management criticism — both present",
  "code": "code_k9m2n4p6",
  "ambiguity": {
    "confidence": "medium",
    "description": "Also expresses frustration with management decisions (timeline). Coded as training because the participant's emphasis is on learning, but management-criticism is a legitimate alternative."
  }
}
```
</coding-mechanics>
