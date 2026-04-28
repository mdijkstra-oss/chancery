You receive numbered sentences and analysis definitions. The numbered
sentences may be preceded and/or followed by unnumbered context sections.
Only numbered sentences are codable.

For each analysis definition, identify passages in the numbered sentences
where it applies. A passage is a contiguous range [start, end], where
start and end are sentence numbers from the "N:" prefixes. A passage may
be a single sentence (start equals end) or span multiple sentences.

Apply a definition when its inclusion criteria fit, even where the match
is not perfectly explicit. If an exclusion criterion clearly applies, do
not return the passage.

Multiple codes can apply to the same or overlapping passages. Return one
entry per (passage, code) pair — two entries with identical start/end
and different analysis_source_id is expected and valid when a passage
fits more than one definition.

For analysis_source_id, use the identifier given with each definition.

Return:
{
  "results": [
    { "start": 12, "end": 14, "analysis_source_id": "callout-4d327gdb" }
  ]
}

If nothing matches, return { "results": [] }.

Example
-------
Definitions:
<analysis id="callout-example-a">
  # Anchoring appeal
  Speaker grounds a position by invoking an external authority or shared norm.
  Inclusion: the position is justified by appeal to law, tradition, or shared identity.
  Exclusion: bare assertions without such grounding.
</analysis>
<analysis id="callout-example-b">
  # Future commitment
  Speaker pledges a specific future action.
  Inclusion: a definite, unhedged statement about what will be done.
  Exclusion: hopes, possibilities, or hedged intentions.
</analysis>

Numbered sentences:
12: As the constitution requires, we will publish the report next week.
13: This builds on the same principle of public accountability.
14: We hope further reforms follow.

Output:
{
  "results": [
    { "start": 12, "end": 13, "analysis_source_id": "callout-example-a" },
    { "start": 12, "end": 12, "analysis_source_id": "callout-example-b" }
  ]
}