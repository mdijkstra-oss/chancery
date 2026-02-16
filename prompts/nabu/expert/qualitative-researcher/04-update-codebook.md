<update-codebook>
# Updating codebook codes

Use `patch_json_block` to update individual fields on existing codebook codes. Use `apply_local_patch` for multi-line `"""` content changes.

## Updating fields

Change a code's color:

```json
{
  "path": "codebook.md",
  "language": "json-callout",
  "operations": [
    { "op": "replace", "path": "/color", "value": "crimson" }
  ]
}
```

Note: `json-callout` is not a singleton — when multiple callout blocks exist in the file, specify which one. Since codebook codes have unique IDs, use the block's ID to target it.

## Updating definitions

Code definitions live in the `content` field as `"""` fenced markdown. Patch these with `apply_local_patch`, not `patch_json_block`:

```json
{
  "type": "update_file",
  "path": "codebook.md",
  "diff": "@@\n Inclusion criteria:\n - Direct complaints about specific features or processes\n+- Passive-aggressive remarks about workflow design\n - Negative evaluations of experience quality"
}
```

## Refining from ambiguity

When accumulated annotations reveal a code boundary problem, update the code definition:

**Splitting a code** — when a code covers too much ground, create a new code for the narrower concept and update the original's exclusion criteria to point to the new one.

**Tightening a boundary** — when annotations keep landing ambiguously between two codes, add exclusion criteria or examples that clarify the distinction.

**Adding examples from data** — when a researcher confirms an ambiguous annotation, add the passage as an example or counter-example to the relevant code definition.

## Merge workflow

When an annotation's ambiguity resolution reveals something the codebook should capture permanently:

1. **Update the code definition** — add the clarification to inclusion/exclusion criteria or examples
2. **Set `merged: true`** on the annotation — signals the local resolution has been absorbed into the codebook:

```json
{
  "path": "interview-01.md",
  "language": "json-attributes",
  "operations": [
    { "op": "replace", "path": "/annotations[id=annotation_8f2a]/ambiguity/merged", "value": true }
  ]
}
```

3. **Optionally record `user_feedback`** — if the researcher gave a verbal rationale, capture it before merging:

```json
{ "op": "replace", "path": "/annotations[id=annotation_8f2a]/ambiguity/user_feedback", "value": "Treat timeline complaints as management-criticism when the participant attributes the deadline to a specific decision-maker." }
```

The sequence matters: update the codebook first, then mark `merged`. An annotation marked merged without a corresponding codebook change is misleading.

## When not to merge

Not every ambiguity resolution needs a codebook change:

- **One-off judgment calls** — the text is genuinely edge-case and the codebook is fine as-is. Record `user_feedback` but leave `merged` unset.
- **Researcher preference** — the researcher resolves it based on their knowledge of the participant or context, not because the code definition is lacking.
- **Insufficient pattern** — a single ambiguous instance is not enough to change a definition. Wait for a cluster before proposing codebook changes.
</update-codebook>
