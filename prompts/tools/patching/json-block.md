---
requires:
  - patch_json_block
---

<patch-json-block>
## Updating Block JSON with `patch_json_block`

For targeted changes to JSON properties within a block, use `patch_json_block` instead of `apply_local_patch`. It applies RFC 6902 JSON Patch operations to the block's parsed JSON and produces the file diff automatically.

```json
{
  "path": "document.md",
  "language": "json-attributes",
  "operations": [
    { "op": "replace", "path": "/tags/1", "value": "final" },
    { "op": "add", "path": "/annotations/-", "value": { "text": "...", "code": "code_abc" } }
  ]
}
```

**Supported operations:** `add`, `remove`, `replace`, `move`, `copy`, `test`

**Paths** use JSON Pointer syntax (RFC 6901):
- `/field` — top-level field
- `/array/-` — append to array
- `/nested/deep/field` — nested access

### Array selectors

Target array items by field value, not by index. Numeric indices (`/annotations/0`) are rejected — use a selector instead:

- `/annotations[id=annotation_8f2a]` — match by id
- `/annotations[id=annotation_8f2a]/code` — nested field on matched item
- `/annotations[ambiguity.confidence=medium]` — nested key with dot notation

Selectors match **all** items with the given key=value.

Annotation `text` is automatically fuzzy-matched against the document prose. You do not need to quote it exactly.

### When to use which

- `patch_json_block` — changing specific JSON properties, adding/removing array entries, any structured data change
- `apply_local_patch` — changing prose, markdown structure, multi-line `"""` content, or non-JSON parts of the file
</patch-json-block>
