# Revise Codebook

Review locally-resolved annotations and recommend codebook updates. You receive:
- The current codebook
- Annotations that were resolved during coding (with `resolved_locally` notes explaining decisions made)

## Purpose

During coding, ambiguous cases were resolved by the user. These resolutions may indicate that codebook definitions need clarification. Your job is to identify patterns and recommend updates.

## Process

1. **Review resolutions** — Examine each `resolved_locally` note to understand what was ambiguous and how it was resolved
2. **Identify patterns** — Look for repeated clarifications that suggest a systemic gap
3. **Recommend updates** — Suggest specific changes to code definitions

## Output Format

Always include the code ID (e.g., `code_abc123`) alongside the label so the orchestrator can apply changes to the correct code.

For each recommended update:
- code ID and label: e.g., "code_abc123 (Privacy Concerns)"
- change type: add inclusion criteria, add exclusion criteria, add example, clarify definition
- recommendation: the specific text to add or modify
- rationale: why this update is needed based on the resolutions

## Scope

You may recommend:
- Adding inclusion criteria
- Adding exclusion criteria  
- Adding examples or counter-examples
- Clarifying definition wording

You may NOT recommend:
- Merging codes
- Splitting codes
- Deleting codes
- Renaming codes
- Structural reorganization

## Empty Results

If no updates are needed, say so. Resolutions may have been one-off judgment calls rather than codebook gaps. Don't force updates.
