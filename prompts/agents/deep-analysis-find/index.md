You receive target text organized in file-specific blocks. Each block is
wrapped in <target file="..." prefix="X"> tags. Sentences are numbered
with the file's prefix letter followed by a number (a1, a2, b1, b2, …).
Only numbered sentences are codable. Unnumbered context may appear
outside the target blocks.

Scan the numbered sentences for passages that perform the function
the definition describes.

Most sentences will not match the definition. Exclude passages where
you cannot connect the passage's function to the definition at all.
Include passages where you can articulate a plausible case that the
definition applies — a specific reason grounded in the definition,
not a vague topical overlap.

A passage is a contiguous range [start, end] of prefixed sentence numbers
within a single file. Passages cannot span across files or across chunk
separators (---) within a file. Start and end at the first and last
sentence that is necessary for the definition to apply — do not include
introductory, elaborating, or echoing sentences.

Matching rules:
- A sentence must perform the function the definition describes,
  not merely contain words or concepts the definition references.
- A sentence that talks ABOUT the topic of the definition without
  actually doing what the definition specifies is not a match.
- Every sentence in the range must contribute to performing
  the function. A sentence that serves a different purpose
  breaks the passage even if it is adjacent.
- If two passages overlap, keep only the one that most precisely
  performs the function.

Return JSON:
  {
    "results": [
      {
        "file": "filename.md",
        "start": "a3",
        "end": "a5",
        "reasonToKeep": "..."
      }
    ]
  }

If nothing matches, return JSON { "results": [] }.
