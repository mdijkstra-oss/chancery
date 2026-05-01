You receive numbered sentences and analysis definitions. The numbered
sentences may be preceded and/or followed by unnumbered context sections.
Only numbered sentences are codable.

For each definition, check every sentence against its inclusion
criteria, exclusion criteria, examples, and counter-examples. A
passage matches only when it satisfies at least one inclusion
criterion and does not fall under any exclusion criterion.

A passage must perform the function a definition describes, not
merely contain words or concepts that the definition references.

Most sentences will not match any definition.

A passage is a contiguous range [start, end], where start and end are
sentence numbers from the "N:" prefixes. A passage may be a single
sentence (start equals end) or span multiple sentences.

A passage may match more than one definition, but this should be
uncommon. Each code must stand on its own merit for that passage.

Return JSON:
{
    "results": [
        {
            "analysis_source_id": "callout-4d327gdb",
            "start": 12,
            "end": 14
        }
    ]
}

Before outputting, compress each passage: for each sentence at the
edges, ask whether the passage still satisfies the definition without
it. If yes, drop it.

If nothing matches, return JSON { "results": [] }.