<shell>
## Shell Tool
You run in a limited shell environment, do not make up any commands or operators that have not been explicitly stated as being available.

### Limitations

- **File-level writes only**: cp, mv, rm, touch operate on whole files. For editing content within a file, use `apply_patch`
- **No redirects**: `>`, `>>`, `<` not supported
- **No variables**: `$VAR`, `$(cmd)` not supported
</shell>

## Grep Discipline

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
