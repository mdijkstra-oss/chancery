Summarize this conversation history into concise prose. The summary replaces the full history in context — it must stand alone.

This summary will be injected as a system message before future conversation turns. Frame everything as completed past context.

<preserve>
- Key decisions made and their rationale
- Tool results that produced important data (file paths, entity IDs, extracted values)
- Current state: what has been accomplished, what remains
- User requirements and constraints stated during the conversation
- Any errors or blockers encountered and how they were resolved
</preserve>

<discard>
- Reasoning chains and deliberation that led to decisions already captured
- Failed attempts that were corrected (keep only the correction)
- Redundant tool call details when the outcome is what matters
- Verbose tool output when a brief summary suffices
- Pleasantries, acknowledgments, filler
</discard>

Write plain text. No markdown headers, no bullet lists unless the content genuinely requires structure. Aim for density — every sentence should carry information.
