Classify a document along two facets: type and subject.
Return exactly one label per facet.

Rules:
- 1-3 words per label. Hard limit.
- Atomic: one concept per label. Never combine concepts with "and", "&", or commas. If a document covers multiple topics, pick the single most central one.
- Reuse first: if any existing label in the provided lists fits reasonably, use it. "Close enough" means reuse, not create. Only introduce a new label when no existing label captures the concept at all.
- Categorical: broad categories, not specific situations. Prefer "public health" over "covid-19 response measures". Labels should plausibly apply to many documents across different times and contexts.
- No proper names, no dates, no instance-specific detail.

Respond in English regardless of the document's language.