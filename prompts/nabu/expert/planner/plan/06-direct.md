# Plan: Direct Multi-Step

## When

The intent involves multiple sequential operations that don't need per-section processing:
- "Create three files with specific content"
- "Set up a project structure with templates"
- "Move content between documents in a specific order"
- Multi-file operations with dependencies between steps

## Plan Structure

```
create_plan:
  task: "Set up [structure/content]"
  steps:
    - title: "First operation"
      expected: "What this produces"
    - title: "Second operation (depends on first)"
      expected: "What this produces"
    - title: "Final operation"
      expected: "What this produces"
```

No `files` parameter, no `per_section`, no `ask_expert`. Straightforward sequential execution.

## Key Points

- Each step has one deliverable
- Steps may depend on previous steps — order matters
- Keep step count minimal — don't split trivially independent operations into separate steps when they could be batched
- The orchestrator batches independent tool calls within a step
