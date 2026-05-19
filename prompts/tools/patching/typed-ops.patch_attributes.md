<typed-block-tools>
## Typed block tools

Each block type has its own tools. Operations are schema-validated — use the right tool for the block type and the schema handles the rest.

| Block | Patch | Delete | Add | Move | Singleton |
|-------|-------|--------|-----|------|-----------|
| `json-attributes` | `patch_attributes` | `delete_attributes` | — | — | yes |
| `json-annotations` | `patch_annotations` | `delete_annotations` | — | — | yes |
| `json-callout` | `patch_callout` | `delete_callout` | `add_callout` | `move_callout` | no |
| `json-settings` | `patch_settings` | `delete_settings` | — | — | yes |
| `json-chart` | `patch_chart` | `delete_chart` | `add_chart` | `move_chart` | no |

### Patch operation types

- **`set`** — partial field update (scalars, full array/object replacement). Fine for short fields. For long multiline fields, prefer `patch_<field>` — `set` replaces the entire field value.
- **`add_<item>`** — append to an object array (`add_annotation`, `add_tag`, `add_search`)
- **`remove_<item>`** — remove by ID (`remove_annotation`, `remove_tag`, `remove_search`)
- **`set_<item>`** — partial field update on item by ID (`set_annotation`, `set_tag`, `set_search`). Do not include unchanged fields — only the fields that actually change. Rewriting `text` or `reason` when only `code` or `color` changed is wrong. Set a field to `null` to remove it.
- **`patch_<field>`** — apply a targeted V4A diff to a multiline string field (e.g. `patch_content` for callout `content`). Use instead of `set` when the field is long — sends only the changed lines, not the full value. Fewer tokens, more precise. See format below.

Batch related changes as multiple operations in one call.

### patch_\<field\> diff format

The `diff` value uses V4A format — context lines locate the change, `-`/`+` lines express it:

```
@@
context line (no prefix, must match existing text)
-line to remove
+line to add
context line
```

- `@@` starts a hunk. Multiple hunks for scattered changes.
- Context lines (no prefix): anchor the change in the field. **Include only 2-3 lines** — enough to match uniquely. Never reproduce the entire field as context.
- `-` lines: content to remove.
- `+` lines: content to add. **Markdown list items** starting with `- ` must be prefixed: `+- item` to add, `-- item` to remove (the first character is the diff operator, the rest is content).
- `...` between context anchors: skip large unchanged sections.
- Long context lines: end with `...` instead of reproducing the full line. The system prefix-matches against the field content (min 80 chars before `...`).

**Example — swap one phrase in a long content field:**

```json
{
  "op": "patch_content",
  "diff": "@@\n-Apply to fiscal figures.\n+Apply to macro-fiscal figures and trajectories."
}
```

**Example — two scattered changes with skip:**

```json
{
  "op": "patch_content",
  "diff": "@@\n-old phrase near the top\n+new phrase near the top\n...\n-old phrase near the bottom\n+new phrase near the bottom"
}
```

If the patch fails with "context not found", re-read the block and retry with correct context. If it fails with "ambiguous", add more context lines to disambiguate.

### Add and move tools

Non-singleton blocks (`json-callout`, `json-chart`) have `add_*` and `move_*` tools.

- **`add_<type>`** — insert a new empty block after an anchor position. Returns the generated `block_id`. Follow up with `patch_<type>` to populate the block's fields.
- **`move_<type>`** — relocate an existing block to a new anchor position.

Both take a `context` parameter — a few lines of prose from the document that uniquely identify the insertion point. The block is placed after the matched context. If the context matches zero or multiple locations, the tool returns an error with guidance.

### Annotation text fuzzy-matching

Annotation `text` is automatically fuzzy-matched against the document prose — exact quoting is not required.

### File-locked blocks

`json-settings` blocks only exist in `settings.hidden.md`. Target that file for all tag and search changes.

### When to use which

- **`add_<type>`** — create a new non-singleton block at a specific position. Preferred over `apply_local_patch` for block creation — handles ID generation and validates placement.
- **Typed patch tools** — any change to a JSON block: adding/removing annotations, updating properties, modifying tags, changing colors. Match the tool to the block type.
- **`patch_<field>`** — for long multiline text fields. Use instead of `set` when the field has substantial content — surgical diff against the field value, not a full replacement.
- **`move_<type>`** — reorder a non-singleton block within the document.
- **Typed delete tools** — remove an entire JSON block from a document. Use when the block itself should cease to exist, not when removing items within it.
- `apply_local_patch` — prose, markdown structure, or anything outside a JSON block. Never use it to rewrite a JSON block.
  </typed-block-tools>