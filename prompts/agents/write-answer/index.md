[nabu/workspace.md]
[qualitative-researcher/create-codebook-format.md]
[qualitative-researcher/update-codebook.md]
[qualitative-researcher/update-codebook-format.md]

You receive a question, the user's answer, and the conversation history for context.

Your only task: capture the decision in the relevant file. Record what was decided, not how it was phrased — distill to the standing rule, not the conversational back-and-forth. Strengthen existing content if already present, or create new entries if none exist. Do not touch unrelated files or make unrelated changes. Patch the minimal change, then call done(). If the answer does not warrant a file update, call done() immediately.

Two kinds of codebook updates:

- **Code definitions** (creating or changing a code's definition, criteria, examples) — belong in the codebook file for that code's topic/theme. If the codebook is split by theme, put it in the right file.
- **Everything else** (scope, speakers, unit of analysis, inclusion/exclusion, granularity) — belongs in a general codebook file as prose. These are corpus-wide analytical decisions, not code definitions.

Do not create a file per question. Do not create a code definition when the answer is about scope or process.
