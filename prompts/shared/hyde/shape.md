The "hydes" array contains exactly 10 hypothetical passages:

3 paragraph passages, one for each angle:
1. direct — explicit language that names or addresses the query topic
2. hedged — surrounding language: conditions, qualifications, background, causes
3. consequence — implications, responses, follow-up, impact

Each paragraph passage:
- Is 4-5 sentences, written in the requested language
- Reads like an excerpt from a real document, not a description or analysis
- Reproduces the voice of the source corpus — does NOT answer the query or address a reader
- Preserves the query's specificity (broad queries produce broad passages; specific queries produce specific passages)

Paragraph object: {"type": "direct" | "hedged" | "consequence", "text": "<passage>"}

6 signal passages that capture specific formulations, phrases, or sentence patterns a reader would encounter in matching text. These are not summaries or condensed paragraphs — they are the kind of individual sentences that would trigger recognition of this pattern in context.

Each signal passage:
- Is 1-2 sentences max, written in the requested language
- Reproduces a characteristic formulation, not a description of one
- Each signal must use different vocabulary and phrasing from the others

Signal object: {"type": "signal", "text": "<short passage>"}

1 keywords passage: a space-separated list of 10–15 surface tokens that would literally appear in matching text. This feeds a lexical (bag-of-words) retriever, not the embedding model.

The keywords passage:
- Lists domain jargon, technical terms, named entities, and inflected forms a reader would encounter in matching text
- Includes multi-word phrases as-is (kept intact; the tokenizer splits them)
- Matches the vocabulary register described in the group description and is written in the requested language
- Omits generic connectives and stopwords ("the", "and", "however", language-equivalent)
- Uses surface forms (no stemming, no lemmatization) — include both singular and plural / inflected variants when the corpus would

Keywords object: {"type": "keywords", "text": "term1 term2 term3 ..."}