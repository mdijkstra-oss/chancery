
<patch-json-block>
## `patch_json_block` — extended path syntax

This tool uses an extended JSON Pointer syntax. Standard RFC 6901 paths work (`/field`, `/nested/deep`, `/array/-` to append), but array access is **selector-based, not index-based**.

### Selectors (not indices)

Numeric indices like `/annotations/0` are rejected. Target array items by selector:

```
/annotations[id=annotation_8f2a]              — exact match
/annotations[id=annotation_8f2a]/code         — field on matched item
/annotations[code!=code_a1b2c3d4]             — not equals
/annotations[review]                          — field exists (truthy)
/annotations[!review]                         — field absent or empty
```

Selectors match **all** items that satisfy the condition. A `remove` on `/annotations[review]` removes every flagged annotation.

### Annotation text fuzzy-matching

Annotation `text` is automatically fuzzy-matched against the document prose — you do not need to quote it exactly.

### Batching

Batch related changes as multiple operations in one call.

### When to use which

- `patch_json_block` — changing specific JSON properties, adding/removing array entries, any structured data change
- `apply_local_patch` — changing prose, markdown structure, multi-line `"""` content, or non-JSON parts of the file
</patch-json-block>
