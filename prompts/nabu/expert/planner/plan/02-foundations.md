# Plan Foundations

## What Is a Step?

A step is one unit of work that delivers a tangible outcome. The test: can you summarize what this step *produced* in one sentence?

- "Identify matching codes for this section" → one step (produces: list of proposed codes)
- "Surface findings to user" → one step (produces: user's confirmation or corrections)
- "Apply confirmed annotations" → one step (produces: patched document)

These are three steps, not one. Each has a distinct deliverable that the next step depends on.

A step MAY involve multiple tool calls when they serve the same deliverable — five patches that apply confirmed annotations is still one step. But "analyze content, discuss with user, and apply changes" bundles three deliverables and MUST be three steps.

**The anti-pattern:** "Let me analyze the codes, confer with the user, and apply annotations" — this is a paragraph pretending to be a step. If it contains "and" connecting actions with different outputs, it's multiple steps.

## The `create_plan` Schema

```
create_plan:
  task: string          # High-level description
  files: [string]       # Files relevant to this task (required for per_section)
  ask_expert:           # Auto-analyze each section with an expert
    expert: string      # Specialist type
    task: string        # Specific task (optional)
    using: string       # Shell command for framework/context
    instructions: string # Extra guidance (optional)
  steps:
    - title: string     # Brief title
      expected: string  # What this step should produce
    - per_section:      # Steps repeated for each file section
        - title: string
          expected: string
```

## Working With File Content

When a plan involves processing the content of files (analysis, coding, transformation):

1. **Determine the files first** — orient if you don't know which files are relevant
2. **Pass `files` to `create_plan`** — even for a single file. Without `files`, sections won't be delivered.
3. **Use `per_section`** for steps that process each section of the files
4. **Do NOT include "read file" steps** — the system provides content automatically
5. **Use `ask_expert` for interpretation** — when content needs domain expertise, add `ask_expert` to the plan. Experts see each section; the orchestrator uses their interpretation for the full picture.

### Per-Section Processing

Large files lose information in the context window middle. Sectioned processing keeps each chunk in focus. Content is split on markdown block boundaries.

When `ask_expert` is set, each section arrives pre-analyzed with an `<analysis>` block from the expert. The orchestrator surfaces this to the user.

### Inline Patches vs. New File

Small, targeted edits — the orchestrator patches the file inline:
- Add a paragraph, insert a table, replace a word
- Append content, update a single section
- **Annotations/coding** — always patch the original file's annotation block

Large transforms that rewrite majority of content — new file:
- Restructuring, reformatting, converting, merging
- Any plan where most sections produce content writes

Convention for new files: `filename (new).md`. Original stays untouched.

**TLDR**: Minority of sections change → patch original. Majority change → new file. For annotations, always patch original.

When designing a plan that uses a new file, include a first step to create it.

### When NOT to Use Files

- Simple metadata-only operations (tags, attributes)
- File structure tasks (create, delete, rename)
- Tasks where you need to find which files are relevant (orient first, then re-plan with files)
