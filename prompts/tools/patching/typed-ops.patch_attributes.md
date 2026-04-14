<typed-block-tools>
## Typed block patch tools

Each block type has its own patch and delete tool. Operations are schema-validated — use the right tool for the block type and the schema handles the rest.

| Block | Patch tool | Delete tool | Singleton |
|-------|-----------|-------------|-----------|
| `json-attributes` | `patch_attributes` | `delete_attributes` | yes |
| `json-annotations` | `patch_annotations` | `delete_annotations` | yes |
| `json-callout` | `patch_callout` | `delete_callout` | no (needs `block_id`) |
| `json-settings` | `patch_settings` | `delete_settings` | yes |
| `json-chart` | `patch_chart` | `delete_chart` | no (needs `block_id`) |

### Operation types

- **`update`** — partial field update (scalars, full array/object replacement). Fine for short fields. For long multiline fields, prefer `patch_<field>` — `update` replaces the entire field value.
- **`add_<item>`** — append to an object array (`add_annotation`, `add_tag`, `add_search`)
- **`remove_<item>`** — remove by ID (`remove_annotation`, `remove_tag`, `remove_search`)
- **`update_<item>`** — partial item update by ID (`update_annotation`, `update_tag`, `update_search`)
- **`patch_<field>`** — apply a V4A diff to a multiline string field (e.g. `patch_content` for callout `content`). Use instead of `update` when the field is long — sends only the changed lines, not the full value. Fewer tokens, more precise.

Batch related changes as multiple operations in one call.

### Annotation text fuzzy-matching

Annotation `text` is automatically fuzzy-matched against the document prose — exact quoting is not required.

### File-locked blocks

`json-settings` blocks only exist in `settings.hidden.md`. Target that file for all tag and search changes.

### When to use which

- **Typed patch tools** — any change to a JSON block: adding/removing annotations, updating properties, modifying tags, changing colors. Match the tool to the block type.
- **`patch_<field>`** — for long multiline text fields. Use instead of `update` when the field has substantial content — surgical diff against the field value, not a full replacement.
- **Typed delete tools** — remove an entire JSON block from a document. Use when the block itself should cease to exist, not when removing items within it.
- `apply_local_patch` — prose, markdown structure, or anything outside a JSON block. Never use it to rewrite a JSON block.
  </typed-block-tools>