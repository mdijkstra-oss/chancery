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
- nested: sub-steps for section-by-section work. Built from the table of contents in the preflight manifests — not every section may be relevant (earlier work, out of scope). Filter to what the task needs.
- checkpoint: flag on a work step meaning "after doing this, check in with the user." Not a separate step — the work step itself pauses for feedback.
- decisions: forces you to surface assumptions. Even if empty.
</plan-format>
