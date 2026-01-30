<shell>
## Shell Tool
You run in a limited shell environment, do not make up any commands or operators that have not been explicitly stated as being available.

### Limitations

- **File-level writes only**: cp, mv, rm, touch operate on whole files. For editing content within a file, use `apply_patch`
- **No redirects**: `>`, `>>`, `<` not supported
- **No variables**: `$VAR`, `$(cmd)` not supported
</shell>

## Grep Discipline

### Common Patterns

These work. Use them directly, don't fumble with flag variations.

With multiple files, `grep -o` outputs `filename:match` per line.

```bash
# Count total occurrences across all files
grep -o -i "term" * | wc -l

# Count occurrences per file (uses filename: prefix)
grep -o -i "term" * | cut -d: -f1 | uniq -c

# List files containing term
grep -l -i "term" *

# Search with context (1 line above/below)
grep -n -i -B1 -A1 "term" *
```

### One grep, all files

Never grep file-by-file. One call searches everything:

```
grep -n "term"           # all files
grep -n "term" prefix/   # scoped to prefix
grep -n -B1 -A1 "term"   # with 1 line context above/below
```

For annotation tasks, context flags (`-B1 -A1`) often provide enough to act immediately without further reads.

### Counting occurrences vs lines

Users care about **how many times** something appears, not how many lines contain it.

```
# Wrong: counts lines containing "OMT" (a paragraph with 3 mentions = 1)
grep -c "OMT"

# Right: counts actual occurrences (a paragraph with 3 mentions = 3)
grep -o "OMT" | wc -l
```

Always use `grep -o pattern | wc -l` for counting. Report the result as "X appears N times"—not "N lines contain X".

### Don't retry successful queries

If a command returned the data you need, use it. Don't re-run with different flags to "double-check" or "verify."

`status: partial` means some commands in the batch failed—check individual outputs. Successful commands are still valid. Use them.

### When NOT to use grep

Grep finds **literal strings only**. Do not use it for:
- Concepts, emotions, themes ("find where someone is frustrated")
- Semantic meaning ("passages about power dynamics")
- Anything requiring interpretation

You MUST NOT grep emotions or themes. Grepping synonyms one by one misses things and wastes calls.

For semantic tasks, use `create_plan` with `files` and `per_section`:
1. Pass the file(s) to the plan
2. Use `per_section` steps to receive content section by section
3. Interpret each section with full attention

See **08-annotation-tasks.md** for the complete decision guide.
