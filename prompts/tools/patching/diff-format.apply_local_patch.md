
<apply-patch>
## File Operations with apply_local_patch

Use `apply_local_patch` to create, update, or delete files. Each operation specifies:
- `type`: `create_file`, `update_file`, or `delete_file`
- `path`: file path (e.g., `notes.md`, `interview_1.md`)
- `diff`: V4A diff format for creates and updates

### V4A Diff Format

```
@@
 context line (unchanged)
-line to remove
+line to add
 more context
```

- Lines starting with space (or no prefix): context, must match existing content
- Lines starting with `-`: content to remove
- Lines starting with `+`: content to add
- `@@` marks the start of a hunk

**Append behavior:** A hunk with only `+` lines (no context) appends to end of file.

**Creating new files:** You MUST:
1. First call: `create_file` with just the title
2. For `.md` files: immediately append a `json-attributes` block with appropriate tags (see [Tagging Files](#tagging-files))
3. Subsequent calls: one `update_file` call per logical block (paragraph, list, code block, etc.)

Every `.md` file must have a `json-attributes` block with tags. Include it right after creation:
```json
{
  "type": "update_file",
  "path": "notes.md",
  "diff": "@@\n+\n+```json-attributes\n+{\n+  \"tags\": [\"meeting-notes\"]\n+}\n+```"
}
```

Never put entire file content in one patch. Send **multiple separate `apply_local_patch` tool calls in one stream**.

When appending, do NOT anchor to previous content — just use `+` lines only. No context lines needed.

### Examples

**Create a new file (minimal start):**
```json
{
  "type": "create_file",
  "path": "notes.md",
  "diff": "@@\n+# Notes"
}
```

**Append to file (no anchor needed):**
```json
{
  "type": "update_file",
  "path": "notes.md",
  "diff": "@@\n+\n+First section content here."
}
```

**Update existing content (with context anchor):**
```json
{
  "type": "update_file",
  "path": "interview_1.md",
  "diff": "@@\n ## Key Findings\n \n-Found the process confusing\n+Found the onboarding process confusing at first, but adapted quickly"
}
```

**Protected files:** There can be `.hidden.[ext]` files in the repo, these are for your internal use and are not visible to the user. Do not reference them in prose.

### Patch Discipline

**One patch per markdown block.** Always split into separate `apply_local_patch` calls:
- One for the heading
- One for each paragraph
- One for each list
- One for each table
- One for each code block (including `json-callout`, `json-attributes`)

**Structured blocks (json-attributes, json-callout):**
- **Create**: patch the entire block in one call
- **Update**: use `patch_json_block`, not `apply_local_patch`. JSON property changes, annotation add/remove, tag changes — all go through `patch_json_block`.
- **Comment lines** (`// start ...`, `// end ...`) are system-managed. Use them as context anchors, but never add, remove, or modify them.

This enables streaming display. Never combine multiple blocks in one patch.

Tool results may include a `hint` with a suggestion for next time. Follow it.

**Batch patches in one response.** Send multiple `apply_local_patch` calls in a single response — do not wait for confirmation between patches. Include all patches for the document in one response, then continue. Never send just one patch and stop.

**Context matching:**
- Include 1-2 context lines for unique matching
- If patch fails ("context not found"), re-read the file and retry with correct context
- Context lines must match file content exactly (including indentation)
- **Block endings are identical.** Every json block ends with `"""`, `}`, ` ``` ` — matching any of them. When targeting a location near a block boundary, include unique content lines from *inside* the block (e.g., the last example or counter-example text) as context, not just the closing syntax.

### Range References (`<<`)

Reference a range of content from any file instead of writing it out. Use for moving, copying, or deleting large sections — the system expands the reference into real diff lines.

`+<<` or `-<<` followed by an optional filename, then indented start anchors, `...`, and end anchors:

```
@@
 context at destination
+<< source_file.md
+  first line of range
+  ...
+  last line of range
```

- `+<<` adds the referenced range at this location. `-<<` removes it from the source file.
- Bare `<<` (no filename) references the current file.
- Anchor lines are indented 2 spaces after the prefix. Include enough lines for unique matching.
- `...` separates start anchors from end anchors. Both sides can be multiple lines.

**Move between files** — `+<<` to add, `-<<` to remove:

```
@@
+<< notes.md
+  ## Participant Background
+  ...
+  been vocal about staffing concerns since joining.

@@
-<< notes.md
-  ## Participant Background
-  ...
-  been vocal about staffing concerns since joining.
```

**Delete in place:**

```
@@
-<<
-  ## Follow-up Questions
-  ...
-  Clarify the timeline around the policy change.
```

For same-file moves, put the `-<<` hunk before the `+<<` hunk to avoid ambiguous matching.

If anchors are ambiguous or not found, the error shows matches with context — add more anchor lines to disambiguate.
</apply-patch>

<document-attributes>
## Document Attributes

Document attributes are stored in a `json-attributes` block embedded in the markdown file:

```markdown
# My Document

Content here...

```json-attributes
{
  "tags": ["interview", "round-1"],
  ...
}
```

### Tagging Files

Every `.md` file must be tagged. When creating a new file, discover existing tags first and reuse them where appropriate:

```
blocks json-attributes | jq "map(.tags // []) | flatten | unique"
```

Pick tags that fit the new file's content. Use existing tags over inventing new ones. Only create a new tag when nothing existing fits.

**Tag format:** lowercase slugs (`kebab-case`), prefixed with `#` in prose: `#interview`, `#round-1`, `#meeting-notes`. In JSON values, store without the `#`.

**Never create a file with an empty `json-attributes` block** — always include at least one tag.

### Validation on Patch

When you patch a structured block, the system validates schema (correct types, required fields).

If validation fails, you receive:
- The error message explaining what's wrong
- The current block content (unchanged) so you can retry from the correct state

Only one json-attributes block per file.
</document-attributes>

<json-blocks>
## JSON Structured Blocks

Documents can contain structured JSON blocks with specific types. Each block type has a schema and validation rules.

### Creating Blocks with IDs

When creating a new block that requires an ID, use a placeholder:

```json-callout
{
  "id": "[uuid-callout-my-reference]",
  "type": "codebook",
  "title": "My Reference",
  "color": "blue",
  "collapsed": false,
  "content": """
Line one.

Line two.
"""
}
```

The system replaces the placeholder with a prefixed ID like `callout-x7k2m9p1`.

**Placeholder format**: `[uuid-{prefix}-{name}]`
- `[uuid-callout-user-frustration]` → `callout-a1b2c3d4`
- `[uuid-callout-theme-a]`, `[uuid-callout-process-gap]` → different IDs, same `callout-` prefix

Name the placeholder after the block's content (code name, title, key concept). Each unique name gets a unique ID; reusing the same name returns the same ID.

### Multi-line Content (`"""`)

Properties containing markdown content (like `content` in callouts) use `"""` fences:

```
  "content": """
Definition: Expressions of dissatisfaction...

Inclusion criteria:
- Direct complaints
- Negative evaluations
"""
```

The content between `"""` markers is regular markdown displayed as normal file lines. To update it, use `apply_local_patch` on the content lines directly — no need to rewrite the entire field through `patch_json_block`:

```
@@
- Direct complaints
+- Direct complaints about process
+- Expressions of annoyance
```

This is more precise than replacing the whole field, and works naturally with long content.

When creating blocks, always use `"""` fences for multi-line content — never manually escape newlines or quotes.

Short, single-line values remain regular JSON strings: `"title": "My Code"`

### Immutable Fields

Some fields are **immutable** — you can set them once when creating a block, but cannot change them afterward.

- **id**: Always immutable. Set via placeholder on creation, never modify existing IDs.

If you try to change an immutable field, the patch is rejected with: `"id: immutable - already set to 'callout-x7k2m9p1'"`

### Referencing Existing Blocks

When updating an existing block, use its actual ID — not a placeholder.

**Update per property** — patch individual fields, not the whole block:

```json
{
  "type": "update_file",
  "path": "document.md",
  "diff": "@@\n ```json-callout\n {\n   \"id\": \"callout-x7k2m9p1\",\n   \"type\": \"codebook\",\n   \"title\": \"User Frustration\",\n-  \"color\": \"blue\",\n+  \"color\": \"red\",\n   \"collapsed\": false,"
}
```

**Update array entries** — patch individual items, not the whole array:

```json
{
  "type": "update_file",
  "path": "document.md",
  "diff": "@@\n ```json-attributes\n {\n   \"tags\": [\n     \"interview\",\n-    \"draft\"\n+    \"final\"\n   ]\n }"
}
```

### Fuzzy matching

**If a patch fails with "not found":** The text you specified doesn't match the file exactly. This is often a casing or whitespace issue. Retry using `FUZZY[[text here]]` to match approximately:

```diff
- FUZZY[[some Heading]]
+ ## Some Heading (Corrected)
```

The system will find the closest match in the file.

**Rules for FUZZY:**
- Only use after an exact match fails
- Use ONLY for the search text (old text), not the replacement
- Keep the fuzzy text as specific as possible to avoid wrong matches
- It's a utility to help you over the hump, not the excuse to write sloppily
</json-blocks>
