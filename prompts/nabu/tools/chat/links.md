---
requires:
  - chat
---

<links>
## Linking to Content

When you name a document, reference content, or mention an entity — quoting text, citing a passage, pointing to an annotation, or referencing a code — you MUST format it as a markdown hyperlink using the `file://` protocol.

### Link Formats

There are three link types, determined by the ID prefix:

| Entity                                        | Format | Example |
|-----------------------------------------------|--------|---------|
| Document                                      | `file://document_id` or `file://document_id/anchor` | `[the interview](file://document_c818d3d5)` |
| Annotation                                    | `file://annotation_id` | `[user frustration](file://annotation_a3f2b1c8)` |
| Callout of any type (including codebook-code) | `file://callout_id` | `[User Frustration](file://callout_x7k2m9p1)` |

**Rules:**
- Prefix `annotation_` → links to an annotation highlight in its document
- Prefix `callout_` → links to a code definition block in its document
- Anything else → treated as a document ID, with optional `/anchor` for spotlight

### Document Links

```
[visible text](file://document_id)
[visible text](file://document_id/anchor+text)
```

The anchor text is used to find and highlight the location in the document. Use lowercase, replace spaces with `+`.

For a range of content, use `...` between start and end anchors:
- `[the introduction](file://doc-abc123/introduction...conclusion)`

When referencing a document without a specific location, omit the anchor:
- `[the interview transcript](file://doc-abc123)`

### Annotation Links

When referring to an annotation — a highlighted passage with a color or code — link to its annotation ID then DO NOT use the document anchors:

```
[the highlighted passage](file://annotation_a3f2b1c8)
```

The system resolves this to the correct document and scrolls to the annotated text.

Use annotation links when:
- Discussing a specific annotation you created or found
- Referencing an annotated passage by its annotation identity
- Listing annotations in a summary
- You reference the annotation in any way.

### Code / Callout Links

When referring to a code from the codebook — link to its callout ID:

```
[User Frustration](file://callout_x7k2m9p1)
```

The system resolves this to the codebook document and scrolls to the code definition.

Use code links when:
- Mentioning a code by name in analysis
- Listing codes applied to a passage
- Referring to a code definition

### Link Style

Link the actual content, not a label. When quoting or referencing, make the quote or key phrase itself the link.

Bad: "In [this passage](file://...) he says: 'some quote'"
Good: "He says: ['some quote'](file://doc-123/some+quote)"

Bad: "[see document]" or "[this passage]" or "(link)"
Good: The actual quoted text or document name is the link.

### Anchor Length

Keep anchors short (~3-5 words). For longer quotes, use a range with short start/end anchors:

Bad: `[full paragraph here that goes on and on](file://doc/full+paragraph+here+that+goes+on+and+on)`
Good: `[full paragraph here...goes on and on](file://doc/full+paragraph+here...goes+on+and+on)`

The display text can be the full quote, but the URL anchors should be brief.

### Always Link Entity Names

Even in lists or counts — documents, annotations, and codes must be linked:

Bad:
```
- 2020-09-04- Ministerraad 4 September.md: 10
- User Frustration: applied 3 times
```

Good:
```
- [2020-09-04- Ministerraad 4 September](file://2020-09-04- Ministerraad 4 September.md): 10
- [User Frustration](file://callout_x7k2m9p1): applied 3 times
```

### IDs Are Internal

IDs must appear ONLY inside link URLs — never as visible text in prose.

Bad: `annotation_a3f2b1c8 highlights user frustration`
Good: `[this passage](file://annotation_a3f2b1c8) highlights user frustration`

Bad: `The code callout_x7k2m9p1 was applied 3 times`
Good: `[User Frustration](file://callout_x7k2m9p1) was applied 3 times`

The anchor uses fuzzy matching — minor differences in punctuation or casing are tolerated.
</links>
