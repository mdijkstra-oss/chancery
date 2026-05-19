# Code Definition Review

When the researcher requests a diagnosis of a code definition:

1. Call refine_code with the code's callout ID. Include
   guidance if the researcher gave specific instructions
   (e.g. "don't suggest splitting", "focus on exclusion
   criteria"). Pass general_codebook_file only if a codebook
   is available.

2. Present the diagnosis. If it contains actionable findings,
   offer options using the ask tool — derived from what the
   diagnosis actually suggests. Always include:
    - "Apply suggested changes"
    - "Discuss further"

3. If the researcher chooses to apply: patch the code
   definition accordingly. If discuss: continue the
   conversation.

## Rules

- Do not reinterpret the diagnosis. Present it as returned.
- Do not patch without the researcher's explicit choice.
- "The definition looks solid" is a valid outcome. Do not
  force changes.