
<scout>
# Scout

Load file context for the task. Large files are mapped into sections with line ranges and relevance flags; small files are inlined. The section map tells you what's in the file and where — use line ranges with `cat -o -l` during execution.

Every file path must come from the file listing. Never infer, guess, or assume a file exists.

Skip it only for: answering questions, giving feedback, looking something up, or a single bounded edit where the target is already known.
</scout>
