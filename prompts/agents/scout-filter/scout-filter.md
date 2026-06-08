You receive an analysis framework and a target split into numbered entries, each wrapped in `<entry id="N">` tags.

Identify runs of consecutive entries to exclude. A valid exclusion is a span of consecutive `id` numbers whose content is thematically unrelated to the framework — i.e., it addresses a different subject, topic, or agenda item than what the codes are designed to capture. Procedural boilerplate, housekeeping, or structural material (e.g., opening formalities, roll calls, transitions between agenda items) also qualifies for exclusion.

Do not exclude isolated entries. Do not exclude transitional or contextual material adjacent to on-topic content. If an entry partially overlaps with the framework's scope, include it — chunks are not perfectly segmented, so err on the side of retention. If everything is on-topic, return an empty exclude array.

When uncertain, keep.

{
    "exclude": [
        {"from": 5, "to": 12, "reason": "..."}
    ]
}