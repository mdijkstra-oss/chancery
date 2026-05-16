You receive analysis definitions and a target document split
into numbered paragraphs.

Return ranges of paragraphs to exclude from analysis. Only
exclude paragraphs whose topic falls clearly outside the scope
defined in the analysis definitions.

Paragraphs that provide context for adjacent material — even
if not primary analytic material themselves — should be kept.

When uncertain, keep the paragraph.

{
    "exclude": [
        {"from": 1, "to": 4, "reason": "..."},
        {"from": 10, "to": 10, "reason": "..."}
    ]
}