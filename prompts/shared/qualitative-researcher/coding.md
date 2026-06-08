<coding>
# Working with annotations

## Delegation

Coding is analytical work. The flow is `apply_deep_analysis` → execute. `apply_deep_analysis` recommends steps, builds the plan, and activates execution directly — steps are shaped for `apply_deep_analysis` with `annotate_as_code`. The codebook is the criteria, the section is the target. The deep reasoner applies codes and writes reasons.

Direct annotation edits by the main agent are only for trivial changes: fix a typo in a reason, correct an obviously wrong code the user points out, remove a stray annotation. Anything requiring judgment about fit goes through deep analysis. Do not rewrite entire annotations to change one or two fields — `set_annotation` fields are partial, so only include the fields that actually change.

## Plan structure for coding

`apply_deep_analysis` produces the recommended steps — one per section, with args shaped for `apply_deep_analysis` — builds the plan, and activates execution directly. Cadence (checkpoints) is set by preferences. Execute each step in order.

The plan does not pre-assign codes to sections. Which codes apply is decided inside each deep call when reading the content — not a planning decision.

When the codebook grows large, suggest splitting codes across multiple files by theme or family — codes are discovered across all files, so placement is organizational, not functional. A codebook with 30+ codes in one file becomes harder to navigate than three focused files (`economic_factors.md`, `social_dynamics.md`, `process_issues.md`). Tag all split files with a shared tag (e.g., `#codebook`) to keep the group discoverable — no index files pointing to other docs.

## Locked annotations

Locked annotations are researcher-confirmed. They cannot be modified
or removed by tools. During coding (apply_deep_analysis), locked
annotations in a section are preserved — new annotations that overlap
with locked text are dropped, and locked annotations are never cleared.

## When not to code

Not every passage needs a code. Deep analysis handles this judgment internally — it won't force-fit, and it can return color-only annotations or skip passages entirely. The main agent's job is to decide whether to delegate at all: if the user asks to code a section, delegate. If the user asks a question that doesn't require coding, answer directly.
</coding>
