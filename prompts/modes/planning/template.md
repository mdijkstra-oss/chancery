<plan-format>
Call `submit_plan` with:

{
  "task": "High-level description of what we're accomplishing",
  "steps": [
    { "title": "Step name", "expected": "What completion looks like" },
    { "nested": [
      { "title": "Introduction", "expected": "Annotations applied" },
      { "title": "Methodology", "expected": "Annotations applied", "checkpoint": true }
    ] },
    { "title": "Final step", "expected": "What completion looks like" }
  ],
  "decisions": ["Judgment calls made during planning"]
}

- steps: 3-7 top-level. Say WHAT, not HOW.
- title: short label — the section name or a brief description of the work. No line ranges, no lists of sub-sections, no methodology language. Think file-tab label, not paragraph. Examples: "Rutte opening remarks", "Code economic sections", "Sweep for missed codes".
- nested: section-by-section work. One nested group per section from the scout map. Inside each group, substeps break the work into focused passes (e.g. one substep per code family). Use section labels and line ranges to identify work units. Do not put unrelated sections into the same nested group. Do not use a nested group with only one substep — that is just a regular step.
- checkpoint: flag on a work step meaning "after doing this, check in with the user." Not a separate step — the work step itself pauses for feedback.
- decisions: forces you to surface assumptions. Even if empty.
</plan-format>
