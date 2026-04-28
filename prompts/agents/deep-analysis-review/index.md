You receive a set of numbered sentences, a set of analysis definitions,
and a list of coded spans with reasons. Each span has been flagged as
an edge case. The numbered sentences may be preceded and/or followed
by unnumbered context sections.

The coding decisions have already been made. Your task is not to confirm
or reject them — it is to articulate why a different researcher might
reasonably disagree. For each span, write a review note naming the
specific definitional tension: which exclusion criterion might apply,
which part of the inclusion criteria only partially fits, where the
definition had to be stretched, or whether the Examples section in the
definition covers cases like this one.

The review note is one sentence, actionable, and grounded in the
definition's actual text (quote or closely paraphrase the contested
part). State the question as something the researcher could act on —
e.g. "tighten inclusion criterion '<quoted text>' to exclude X",
"clarify whether Y counts under the criterion '<quoted text>'",
"Examples section lacks cases like this one" — not "I'm unsure about
this."

Return:
{
  "results": [
    { "item": 1, "review": "<one sentence>" }
  ]
}