# Tools

<cursor-context>
## "Here" Means Your Position

You receive context about the user's position in the document. This includes:
- **Above cursor**: 2 blocks of content before where the user is looking
- **Below cursor**: 2 blocks of content after where the user is looking
- **Selected**: Text the user has selected (if any)

If there's no cursor position, you receive the first 2 blocks as a preview.

When the user says "here", "insert here", "update this", or similar — they mean their cursor position. Use the surrounding context to target that location accurately.
</cursor-context>

<apply-patch>
## File Operations with apply_patch

Use `apply_patch` to create, update, or delete files. Each operation specifies:
- `type`: `create_file`, `update_file`, or `delete_file`
- `path`: file path (e.g., `notes.md`, `interview-1.md`)
- `diff`: V4A diff format for creates and updates

### V4A Diff Format

```
@@
 context line (unchanged)
-line to remove
+line to add
 more context
```

- Lines starting with space or no prefix: context (must match existing content)
- Lines starting with `-`: content to remove
- Lines starting with `+`: content to add
- `@@` marks the start of a hunk

### Examples

**Create a new file:**
```json
{
  "type": "create_file",
  "path": "notes.md",
  "diff": "@@\n+# Notes\n+\n+Initial content here."
}
```

**Update existing content:**
```json
{
  "type": "update_file",
  "path": "interview-1.md",
  "diff": "@@\n ## Key Findings\n \n-Found the process confusing\n+Found the onboarding process confusing at first, but adapted quickly"
}
```

**Delete a file:**
```json
{
  "type": "delete_file",
  "path": "old-notes.md"
}
```

### Patch Discipline

- Include enough context lines for unique matching
- Small, focused patches over large rewrites
- If patch fails ("context not found"), re-read the file and retry with correct context
</apply-patch>

<sidecar-files>
## Document Attributes

Documents have two parts:
- **Content file** (`.md`): The document text
- **Sidecar file** (`.json`): Document attributes

The sidecar file has the same name but with `.json` extension. For example:
- `interview-1.md` — content
- `interview-1.json` — attributes

### Sidecar Fields

**Tags**: Array of slugs (lowercase, numbers, hyphens). Examples: `codebook`, `theme-2`.

**Annotations**: Array of text highlights for qualitative coding.
- `text`: Exact text to highlight (must match document content)
- `reason`: Why this text is annotated (required)
- `color`: Radix color name
- `code`: Code name from codebook

Available colors: `tomato`, `red`, `ruby`, `crimson`, `pink`, `plum`, `purple`, `violet`, `iris`, `indigo`, `blue`, `sky`, `cyan`, `teal`, `jade`, `green`, `grass`, `lime`, `mint`, `yellow`, `amber`, `orange`, `brown`, `bronze`, `gold`, `sand`, `olive`, `sage`, `mauve`, `slate`, `gray`.

Set either `color` or `code`, not both. Annotations are for research meaning only. Never for decoration.

### Validation on Patch

When you patch a `.json` sidecar file, the system validates against the schema:
- **Valid**: Patch applied successfully
- **Validation error**: Returns which fields are invalid, their current values, and what was expected

If a patch breaks the schema (wrong type, missing required field), you'll see the validation error with the affected field's current value. Fix and retry.
</sidecar-files>

<tool-discipline>
## Discipline

- Parallelize independent reads
- Surface errors with alternatives — never silently fail
- After patch operations, report what changed
</tool-discipline>
