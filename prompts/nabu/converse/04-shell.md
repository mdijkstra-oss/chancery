<shell>
## Shell Tool

Virtual shell for exploring and managing documents.

### Read Commands

**cat** — Print file contents
```
cat [-n] [-o offset] [-l limit] <file>
```

**head** / **tail** — First/last lines
```
head [-n N] <file>
tail [-n N] <file>
```

**ls** — List files
```
ls [-l] [prefix]
```

**grep** — Search for patterns
```
grep [-n] [-i] <pattern> [path]
```

**find** — Find files by name
```
find [-name pattern] [prefix]
```

**wc** — Count lines/words/chars
```
wc [-l] [-w] [-c] <file>
```

### Text Processing

**cut** — Extract fields
```
cut -d<delim> -f<fields>
```

**sort** — Sort lines
```
sort [-u] [-r] [-n]
```

**uniq** — Remove adjacent duplicates
```
uniq [-c]
```

**tr** — Translate/delete characters
```
tr [-d] [-s] <set1> [set2]
```

**sed** — Substitute patterns (s/// only)
```
sed 's/pattern/replacement/[g]'
```

**echo** — Print text
```
echo [text...]
```

### File Operations

**cp** — Copy file
```
cp <source> <dest>
```

**mv** — Rename file
```
mv <source> <dest>
```

**rm** — Delete file
```
rm <file>...
```

**touch** — Create empty file
```
touch <file>
```

### Operators

- `|` — Pipe output
- `&&` — Run if previous succeeded
- `||` — Run if previous failed
- `;` — Run regardless

### Limitations

- **File-level writes only**: cp, mv, rm, touch operate on whole files. For editing content within a file, use `apply_patch`
- **No redirects**: `>`, `>>`, `<` not supported
- **No variables**: `$VAR`, `$(cmd)` not supported
</shell>

## When NOT to use grep

Grep finds **literal strings only**. Do not use it for:
- Concepts, emotions, themes ("find where someone is frustrated")
- Semantic meaning ("passages about power dynamics")
- Anything requiring interpretation

For these, use `create_plan` with `files` and `per_section` (explore first if needed but for full file interpretation use plans):
1. Pass the file(s) to the plan
2. Use `per_section` steps to receive content section by section
3. Use your understanding to identify relevant passages

Grepping synonyms one by one will miss things and waste time. Read the content and interpret it.
