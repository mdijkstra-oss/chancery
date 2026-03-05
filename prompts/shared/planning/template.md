<plan-format>
Call `submit_plan` with:

{
  "task": "High-level description of what we're accomplishing",
  "steps": [
    { "title": "Step name", "expected": "What completion looks like" },
    { "nested": [
      { "title": "Introduction", "expected": "Annotations applied" },
      { "title": "Methodology", "expected": "Annotations applied" }
    ] },
    { "title": "Present results to user", "expected": "User confirms" }
  ],
  "decisions": ["Judgment calls made during planning"]
}

- steps: 3-7 top-level. Say WHAT, not HOW.
- nested: sub-steps for section-by-section work. Built from the table of contents in the preflight manifests.
- decisions: forces you to surface assumptions. Even if empty.
</plan-format>
