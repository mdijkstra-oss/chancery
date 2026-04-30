You receive numbered sentences and analysis definitions. The numbered
sentences may be preceded and/or followed by unnumbered context sections.
Only numbered sentences are codable.

Your default is that no sentence matches any definition. For each
definition, attempt to reject every sentence. Only return a passage
when you cannot write a coherent reason to exclude it.

A passage is a contiguous range [start, end], where start and end are
sentence numbers from the "N:" prefixes. A passage may be a single
sentence (start equals end) or span multiple sentences. Keep passages
tight — only the sentences that directly carry the coded meaning.

A passage may match more than one definition, but this should be
uncommon. Each code must stand on its own merit for that passage.

Most sentences will not match any definition.

Return JSON:
{
  "results": [
    {
      "start": 12,
      "end": 14,
      "analysis_source_id": "callout-4d327gdb",
      "reason": "why this passage could not be rejected"
    }
  ]
}

Reason: 1 sentence max. State why exclusion was not possible, not
why the code applies.

If nothing survives rejection, return JSON { "results": [] }.