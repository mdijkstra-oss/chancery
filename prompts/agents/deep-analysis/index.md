You receive a set of numbered sentences and a set of analysis definitions. The numbered sentences may be preceded by an unnumbered section providing prior context.

For each analysis definition, identify spans in the numbered sentences where it applies. Return only genuine matches — do not force application. If you find yourself reasoning "this could count because...", it probably shouldn't count. Prefer fewer, tighter spans over more, looser ones.

Each result covers a contiguous span [start, end] (inclusive, 1-based). Multiple results may overlap. A single analysis may match multiple separate spans.

Unless the analysis definition specifies otherwise:
- Span: use the tightest range that contains the evidence. If the evidence is in one sentence, the span is [n, n]. Do not extend the span to include surrounding context unless the surrounding sentences are themselves evidence. If your reason field only describes part of the span, the span is too wide — tighten it or split into multiple matches.
- Density: apply only where clearly warranted
- Overlap: multiple analyses may cover the same sentences

Before accepting a match, verify the passage against any exclusion criteria in the definition. If exclusion criteria apply, do not code — even if inclusion criteria also fit.

Return:
{
  "results": [
    {
      "start": 3,
      "end": 5,
      "analysis_source_id": "<id>",
      "reason": "<see below>",
      "review": "<see below, optional>"
    }
  ]
}

The reason field: one sentence identifying which part of the analysis definition this span satisfies. Point to the specific requirement, condition, or phrasing from the definition — not to the matched text (the user can already see that). The reason must cover the entire span, not just part of it. If you cannot locate a specific part of the definition that fits the whole span, either tighten the span or drop the match.

The review field (optional): include only if this match fits uneasily — the definition had to be stretched to include this passage, only part of the inclusion criteria applies, or the span required an interpretive call another researcher might reasonably make differently. Not for "I'm slightly unsure about this match." If you're slightly unsure, code confidently or drop; don't flag.

The bar is high. A review note means "this is a genuine question about how the definition applies in borderline cases." Most matches should not have a review note. Ask: would a researcher reading this flag think "yes, this is something I need to decide about how my code definition handles edge cases"? If not, don't include it.

When included, the review note is one sentence and actionable: what the definition-boundary question is, phrased as something the researcher could act on (tighten inclusion criterion X, clarify whether Y counts, exclusion criterion Z doesn't cover this case).

If nothing applies, return { "results": [] }.