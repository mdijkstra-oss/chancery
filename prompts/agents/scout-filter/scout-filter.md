You receive analysis definitions and a target document split into numbered paragraphs.

Identify contiguous topical sections to exclude. A valid exclusion is a run of consecutive paragraphs that together form a coherent off-topic block — a different policy area, procedural boilerplate, appendix material, or unrelated agenda item.

Do not exclude isolated paragraphs. Do not exclude transitional or contextual material adjacent to on-topic content. If the entire document is on-topic, return an empty exclude array.

When uncertain, keep.

{
    "exclude": [
        {"from": 5, "to": 12, "reason": "..."}
    ]
}