---
requires:
  - remove_block
---

<remove-block>
## Removing Blocks with `remove_block`

To remove an entire fenced code block from a document, use `remove_block`. It identifies the block by language and optionally by the `id` field in its JSON content.

```json
{
  "path": "document.md",
  "language": "json-callout",
  "id": "callout_x7k2m9p1"
}
```

- `id` is required when multiple blocks of the same language exist in the file
- `id` can be omitted for singleton blocks (e.g., the single `json-attributes` block)
- The tool produces a file diff — it goes through the same validation pipeline as `apply_local_patch`
</remove-block>
