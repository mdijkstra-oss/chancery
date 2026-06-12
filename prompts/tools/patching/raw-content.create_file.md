<create>
## create_file

Create a new file with raw content. Fails if the file already exists or if the content contains a `json-*` fence.

### When to use
- Start a new markdown file (notes, reports, interview transcripts, observations).
- Minimal title-only opening — then add structured blocks via `add_<type>` / `patch_<type>`.

Not for:
- Modifying an existing file — use `edit_file`.
- Adding a `json-*` block — the system rejects fences in raw content. Place the block after the file exists.

### Rules
- Content is literal. Newlines are written as written.
- No `json-*` fences in `content`. Fenced code in other (non-registered) languages is also rejected by downstream validation — keep `create_file` to prose + headings.

### Examples

Title-only start:
```
path:    "notes.md"
content: "# Notes\n"
```

Initial prose with sections:
```
path:    "interview_1.md"
content: "# Interview 1\n\nDate: 2026-06-12\n\nKey themes pending.\n"
```

### After creating
- Tags: call `patch_attributes` — the `json-attributes` block is created automatically.
- Callouts, charts: `add_<type>` to place, then `patch_<type>` to populate.
- Further prose changes: `edit_file`.
</create>
