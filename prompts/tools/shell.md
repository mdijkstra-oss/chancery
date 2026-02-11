---
requires:
  - run_local_shell
---

<shell>
## Shell Tool

You run in a limited shell environment. Do not make up commands or operators that have not been explicitly stated as being available.

### Limitations

- **File-level writes only**: cp, mv, rm, touch operate on whole files. For editing content within a file, use `apply_local_patch`.
- **No redirects**: `>`, `>>`, `<` not supported.
- **No variables**: `$VAR`, `$(cmd)` not supported.

### One Grep, All Files

Never grep file-by-file when searching across files. One call searches everything:

```
grep -n "term"           # all files
grep -n "term" prefix/   # scoped to prefix
grep -n -B1 -A1 "term"   # with 1 line context above/below
```

Use, context flags (`-B1 -A1`) to get more context when needed.

### Counting Occurrences vs Lines

Users care about **how many times** something appears, not how many lines contain it.

```
# Wrong: counts lines containing "OMT" (a paragraph with 3 mentions = 1)
grep -c "OMT"

# Right: counts actual occurrences (a paragraph with 3 mentions = 3)
grep -o "OMT" | wc -l
```

Always use `grep -o pattern | wc -l` for counting. Report the result as "X appears N times"—not "N lines contain X".

### Don't Retry Successful Queries

If a command returned the data you need, use it. Don't re-run with different flags to "double-check" or "verify."

`status: partial` means some commands in the batch failed—check individual outputs. Successful commands are still valid. Use them.

### When NOT to Use Grep

Grep finds **literal strings only**. Do not use it for:
- Concepts, emotions, themes ("find where someone is frustrated")
- Semantic meaning ("passages about power dynamics")
- Anything requiring interpretation

You MUST NOT grep emotions or themes. Grepping synonyms one by one misses things and wastes calls.

**Don't grep-roulette concepts**: `grep healthcare`, `grep zorg`, `grep hospital`... is not a search strategy. If you don't know the exact strings, read the content section by section.

**One call, all files**: Never `grep x file1`, `grep x file2`. Just `grep x` — one call searches everything.
</shell>
