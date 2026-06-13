<edit>
## edit_file

Find a unique span in a file and replace it. Match resolves in two stages: exact substring first; if not found, token-strict (case-insensitive, punctuation- and diacritic-insensitive, Unicode-aware).

### When to use
- Modify markdown prose, headings, lists, tables.
- Append by anchoring on a unique trailing passage and writing the new tail in `replacement`.
- Delete a passage by setting `replacement` to an empty string.

Not for:
- Anything inside a `json-*` block — use the typed tools (`patch_<type>`, `add_<type>`, `delete_<type>`).
- Creating a new file — use `create_file`.

### Match shapes

Pass exactly one `match` object. The discriminator `type` chooses the shape.

**`full_anchor`** — you can quote the exact text to replace.
```
"match": { "type": "full_anchor", "anchor": "..." }
```
The whole matched span becomes the edit target. The anchor is replaced by `replacement`.

**`spanned_anchor`** — the span is too long or messy to quote, but the start and end edges are easy to pin down uniquely.
```
"match": { "type": "spanned_anchor", "anchor_start": "...", "anchor_end": "..." }
```
The replaced span runs from the start of `anchor_start` through the end of `anchor_end`. Both anchors are inside the replaced span (not preserved). `anchor_end` is searched in the content **after** `anchor_start`'s end.

Pick `full_anchor` whenever you can. Reach for `spanned_anchor` only when the middle is long or contains text you don't want to reproduce verbatim.

### Anchor rules
- Each anchor must resolve to exactly one location. Multiple matches → error with line context per match.
- Anchors compare on raw bytes for exact-substring, then fall back to token-strict normalization. Whitespace inside an anchor is significant unless the token fallback engages.

### Block boundaries
- The matched span cannot overlap a `json-*` block. JSON-block bytes are invisible to anchor matching — if your anchor only exists inside a block, you'll get "not found".
- `replacement` cannot introduce a `json-*` fence. Use `add_<type>` / `patch_<type>` to place structured blocks.

### Examples

Simple replace:
```
"match": { "type": "full_anchor", "anchor": "Found the process confusing" }
"replacement": "Found the onboarding process confusing at first, but adapted quickly"
```

Range with edge anchors:
```
"match": {
  "type": "spanned_anchor",
  "anchor_start": "## Follow-up Questions",
  "anchor_end":   "How will we measure adoption?"
}
"replacement": "## Follow-up Questions\n\nFinalized list — see attached."
```

Append after a unique closing:
```
"match": { "type": "full_anchor", "anchor": "## Conclusion\nReady for review." }
"replacement": "## Conclusion\nReady for review.\n\n## Next Steps\nSchedule a follow-up."
```

Delete a passage:
```
"match": { "type": "full_anchor", "anchor": "Outdated paragraph that no longer applies.\n" }
"replacement": ""
```

### Discipline
- One logical change per call. Send several `edit_file` calls in one response for independent edits.
- Same file, overlapping anchors: sequence them across responses — later edits must see the prior result.
- Edits to structured-block fields go through their typed tool, not here.

### Recovery
- "anchor matches multiple locations" — grow the anchor, or switch to `spanned_anchor` and pin both edges.
- "not found" — re-read the file; quote real text rather than paraphrasing.
- "anchor_end not found following anchor_start" — verify the end text actually appears after the start text.
- "Cannot create `json-*` block" — use `add_<type>` then `patch_<type>`.
</edit>

<document-attributes>
## Document Attributes

Document attributes are stored in a `json-attributes` block embedded in the markdown file:

```markdown
# My Document

Content here...

```json-attributes
{
  "tags": ["tag-abc12345", "tag-def67890"],
  ...
}
```

### Tagging Files

Tag definitions live in `settings.hidden.md` (`json-settings` block). Discover existing definitions:

```
blocks json-settings | jq ".[0].tags // []"
```

In `json-attributes`, reference tags by ID: `"tags": ["tag-abc12345"]`. In prose, use `#label` form (e.g. `#interview`) — auto-linkified in the UI.

`preferences.md` and `settings.hidden.md` are protected — they cannot be deleted or renamed.

### Validation on patch
When you patch a structured block, the system validates schema (correct types, required fields). On failure you receive the error and the unchanged block content so you can retry from the correct state. Only one `json-attributes` block per file.
</document-attributes>

<json-blocks>
## JSON Structured Blocks

`edit_file` and `create_file` refuse to touch `json-*` blocks. Use the typed tools.

### Creating blocks
1. Write surrounding prose with `create_file` / `edit_file`.
2. Place the block: `add_callout` / `add_chart` (returns the generated `block_id`).
3. Populate fields: `patch_callout` / `patch_chart` with that `block_id`.

Singletons (`json-attributes`, `json-settings`, `json-annotations`) are placed by their `patch_<type>` directly.

### Immutable fields
Some fields are immutable — settable once at create, never modifiable. `id` is always immutable. Trying to change one is rejected: `"id: immutable - already set to 'callout-x7k2m9p1'"`.

### Updating blocks
Use the typed patch tool. Reference existing blocks by their real ID, not a placeholder.
</json-blocks>
