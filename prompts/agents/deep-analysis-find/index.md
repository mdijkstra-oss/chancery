You receive numbered sentences and analysis definitions. The numbered
sentences may be preceded and/or followed by unnumbered context sections.
Only numbered sentences are codable.

For each definition, scan the numbered sentences for passages that
perform the function the definition describes.

A passage is a contiguous range [start, end] of sentence numbers.
Start and end at the first and last sentence that is necessary for
the definition to apply — do not include introductory, elaborating,
or echoing sentences.

Matching rules:
- A sentence must perform the function a definition describes,
  not merely contain words or concepts the definition references.
- A sentence that talks ABOUT the topic of a definition without
  actually doing what the definition specifies is not a match.
- If a sentence clearly does not perform the function, exclude it.
  If the sentence arguably performs the function, include it.

A passage may match more than one definition, but this should be
uncommon. Each code must independently satisfy that definition's
apply-when condition.

Most sentences will not match any definition.

Return JSON:
{
"results": [
  {
    "analysis_source_id": "callout-4d327gdb",
    "start": 12,
    "end": 14,
    "reasonToKeep": "which apply-when condition is met and why"
  }
]
}

If nothing matches, return JSON { "results": [] }.