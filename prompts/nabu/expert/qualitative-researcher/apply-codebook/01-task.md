# Apply Codebook

Apply the codebook to the content. You receive:
- The codebook (codes with definitions, criteria, examples)
- Content to analyze
- Existing annotations on this content (if any)

## Process

For each passage that matches a code:

1. **Identify** — Find text that fits a code's definition
2. **Match** — Select the most appropriate code using the definition, inclusion/exclusion criteria, and examples
3. **Justify** — Explain why this code applies
4. **Confidence** — If uncertain, state your confidence and the source of ambiguity

For existing annotations:
- Review whether they still fit given the current codebook
- Flag for deletion if the code no longer applies or the match is incorrect

## Output Format

Always include the code ID (e.g., `code_abc123`) alongside the human-readable label so the orchestrator can apply changes precisely.

For new annotations:
- text: exact passage from content
- code ID and label: e.g., "code_abc123 (Privacy Concerns)"
- reason: why this code applies
- confidence: high/medium/low
- ambiguity: explain if confidence < high

For deletions:
- annotation ID: the existing annotation to remove
- reason: why it should be removed

Notes: patterns observed, potential codebook gaps

## Empty Results

It is valid to find no matches. If nothing in the content fits the codebook, say so clearly. Don't force matches.

## Guidelines

- Quote the exact text being coded
- One annotation per distinct passage
- A passage may have multiple codes if genuinely applicable
- Prefer precision over coverage—skip marginal matches
- Flag passages that seem important but don't fit any code
