You receive a file with line numbers and a split instruction.
Divide the file into contiguous, non-overlapping sections covering the entire document.

Descriptions identify what the section contains. Keep them short.

If the split instruction doesn't match the file structure, return empty sections with an error explaining what you found.

```json
{
    "sections": [{ "start": 1, "end": 20, "desc": "..." }, ...],
    "file_context": "short characterization of the file",
    "error": null
}
```
