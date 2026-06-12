<edit>
## edit_file

Replace a unique substring in an existing file. Match resolves in two stages: exact substring first; if not found, token-strict (case-insensitive, punctuation- and diacritic-insensitive, Unicode-aware).

### When to use
- Modify markdown prose, headings, lists, tables.
- Append by anchoring on a unique trailing passage and adding new text after it.
- Delete a passage by setting `replacement` to an empty string.

Not for:
- Anything inside a `json-*` block — use the typed tools (`patch_<type>`, `add_<type>`, `delete_<type>`).
- Creating a new file — use `create_file`.

### Matching rules
- The needle must resolve to exactly one location. Ambiguous → error: add more surrounding context, or use `...`.
- `...` (literal three dots, once per needle) elides the middle of a range. Format: `before ... after`. Each anchor must resolve uniquely; the replacement replaces the entire span from `before.start` to `after.end`.
- Anchors are matched on the raw content. Whitespace inside an anchor is significant unless the token-strict fallback engages.

### Block boundaries
- The matched span cannot overlap a `json-*` block. The tool refuses with the responsible typed tool to use.
- The `replacement` cannot introduce a `json-*` fence. Use `add_<type>` / `patch_<type>` to place and populate structured blocks.

### Examples

Simple replace:
```
needle:      "Found the process confusing"
replacement: "Found the onboarding process confusing at first, but adapted quickly"
```

Range with ellipsis:
```
needle:      "## Follow-up Questions\nDraft list...How will we measure adoption?"
replacement: "## Follow-up Questions\n\nFinalized list — see attached."
```

Append after a unique closing:
```
needle:      "## Conclusion\nReady for review."
replacement: "## Conclusion\nReady for review.\n\n## Next Steps\nSchedule a follow-up."
```

Delete a passage:
```
needle:      "Outdated paragraph that no longer applies.\n"
replacement: ""
```

### Discipline
- One logical change per call. Send several `edit_file` calls in one response for independent edits.
- Same file, overlapping needles: sequence them across responses — later edits must see the prior result.
- Edits to structured-block fields go through their typed tool, not here.

### Recovery
- "needle matches multiple locations" → grow the needle, or wrap it in `before ... after` to anchor a range.
- "needle not found" → re-read the file; quote real text rather than paraphrasing.
- "block is read-only" → switch to the named `patch_<type>` / `delete_<type>` tool.
- "Cannot create `json-*` block" → use `add_<type>` then `patch_<type>`.
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
