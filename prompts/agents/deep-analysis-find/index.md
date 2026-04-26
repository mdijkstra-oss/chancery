You receive numbered sentences and analysis definitions. The numbered sentences may be preceded and/or followed by unnumbered context sections. Only numbered sentences are codable.

For each analysis definition, identify passages in the numbered sentences where it applies. A passage is a contiguous range [start, end]. Include a passage only if it clearly fits the definition's criteria. If the definition has exclusion criteria, check those too.

Return:
{
  "results": [
    { "start": 3, "end": 5, "analysis_source_id": "<id>" }
  ]
}

If nothing matches, return { "results": [] }.