<shell>
## Shell Tool

You have access to a read-only virtual shell for exploring documents. Use it to search, filter, and inspect file contents.

### Available Commands

**cat** — Print file contents
```
cat [-n] [-o offset] [-l limit] <file>
```
- `-n`: Number output lines
- `-o N`: Start at line N (default 1)
- `-l N`: Show at most N lines

**ls** — List files
```
ls [-l] [prefix]
```
- `-l`: Long format with file sizes

**grep** — Search for patterns
```
grep [-n] [-i] <pattern> [path]
```
- `-n`: Show line numbers
- `-i`: Case insensitive

**find** — Find files by prefix
```
find [prefix]
```

**wc** — Count lines, words, characters
```
wc [-l] [-w] [-c] <file>
```
- `-l`: Lines only
- `-w`: Words only
- `-c`: Characters only

### Operators

- `|` — Pipe output to next command
- `&&` or `;` — Chain commands

### Examples

```bash
# Search for "pain point" in all interview files
grep -i "pain point" /interviews

# Show first 20 lines of a file
cat -l 20 /notes.md

# Count how many times "confused" appears
grep confused /interview-1.md | wc -l

# List all files with sizes
ls -l /
```

### Limitations

- **Read-only**: No file creation or modification (use `apply_patch` for that)
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
