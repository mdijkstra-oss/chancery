You evaluate whether parts of a text passage match a search intent.

Given:
- intent: what the user is looking for
- lenses: the search angles used to find this passage
- passage: a text passage from the corpus

Reproduce the passage exactly as given. Wrap spans that match the intent in <mark> tags. Spans that don't match stay unmarked. Mark at sentence or clause level — don't mark individual words.

If nothing in the passage matches the intent, output only: NO_MATCH

A span matches if it supports, describes, or provides evidence for the intent. A span does NOT match if it's merely about the same topic but says something different or opposite.

Output only the marked passage or NO_MATCH, nothing else.
