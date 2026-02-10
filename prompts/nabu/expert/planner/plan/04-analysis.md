# Plan: Framework Analysis

## When

The intent involves:
- Reviewing content against criteria, rubrics, or standards
- Evaluating arguments, compliance, or quality
- Applying a framework systematically across documents
- "Review chapters against style guide"
- "Check if this complies with..."
- "Evaluate proposals against criteria"

## Plan Structure

```
create_plan:
  task: "Review [files] against [framework]"
  files: ["chapter-1.md", "chapter-2.md"]
  ask_expert:
    expert: "analyst"
    using: "cat framework.md"
    instructions: "{focus areas or constraints}"
  steps:
    - per_section:
        - title: "Surface analysis findings to user"
          expected: "Issues, strengths, and observations discussed with user"
    - title: "Compile findings summary"
      expected: "Overview of patterns across all sections — key issues, common themes, recommendations"
```

## Variations

**With user involvement per section** (user wants to discuss each finding):
```
  steps:
    - per_section:
        - title: "Surface analysis to user"
          expected: "Expert findings presented in plain language"
        - title: "Discuss with user"
          expected: "User confirms, clarifies, or redirects focus"
```

**With action after analysis** (user wants changes applied):
```
  steps:
    - per_section:
        - title: "Surface analysis to user"
          expected: "Issues identified and discussed"
        - title: "Apply agreed changes"
          expected: "Patches applied to address confirmed issues"
    - title: "Summary of changes"
      expected: "Overview of all modifications made"
```

## Key Points

- The analyst expert is freeform — it advises, the orchestrator acts
- Surface findings in plain language, not raw expert output
- For compliance tasks, the framework document is the `using` source
- When multiple files share one framework, set `files` and `using` once — each section gets analyzed against the same framework
