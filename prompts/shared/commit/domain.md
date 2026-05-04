<commit-domain>
# Qualitative research workspace

The workspace stores qualitative research documents as markdown files. Each file is a research artifact — an interview transcript, a codebook, a set of field notes, an analytical memo, preferences, or a settings file.

## Document structure

Files are markdown with embedded JSON blocks fenced as `json-callout`, `json-annotations`, or `json-settings`. These blocks carry structured data inline with the researcher's prose.

- **Callouts** (`json-callout`): highlighted segments of text with an ID, color, and optional annotation references
- **Annotations** (`json-annotations`): analytical notes attached to callouts — coded passages with reason and code reference
- **Settings** (`json-settings`): project configuration — tags, codebook definitions, analytical parameters

## ID conventions

Every callout, annotation, and code has an ID matching the pattern `[a-z]+-[0-9][a-z0-9]{7}` (e.g. `callout-1bc12345`, `code-9af34567`, `annotation-2de89012`). Annotations reference callouts by ID. Codes reference codebook entries by ID. These cross-file references create a web of dependencies.

## What researchers do

Researchers code interviews (attach analytical codes to text passages), restructure documents, draft memos, revise codebooks, annotate field notes, and adjust settings. Each of these is a distinct research action. A single editing session might involve several unrelated actions — coding one interview, then restructuring another document.

## Commit message language

Describe the research action, not the file operation. "Coded resilience in interview_sarah" — not "updated interview_sarah.md". Use researcher vocabulary: coded, annotated, restructured, drafted, revised, merged codes, split code, added tag, reorganized.
</commit-domain>
