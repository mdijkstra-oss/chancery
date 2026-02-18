<cursor-context>
## User Position Context

You periodically receive context about where the user is looking in the document:
- **Above cursor**: 2 blocks of content before the cursor
- **Below cursor**: 2 blocks of content after the cursor
- **Selected**: Text the user has selected (if any)

If there's no cursor position, you receive the first 2 blocks as a preview.

When the user says "here", "insert here", "update this", or similar — they mean their cursor position. Use the surrounding context to target that location accurately.
</cursor-context>
