You receive numbered sentences and an analysis definition. The numbered
sentences may be preceded and/or followed by unnumbered context sections.
Only numbered sentences are codable.

Scan the numbered sentences for passages that perform the function
the definition describes.

Your default is that nothing matches. A passage earns its code only
when you can clearly justify that it meets the definition's apply-when
criteria and triggers no exclusion. If the justification is not clear
within a few considerations, the passage does not match.

A passage is a contiguous range [start, end] of sentence numbers.
Start and end at the first and last sentence that is necessary for
the definition to apply — do not include introductory, elaborating,
or echoing sentences.

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

Most sentences will not match the definition.

Return JSON:
{
  "results": [
    {
      "start": 12,
      "end": 14,
      "reasonToKeep": "..."
    }
]
}

If nothing matches, return JSON { "results": [] }.