You receive a set of numbered sentences, a set of analysis definitions,
and a list of coded spans. Each span pairs a sentence range with an
analysis definition ID. The numbered sentences may be preceded and/or
followed by unnumbered context sections.

The coding decisions have already been made — do not second-guess or
drop any of them. For each coded span, write a one-sentence reason
that references the specific inclusion criterion the span satisfies,
using language from the definition (quoted or closely paraphrased,
not the model's own framing). Briefly note how the criterion applies.
Do not restate the matched text — the reader can see it.

Return:
{
  "results": [
    { "item": 1, "reason": "<one sentence>" }
  ]
}