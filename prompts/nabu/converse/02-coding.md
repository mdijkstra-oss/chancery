# Coding Documents

<coding-documents>
Apply qualitative codes from a codebook to documents.

## Trigger

User asks to:
- "Code this file/document"
- "Apply codebook to..."
- "Annotate with codes"
- "Do qualitative coding on..."

## Key Principle

**Coding adds annotations to the original file.** Do NOT create a separate "coded" file.

## Before Delegating

Gather what you need:

1. **Files** — Which files to code? From context, or ask.
2. **Codebook** — Confirm it exists (tagged `codebook` or named as such).
3. **Existing annotations** — Check: `blocks json-attributes {files} | jq '.[] | .annotations | length'`
   - If files already have codings, ask: "Should I continue from where it left off, or review everything fresh?"
4. **Context** – Any other information you must gather before delegating.

5. Then delegate:

```
delegate_plan:
  intent: "Apply codebook to {files}"
  outcome: "All files coded, user reviewed each section"
  context: "ls *.md && blocks json-attributes *.md | jq '.[] | .tags'"
  involvement: "Review and confirm annotations per section"
  constraints: "{annotation handling, focus areas}"
```

## During Execution

The expert codes each section — annotations arrive already applied with `pending` status.

**Surface what happened.** Read the pending annotations, tell the user in plain language what was found. No JSON, no IDs, no technical details.

**Handle feedback:**
- Small tweaks ("remove that one", "change the code to X") → patch directly
- Bigger requests ("I think these codes should also apply here", "re-analyze with focus on Y") → call `ask_expert` for that section
- Ambiguities resolved by the user → record in `resolved_locally` on the annotation

**Don't re-apply.** The expert already wrote the annotations. Don't translate the summary into duplicate patches.

## Edge Cases

**No codebook exists** → "I don't see a codebook. Want me to help create one?"

**Nothing matched** → Valid. "No codes apply to this section."

**User disagrees** → Follow their judgment. MUST Note in `resolved_locally` for relevant annotations.

</coding-documents>
