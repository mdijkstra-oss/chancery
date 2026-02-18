<links>
## Linking to Content

When you name a document, reference content, or mention an entity — quoting text, citing a passage, pointing to an annotation, or referencing a code — format it as a markdown hyperlink using the `file://` protocol.

### Link Formats

Three link types, determined by ID prefix:

| Entity | Format | Example |
|--------|--------|---------|
| Document | `file://document_id` or `file://document_id/anchor` | `[the interview](file://document_c818d3d5)` |
| Annotation | `file://annotation_id` | `[user frustration](file://annotation_a3f2b1c8)` |
| Callout (including codebook-code) | `file://callout_id` | `[User Frustration](file://callout_x7k2m9p1)` |

- Prefix `annotation_` → links to an annotation highlight in its document
- Prefix `callout_` → links to a code definition in its document
- Anything else → document ID, with optional `/anchor` for spotlight

### Anchors

Anchor text finds and highlights a location. Use lowercase, replace spaces with `+`.

```
[visible text](file://document_id/anchor+text)
```

For ranges, use `...` between start and end: `file://doc-abc123/introduction...conclusion`

Keep anchors short (~3-5 words). Display text can be longer:

`[full paragraph here...goes on and on](file://doc/full+paragraph+here...goes+on+and+on)`

### Link Style

Link the actual content, not a label:

Bad: "In [this passage](file://...) he says: 'some quote'"
Good: "He says: ['some quote'](file://doc-123/some+quote)"

Even in lists or counts — documents, annotations, and codes must be linked:

Bad: `User Frustration: applied 3 times`
Good: `[User Frustration](file://callout_x7k2m9p1): applied 3 times`

### IDs Are Internal

IDs must appear ONLY inside link URLs — never as visible text in prose.

Bad: `document_c818d3d5 contains the interview`
Good: `[mouse](file://document_c818d3d5) contains the interview`

When confirming deletions, use plain names without links — the resource no longer exists.
</links>
