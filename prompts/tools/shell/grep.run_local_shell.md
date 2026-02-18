
<shell-grep>
## grep discipline

### One call, all files

Never grep file-by-file. One call searches everything:

```
grep -n "term"           # all files
grep -n "term" prefix/   # scoped to prefix
grep -n -B1 -A1 "term"   # with context
```

### Counting occurrences vs lines

Users care about **how many times** something appears, not how many lines contain it.

```
# Wrong: counts lines containing "OMT" (a paragraph with 3 mentions = 1)
grep -c "OMT"

# Right: counts actual occurrences (a paragraph with 3 mentions = 3)
grep -o "OMT" | wc -l
```

Always use `grep -o pattern | wc -l` for counting. Report "X appears N times" — not "N lines contain X".

### Don't retry successful queries

If a command returned the data you need, use it. Don't re-run with different flags to "double-check."

`status: partial` means some commands in the batch failed — check individual outputs. Successful commands are still valid.

### When NOT to use grep

Grep finds **literal strings only**. Do not use it for concepts, emotions, themes, or semantic meaning.

You MUST NOT grep emotions or themes. Grepping synonyms one by one misses things and wastes calls.

**Don't grep-roulette concepts**: `grep healthcare`, `grep zorg`, `grep hospital`... is not a search strategy. If you don't know the exact strings, read the content section by section.
</shell-grep>
