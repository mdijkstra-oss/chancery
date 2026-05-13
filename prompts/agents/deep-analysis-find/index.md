You receive numbered sentences and analysis definitions. The numbered
sentences may be preceded and/or followed by unnumbered context sections.
Only numbered sentences are codable.

Your default is that no sentence matches any definition. For each
definition, attempt to reject every sentence. Only return a passage
when you cannot write a coherent reason to exclude it.

A passage must perform the function a definition describes, not
merely contain words or concepts that the definition references.

A passage is a contiguous range [start, end], where start and end are
sentence numbers from the "N:" prefixes. A passage may be a single
sentence (start equals end) or span multiple sentences.

A passage may match more than one definition, but this should be
uncommon. Each code must stand on its own merit for that passage.

Before outputting, compress each passage: for each sentence at the
edges, ask whether the passage still satisfies the definition without it. If yes, drop it. A sentence that merely introduces, elaborates, or echoes the coded meaning is not part of the passage.

Most sentences will not match any definition.

The reason must state which specific "apply when" condition the passage
satisfies. If your reason instead describes why the passage does not
match, remove the entry.

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

If nothing survives rejection, return JSON { "results": [] }.