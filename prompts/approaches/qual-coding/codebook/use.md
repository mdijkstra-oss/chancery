---
description: The researcher wants to work on their codebook — creating it, reshaping it, splitting codes, or checking whether it holds up. The codebook is the object of attention.
---

## Constraints

Codebook files use the workspace's codebook-code entry format (json-callout blocks).
This is not a style preference — the UI, annotation engine, and coding tools
depend on it. Every codebook task must produce or maintain this format.

## Quality checks (surface to user, do not auto-fix)

Boundary clarity
Codes without clear "apply when / do not apply when" criteria — flag which ones.

Near-duplicates
Codes that overlap in scope — show the pair and let user decide.

Compound codes
Codes that bundle multiple concepts into one entry limit cross-dimensional queries.
This system applies multiple codes to the same text effortlessly, so there is
no reason to combine concepts. Recommend decomposition using an example
from their actual codebook, not a generic one.

Framework alignment
If user preferences or conversation specify an analytical framework,
check whether the codebook matches. A grounded theory project with a
rigid 40-code deductive codebook is a tension worth flagging.

## Structural suggestions (recommend, don't block)

If the codebook exceeds ~8 ungrouped codes, suggest organizing across files
by thematic group — improves both human navigation and per-group coding passes.

Do not keep old location where codes were, no unnecessary duplication. On splitting old file becomes index page for separated content

If the codebook is incomplete for the task at hand (e.g., coding a transcript
about vaccination but no vaccine-related codes exist), note the gap.