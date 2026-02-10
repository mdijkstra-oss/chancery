# Plan: Qualitative Coding

## When

The intent involves:
- "Code this file/document"
- "Apply codebook to..."
- "Annotate with codes"
- "Do qualitative coding on..."

## Prerequisites

Before planning, verify via shell:
- Codebook exists in workspace (file tagged `codebook` or named as such)
- Documents to code are identified
- Check for existing annotations:

```
blocks json-attributes {files} | jq '.[] | .annotations | length'
```

If files already have annotations, the orchestrator should have included guidance in the constraints about how to handle them (continue from uncoded, review all, or specific focus). Pass this as `instructions` to the expert.

## Plan Structure

```
create_plan:
  task: "Apply codebook to [files]"
  files: ["interview-1.md", "interview-2.md"]
  ask_expert:
    expert: "qualitative-researcher"
    task: "apply-codebook"
    using: "cat Codebook.md"
    instructions: "{annotation handling from constraints}"
  steps:
    - per_section:
        - title: "Read applied annotations and surface to user"
          expected: "Read pending annotations from file. Combined with expert summary, relay findings to user in plain language."
        - title: "Review with user"
          expected: "User confirms annotations or requests changes."
        - title: "Apply user adjustments"
          expected: "Requested changes applied. Ambiguities recorded with resolved_locally. No changes if user confirmed."
    - title: "Revise codebook from resolutions"
      expected: "Expert updates codebook definitions based on resolved ambiguities. Changes applied with pending status."
```

## Key Points

- The expert applies annotations directly with `pending` status — the orchestrator reads and surfaces them, does not re-apply
- Each section needs user interaction (surface → review → adjust)
- The final step uses `qualitative-researcher/revise-codebook` to update codebook based on `resolved_locally` entries
- Skip the revise step if no resolutions occurred
