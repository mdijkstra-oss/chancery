
<shell-blocks-jq>
## Querying structured data with `blocks | jq`

`blocks` is the entry point for all structured data queries. It extracts JSON from fenced code blocks. `jq` processes the output. There are no standalone JSON files — `jq` is only useful piped from `blocks`.

### Block languages

- `json-attributes` — document metadata: tags, annotations
- `json-callout` — embedded blocks: codebook codes, references, notes

### Common patterns

Discover available codes:
```
blocks json-callout | jq "map(select(.type == \"codebook-code\")) | map({id, title})"
```

List all tags across files:
```
blocks json-attributes | jq "map(.tags // []) | flatten | unique"
```

Check existing annotations on a file:
```
blocks json-attributes doc.md | jq ".[0].annotations"
```

### Cross-file queries with `-p`

Without `-p`: returns `[{...block}, ...]` — block JSON only, no file context.

With `-p`: returns `[{file: "path", json: {...block}}, ...]` — use when you need to know which file a result came from.

```
blocks json-attributes -p | jq "map(select(.json.tags | contains([\"interview\"]))) | map(.file)"
```

### One call, filter with jq

Don't call `blocks` per file when querying multiple files. Call once, filter with jq:

```
# Wrong
blocks json-attributes file1.md | jq ...
blocks json-attributes file2.md | jq ...

# Right
blocks json-attributes -p | jq "map(select(.file | startswith(\"interview\"))) | ..."
```
</shell-blocks-jq>
