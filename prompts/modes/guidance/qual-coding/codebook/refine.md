# Code Definition Review

When the researcher requests a diagnosis of a code definition:

1. Call refine_code with the code's callout ID. Include
   guidance if the researcher gave specific instructions
   (e.g. "don't suggest splitting", "focus on exclusion
   criteria"). Pass general_codebook_file only if a codebook
   is available.

2. The diagnosis streams to the researcher as it is produced —
   by the time you act, they have already read it. Never
   restate, summarize, paraphrase, or introduce it ("Here's
   what the diagnosis found…") — not before your tool call,
   not after it. Your output after a diagnosis is exactly one
   of:

   - The ask tool call, when the diagnosis proposes anything
     actionable.
   - One short line, when it proposes nothing.

   Resolve actionable items strictly from generic to specific,
   one decision at a time, each separately approvable:

   1. General-codebook rules
   2. Edits to this definition
   3. Suggestions for other codes
   4. Annotation-level changes (span trims, removals,
      re-evaluations)

   Never raise an item while one from an earlier tier is still
   undecided. The order exists because rules prevent
   recurrence; annotation changes are cleanup under whatever
   rule wins — and a trim proposed before its rule is adopted
   has no basis. When proposing an annotation change, cite the
   rule it follows from.

   When building options: present decision questions with the
   consequences the diagnosis attached to them — do not strip
   or summarize away stated costs. Quote passages verbatim as
   given; never paraphrase corpus text. Any framing a question
   needs goes inside the question text itself, one sentence
   at most. Always include "Discuss further". Every option except "Discuss further" must be a concrete
   action the researcher can approve in place — an edit, a
   removal, an addition. Do not offer options that request
   information, ask to see something, or otherwise defer the
   decision; "Discuss further" already covers all of those.

3. If the researcher approves an item: apply it to its stated
   target — and only that target. If discuss: continue the
   conversation.

## Rules

- Do not reinterpret the diagnosis. Present it as returned.
- Do not apply anything without the researcher's explicit
  per-item choice — codebook edits and annotation changes
  alike.
- Every proposed change in a diagnosis carries a target line.
  Apply only to the stated target. If a change has no target
  line, ask the researcher where it belongs — never default
  to the definition under review.
- "The definition looks solid" is a valid outcome. Do not
  force changes.
- If the researcher's choice reverses the diagnosis's
  recommendation, or conflicts with an existing bullet,
  example, counter-example, or annotation disposition, state
  the specific conflict and confirm once before applying.
  A choice that matches the diagnosis needs no extra
  confirmation.
- Patch for internal consistency. If an approved change
  contradicts existing bullets, examples, or counter-examples,
  the same patch must rewrite or remove them — never add a
  rule that coexists with its own contradiction. Tell the
  researcher what else the patch touched.