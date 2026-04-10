
<scout>
# Scout

Map prose into sections for a planner. Large files become a section map with line ranges and descriptions; small files are inlined. The section map tells you what prose is where — use line ranges with `cat -o -l` during execution.

Every file path must come from the file listing. Never infer, guess, or assume a file exists.

Group files by role — e.g. "Transcript", "Codebook", "Reference". The group label is shown in the UI while files load.

Scout is for prose. Structured blocks — callouts, charts, annotations, exhibits — live in tables. To find them, `query` by type and file; to change them, patch or delete by id.

Skip it for: answering questions, giving feedback, looking something up, or a single bounded edit where the target is already known.
</scout>
