<coding>
# Working with annotations

## Plan structure for coding

When coding a file section by section, group codes by theme within each section. A section step becomes a nested group — one substep per code cluster, not one monolithic "code this section" step. Group by the codebook's own structure: if codes fall into families (economic, social, procedural), each family is a substep. This keeps each step focused and makes `internal` carry-forward manageable.

When the codebook grows large, suggest splitting codes across multiple files by theme or family — codes are discovered across all files, so placement is organizational, not functional. A codebook with 30+ codes in one file becomes harder to navigate than three focused files (`economic_factors.md`, `social_dynamics.md`, `process_issues.md`). Tag all split files with a shared tag (e.g., `#codebook`) to keep the group discoverable — no index files pointing to other docs.

## Multi-coding

If a passage genuinely fits multiple codes, add multiple annotations — one per code. Each annotation stands on its own with its own reason grounding the coding decision.

## Review

The `review` field flags any annotation for human attention. It works on both code-linked and color-only annotations. The guiding principle: flag when discussing the annotation might update the codebook.

**Definition stretch** — the code is the closest match but the definition doesn't fully cover what's in the text. Explain where the definition is being stretched.

**Boundary friction** — the passage could reasonably go to either of two codes. Code it as the stronger fit, flag explaining the tension and the competing code. Repeated friction between the same pair signals a boundary problem.

**Codebook gap** — the passage is analytically relevant but no existing code fits. Highlight with color, flag explaining what the passage captures that the codebook misses.

**In vivo candidate** — the participant's exact words capture something no code covers. Highlight with color, flag surfacing the language as a potential new code.

**Disconfirming evidence** — the passage challenges an emerging pattern or contradicts how a code has been applied elsewhere. Code it, flag explaining the tension.

**Emerging pattern** — you notice something across annotations that the codebook doesn't account for. Flag a representative instance explaining the pattern.

When a passage presents genuine codebook-relevant ambiguity, preserve that ambiguity at capture time with `review`, even if work can continue. Discussion thresholds apply later; preservation happens now. An ambiguity that recurs is a pattern — if the first instance wasn't flagged, the pattern is invisible.

An annotation without `review` is confident. An annotation with `review` is flagged. Do not use review as a hedge for weak decisions. If a code doesn't fit, don't force it — use color and flag the gap instead.

## Review cadence

After each section, triage your own flags. The question per flag: would this change how the next section is coded? If yes — surface it to the researcher before continuing. If no — it stays flagged for the batch review. A flag that recurs (same boundary friction, same gap, same definition stretch) is a pattern, and patterns change coding. A flag that appears once is a candidate for the batch review.

After all coding is done, suggest a codebook review as a follow-up — a dedicated pass across files to collect remaining flags, group them by code, and determine which need codebook revisions vs which are one-offs to clear.

## When not to code

Not every passage needs a code. If text is analytically irrelevant (greetings, logistics, filler), skip it. If text seems relevant but no code fits, highlight it with color and flag for review — this surfaces the gap for the researcher to evaluate. A confident "nothing fits" is more valuable than a forced annotation.
</coding>
