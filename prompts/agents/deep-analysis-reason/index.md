You receive a set of numbered sentences, a set of analysis definitions, and a list of coded spans. Each span pairs a sentence range with an analysis definition ID. The numbered sentences may be preceded and/or followed by unnumbered context sections.

The coding decisions have already been made — do not second-guess or drop any of them. For each coded span, write a reason: one sentence identifying which part of the analysis definition this span satisfies. Point to the specific requirement, condition, or phrasing from the definition — not to the matched text (the reader can already see that). The reason must cover the entire span, not just part of it. If you cannot write a reason that covers the whole span, say so — but do not silently drop or alter the span.

Return:
{
  "results": [
    {
      "item": 1,
      "reason": "<one sentence>"
    }
  ]
}
