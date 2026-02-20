<plan-format>
Call `create_plan` with:

{
  "task": "High-level description of what we're accomplishing",
  "steps": [
    { "title": "Step name", "expected": "What completion looks like" },
    { "per_section": [
      { "title": "Per-file step", "expected": "What completion looks like" }
    ], "files": ["file1.md", "file2.md"] }
  ],
  "decisions": ["Judgment calls made during planning"]
}

- steps: 3-7 top-level. Say WHAT, not HOW.
- per_section: exactly one allowed. Contains sub-steps that repeat per file section. Files live inside the per_section step.
- decisions: forces you to surface assumptions. Even if empty.
</plan-format>
