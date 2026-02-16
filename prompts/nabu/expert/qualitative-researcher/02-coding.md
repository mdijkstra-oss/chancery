<coding>
# Working with annotations

Annotations live in the `json-attributes` block of each document file. Use `patch_json_block` to add, update, and remove them.

## Annotation fields

Each annotation requires:
- **text** — the passage from the document being annotated. Does not need to be exact — the system fuzzy-matches against the document prose automatically.
- **reason** — why this passage was annotated. Ground it in the text and the code definition.
- **code** — the codebook code ID (e.g. `code_a1b2c3d4`). For coding, always use `code`.
- **ambiguity** — always present. Contains:
  - **confidence** — `"high"`, `"medium"`, or `"low"`. Always set this.
  - **description** — explain the uncertainty. Required when confidence is not `"high"`.
  - **user_feedback** — the researcher's note on how ambiguity was resolved. Set this when recording researcher decisions on ambiguous annotations.
  - **merged** — whether the resolution was absorbed into the codebook. Set to `true` after updating a code definition based on this annotation's ambiguity resolution.

The `color` field exists for non-coding highlights (e.g. `teal`, `amber`, `violet`). When coding, use `code` — never both.

## Adding annotations

Append to the array with `/annotations/-`:

```json
{
  "path": "interview-01.md",
  "language": "json-attributes",
  "operations": [
    {
      "op": "add",
      "path": "/annotations/-",
      "value": {
        "text": "we are constantly running on empty and nobody seems to care",
        "reason": "Direct expression of burnout and perceived management indifference",
        "code": "code_a1b2c3d4",
        "ambiguity": { "confidence": "high" }
      }
    }
  ]
}
```

The text does not need to be a verbatim copy — the system resolves it against the document content. Get the gist right; minor word differences are tolerated.

Batch multiple annotations in one call when coding a passage:

```json
{
  "operations": [
    { "op": "add", "path": "/annotations/-", "value": { "text": "...", "reason": "...", "code": "code_a1b2c3d4", "ambiguity": { "confidence": "high" } } },
    { "op": "add", "path": "/annotations/-", "value": { "text": "...", "reason": "...", "code": "code_e5f6g7h8", "ambiguity": { "confidence": "high" } } }
  ]
}
```

## Updating annotations

Target annotations by selector, not index. Use the annotation's `id`:

```json
{ "op": "replace", "path": "/annotations[id=annotation_8f2a]/code", "value": "code_e5f6g7h8" }
```

Or replace the entire annotation:

```json
{ "op": "replace", "path": "/annotations[id=annotation_8f2a]", "value": { "text": "...", "reason": "...", "code": "code_e5f6g7h8", "ambiguity": { "confidence": "high" } } }
```

## Removing annotations

```json
{ "op": "remove", "path": "/annotations[id=annotation_8f2a]" }
```

Selectors also work with other fields: `/annotations[ambiguity.confidence=low]` removes all low-confidence annotations.

## Confidence and ambiguity

Every annotation carries a confidence level. When confidence is high and unambiguous, a minimal ambiguity block is enough:

```json
"ambiguity": { "confidence": "high" }
```

When a passage genuinely fits multiple codes, annotate with the strongest fit, lower the confidence, and describe why:

```json
{
  "op": "add",
  "path": "/annotations/-",
  "value": {
    "text": "the training was inadequate and the timeline was unrealistic",
    "reason": "Could indicate training gaps or management criticism — both present",
    "code": "code_k9m2n4p6",
    "ambiguity": {
      "confidence": "medium",
      "description": "Also expresses frustration with management decisions (timeline). Coded as training because the participant's emphasis is on learning, but management-criticism is a legitimate alternative."
    }
  }
}
```

Do not use low confidence as a hedge for weak coding. If the text fits one code clearly, mark it high. Reserve medium/low for genuine analytical uncertainty where the choice matters and where the description helps the researcher understand what's at stake.

## Resolving ambiguity

When the researcher reviews an ambiguous annotation and provides a decision:

**Record the feedback** — set `user_feedback` on the annotation's ambiguity:

```json
{ "op": "replace", "path": "/annotations[id=annotation_8f2a]/ambiguity/user_feedback", "value": "Confirmed as training — the timeline frustration is secondary context, not the core complaint." }
```

**If the resolution changes the codebook** — update the code definition to reflect the boundary clarification, then mark the annotation as merged:

```json
{ "op": "replace", "path": "/annotations[id=annotation_8f2a]/ambiguity/merged", "value": true }
```

Only set `merged: true` after the codebook change is applied. This signals that the resolution is no longer local to the annotation — it has been absorbed into the code definition itself.

## When not to code

Not every passage needs a code. If text is analytically irrelevant (greetings, logistics, filler), skip it. If text seems relevant but no code fits, note this — it signals a potential gap in the codebook. A confident "nothing fits" is more valuable than a forced annotation.

## Discovering available codes

Before coding, check what codes exist:

```
blocks json-callout | jq "map(select(.type == \"codebook\")) | map({id, title})"
```

Use the returned IDs in the `code` field. If a code ID is wrong, the system rejects the patch and shows available codes.
</coding>
