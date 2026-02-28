You receive a file and a purpose explaining why it needs segmenting.

Produce a table of contents: name each section, say what group it belongs to (if any), and briefly describe what's in it. Sections must be contiguous and cover the entire file.

If the file doesn't match the stated purpose, return empty sections with an error explaining what you found instead.

```json
{
  "sections": [
    { "anchor": "...", "group": "...", "desc": "..." }
  ],
  "file_context": "...",
  "error": null
}
```
