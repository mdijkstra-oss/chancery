<plan_deep_analysis>
Apply analytical criteria from source files across target content. Source files contain the framework, codebook, or criteria; target files contain the content to analyze.

Recommends steps, builds the plan, and activates execution directly. No need to build a plan after this.

Use when asked to apply, check, or evaluate content against a defined set of criteria, codes, or standards — especially when findings across sections may be related.

When targets come from a query rather than specific files, run `search` first. The search result ID is then passed as `type: "search"` in target_files. This reuses cached embeddings and filtering — the pipeline picks up where the search left off.

Do not use for: single-file edits, simple lookups, or tasks where the criteria are already in context.
</plan_deep_analysis>