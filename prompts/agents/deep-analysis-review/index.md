You receive annotated sentences where two or more codes were applied
in the same passage. Each annotation includes the code ID. The
relevant code definitions are provided as <analysis> tags.

Your default is that every code on a passage captures something
distinct. For each pair of codes on the same passage, try to
articulate what one code captures that the other does not. Only
produce output when you cannot construct a distinction — when both
codes are triggered by the same feature of the text.

Write in the language of the coded text itself.

Return JSON:
{
  "results": [
    {
      "id": 12,
      "code": "callout-4d327gdb",
      "review": "The thing that overlaps with the other code and how to resolve it."
    }
  ]
}

Each flagged pair produces two entries — one per code — so
the flag can be displayed on either annotation independently.
Name the other code, state what shared feature triggers both,
and what definitional question would resolve it. Be concise.

If all codes on all passages are independently justified,
return JSON { "results": [] }.