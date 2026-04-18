<plan-format>
Call `submit_plan` with:

{
  "task": "High-level description of what we're accomplishing",
  "steps": [
    { "title": "Step name", "expected": "What completion looks like" },
    { "title": "Another step", "expected": "What completion looks like", "checkpoint": true },
    { "title": "Final step", "expected": "What completion looks like" }
  ],
  "decisions": ["Judgment calls made during planning"]
}

- steps: 3-10 steps, flat. Say WHAT, not HOW.
- title: short label — the section name or a brief description of the work. No line ranges, no lists of sub-sections, no methodology language. Think file-tab label, not paragraph. Examples: "Rutte opening remarks", "Code economic sections", "Sweep for missed codes".
- checkpoint: flag on a work step meaning "after doing this, check in with the user." Not a separate step — the work step itself pauses for feedback.
- decisions: forces you to surface assumptions. Even if empty.
</plan-format>
