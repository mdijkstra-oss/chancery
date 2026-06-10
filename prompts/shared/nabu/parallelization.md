# Tool Call Parallelization

**Emit together** = multiple calls in one response, before waiting for results.
**Wait** = do not emit the next call until responses from previous calls are received.

---

## Rules

**id-dependency** — Never emit a call that uses an ID not yet received. Emit all calls that generate IDs together, wait, then emit all calls that use those IDs together.

**same-file-prose** — Multiple `apply_local_patch` on the same file may be emitted together only if each patch anchors on content already written and unaffected by the other patches. When building a new file, all patches go together — each appends to already-landed content. When editing an existing file, emit together only if anchors are in clearly separate regions. Within the same section, wait between patches.

**same-block-operations** — Multiple operations on the same json block belong in one call's `operations` array. Never emit two separate `patch_*` calls targeting the same block.

**default** — If none of the above apply, emit together.

---

## Tool reference

| Tool | Emit together? | Notes |
|------|---------------|-------|
| `apply_local_patch` | see same-file-prose | Prose anchor based — separate regions ok |
| `patch_*` `delete_*` `add_*` | ✅ across blocks/files | Same block → operations array |
| `run_local_shell` | ✅ if independent | Wait if next command needs previous result |
| `query` | ✅ if independent | Wait if next query needs previous result |
| `copy_file` `rename_file` `remove_file` | ✅ if independent | Not with other ops on same file |
| `start_planning` | ❌ always solo | Mode transition |
| `ask` | ❌ always solo | Blocks on user response |
| `search` | ❌ always solo | User-facing interaction |

---

## Scenarios

### Tag files — tag already exists

Emit all `patch_attributes` calls together. Wait.

```
patch_attributes(file1, set tags)
patch_attributes(file2, set tags)
patch_attributes(file3, set tags)  ...
```

### Tag files — tag may not exist

1. Emit query to check whether tag exists in settings. Wait.
2. If tag exists — go to step 3. If not: emit `patch_settings(add_tag)`. Wait.
3. Emit all `patch_attributes` calls together. Wait.

---

### Create file with blocks (callouts / charts)

Three phases — see id-dependency rule.

**Single file:**

1. Emit all prose patches together — `create_file` first, then each `update_file` anchors on previously written content. Wait.
2. Emit all `add_callout` / `add_chart` calls together. Wait for all IDs.
3. Emit all `patch_callout` / `patch_chart` calls together using the received IDs. Wait.

**Multiple files:**

1. For each file: emit all prose patches for that file together. Wait. Repeat per file.
2. Emit all `add_*` calls across all files together. Wait for all IDs.
3. Emit all `patch_*` calls across all files together. Wait.

---

### Annotate files (coding)

Different files — emit all `patch_annotations` calls together. Same file — batch all operations into one call's `operations` array.

```
patch_annotations(file1, [add_annotation, add_annotation, ...])
patch_annotations(file2, [add_annotation, ...])
patch_annotations(file3, [add_annotation, ...])
```

If an annotation references a code ID, verify the code exists first — same gate pattern as tag-may-not-exist above.

---

### Reads

Emit independent `query` and `run_local_shell` calls together. Wait if a later call depends on an earlier result.

---

### Bulk file operations

Emit all `rename_file`, `remove_file`, `copy_file` calls together as long as files are independent. Wait. Never pair rename or remove with another operation on the same file in the same response.