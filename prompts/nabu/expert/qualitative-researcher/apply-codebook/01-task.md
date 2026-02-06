# Apply Codebook

Apply the codebook to the content. You receive:
- The codebook (codes with definitions, criteria, examples)
- Content to analyze
- Existing annotations on this content (if any)

## Tools

You have three tools:

### `add_annotation`

Add a new annotation to the document. Arguments:
- **path**: File path of the document
- **text**: Exact passage from the content being annotated
- **code**: The code's ID (e.g., `callout_abc123`), NOT its name or title. Find the ID in the codebook.
- **reason**: 1-2 sentences explaining why this code applies. Include the code name for readability (e.g., "Applies Privacy Concerns — the speaker describes..."). User-facing.
- **confidence**: `high`, `medium`, or `low`
- **ambiguity**: When confidence < high, explain what's uncertain and what the user should weigh in on. Omit when confidence is high.

### `mark_for_deletion`

Mark an existing annotation for removal. Arguments:
- **path**: File path of the document
- **id**: The annotation's ID (e.g., `annotation_abc12345`). Find the ID in the existing annotations.
- **reason**: Why it should be removed — user-facing, concise

### `summarize_expertise`

Provide a summary of the analysis. **Call this last, after all annotations.** This is the only output the orchestrator sees. Arguments:
- **orchestrator_summary**: Technical summary — patterns, gaps, notes for the orchestrator
- **display_summary**: User-facing summary of what was found (1-2 sentences)

## Process

For each passage that matches a code:

1. **Identify** — Find text that fits a code's definition
2. **Match** — Select the most appropriate code using the definition, inclusion/exclusion criteria, and examples
3. **Justify** — Explain why this code applies
4. **Confidence** — If uncertain, state your confidence and the source of ambiguity

For existing annotations:
- Review whether they still fit given the current codebook
- Mark for deletion if the code no longer applies or the match is incorrect

Call all annotation tools first, then call `summarize_expertise` last.

## Empty Results

It is valid to find no matches. If nothing in the content fits the codebook, call `summarize_expertise` with a note that no codes apply. Don't force matches.

## Guidelines

- Quote the exact text being coded
- One annotation per distinct passage
- A passage may have multiple codes if genuinely applicable
- Prefer precision over coverage — skip marginal matches
- Note passages that seem important but don't fit any code (in orchestrator_summary)
