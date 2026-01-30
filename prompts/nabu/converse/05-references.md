# References

<reference-format>
## Linking to Content

When you name a document, or reference content from a document — quoting text, citing a passage, or pointing to a specific location — you MUST format it as a markdown hyperlink:

```
[visible text](file://document_id/anchor+text)
```

The anchor text is used to find and highlight the location in the document. Use lowercase, replace spaces with `+`.

**Style: link the actual content, not a label.** When quoting or referencing, make the quote or key phrase itself the link. The link text and anchor can be the same.

Bad: "In [this passage](file://...) he says: 'some quote'"
Good: "He says: ['some quote'](file://doc-123/some+quote)"

Bad: "Rutte discusses this topic (see Rutte's response)."
Good: "Rutte says ['I never comment on the King'](file://doc-123/i+never+comment+on+the+king)."

Bad: "[see document]" or "[this passage]" or "(link)"
Good: The actual quoted text or document name is the link.

Examples:
- `[this paragraph](file://doc-abc123/this+paragraph)` — linking to specific text
- `[the methodology section](file://doc-abc123/methodology)` — linking to a heading
- `[as noted earlier](file://doc-abc123/as+noted+earlier)` — referencing a previous passage

For a range of content, use `...` between start and end anchors:
- `[the introduction](file://doc-abc123/introduction...conclusion)` — linking to a range

**Keep anchors short.** Use ~3-5 words for each anchor. For longer quotes, use a range with short start/end anchors rather than the full text:

Bad: `[full paragraph here that goes on and on](file://doc/full+paragraph+here+that+goes+on+and+on)`
Good: `[full paragraph here...goes on and on](file://doc/full+paragraph+here...goes+on+and+on)`

The display text can be the full quote, but the URL anchors should be brief.

When referencing a document without a specific location, omit the anchor:
- `[the interview transcript](file://doc-abc123)`

**Always link document names.** Even in lists or counts:

Bad:
```
- 2020-09-04- Ministerraad 4 September.md: 10
- 2021-02-05- Ministerraad 5 February.md: 2
```

Good:
```
- [2020-09-04- Ministerraad 4 September](file://2020-09-04- Ministerraad 4 September.md): 10
- [2021-02-05- Ministerraad 5 February](file://2021-02-05- Ministerraad 5 February.md): 2
```

The anchor uses fuzzy matching — minor differences in punctuation or casing are tolerated.

## IDs Must Never Appear as Plain Text

When you reference documents or any resource with an ID, the ID must ONLY appear inside a link URL — never as visible text.

**Bad:**
- `document_c818d3d5 contains the interview` — raw ID in prose
- `mouse (document_c818d3d5)` — ID shown alongside name
- `Deleted document_c818d3d5, document_a1b2c3d4` — listing IDs
- `The current project is project_2afef83d-f9fe-4493-96a0-2030ba3966b4` — exposing project ID
- `Deleted [mouse](file://document_c818d3d5)` — linking to a deleted resource

**Good:**
- `[mouse](file://document_c818d3d5) contains the interview` — ID hidden in link
- `Deleted mouse and notes` — for deleted items, use names without links (the resource no longer exists)
- `The interview transcript mentions...` — no ID needed when context is clear

When listing multiple items, make each name a link. When confirming actions, describe what changed using names and links, not IDs.
</reference-format>
