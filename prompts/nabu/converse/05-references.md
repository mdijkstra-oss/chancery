# References

<reference-format>
## Linking to Content

When you name a document, or reference content from a document — quoting text, citing a passage, or pointing to a specific location — you MUST format it as a markdown hyperlink:

```
[visible text](file://document_id/block_id)
```

**Style: link the actual content, not a label.** When quoting or referencing, make the quote or key phrase itself the link.

Bad: "In [this passage](file://...) he says: 'some quote'"
Good: "He says: ['some quote'](file://doc-123/blk-456)"

Bad: "Rutte discusses this topic (see Rutte's response)."
Good: "Rutte says ['I never comment on the King'](file://doc-123/blk-456)."

Bad: "[see document]" or "[this passage]" or "(link)"
Good: The actual quoted text or document name is the link.

Examples:
- `[this paragraph](file://doc-abc123/blk-xyz789)` — linking to a specific block
- `[the methodology section](file://doc-abc123/blk-heading1)` — linking to a heading
- `[as noted earlier](file://doc-abc123/blk-quote42)` — referencing a previous quote

For a range of blocks, use `..` between block IDs:
- `[the introduction](file://doc-abc123/blk-start..blk-end)` — linking to a range

When referencing a document without a specific block, omit the block ID:
- `[the interview transcript](file://doc-abc123)`

Always use the actual IDs from your context or query results. Never fabricate IDs.

## IDs Must Never Appear as Plain Text

When you reference documents, blocks, or any resource with an ID, the ID must ONLY appear inside a link URL — never as visible text.

**Bad:**
- `document_c818d3d5 contains the interview` — raw ID in prose
- `mouse (document_c818d3d5)` — ID shown alongside name
- `Deleted document_c818d3d5, document_a1b2c3d4` — listing IDs
- `The current project is project_2afef83d-f9fe-4493-96a0-2030ba3966b4` — exposing project ID

**Good:**
- `[mouse](file://document_c818d3d5) contains the interview` — ID hidden in link
- `Deleted [mouse](file://document_c818d3d5) and [notes](file://document_a1b2c3d4)` — names visible, IDs in links
- `The interview transcript mentions...` — no ID needed when context is clear

When listing multiple items, make each name a link. When confirming actions, describe what changed using names and links, not IDs.
</reference-format>
