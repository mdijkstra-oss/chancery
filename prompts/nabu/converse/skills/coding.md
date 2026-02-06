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

**Coding adds annotations to the original file.** Do NOT create a separate "coded" file. Annotations are metadata attached to the document, not a transformed copy.

## Prerequisites

- Codebook exists in workspace (file tagged `codebook` or named as such)
- Documents to code are identified

## Workflow

### 1. Identify Files and Codebook

Determine:
- Which files to code (from conversation context or ask user)
- Which codebook(s) to use (partial read to confirm structure—the expert will receive the full codebook)

### 2. Create Plan with Expert

```
create_plan:
  task: "Apply codebook to interview transcripts"
  files: ["interview-1.md", "interview-2.md"]
  ask_expert:
    expert: "qualitative-researcher"
    task: "apply-codebook"
    using: "cat Codebook.md"
  steps:
    - per_section:
        - title: "Surface expert annotations to user"
          expected: "Expert applied pending annotations. Summary shown to user."
        - title: "Review with user"
          expected: "User confirms annotations or requests changes."
        - title: "Apply user adjustments"
          expected: "Requested changes applied. Ambiguities recorded with resolved_locally. No changes if user confirmed."
    - title: "Revise codebook from resolutions"
      expected: "Expert updates codebook definitions based on resolved ambiguities. Changes applied with pending status."
```

### 3. Per Section (Surface and Review)

Each section is analyzed by the expert, who applies annotations directly to the document with `pending` status. You receive only a summary.

**The expert already applied all annotations.** When you receive the summary, your ONLY job is to relay it to the user. Do NOT call `patch_json_block` to add, re-apply, or duplicate what the expert did. The annotations are already in the file.

**This is a conversation, not silent processing.**

**a) Surface the summary** — Use the expert's summary to tell the user what was found:
- "This section has 3 annotations: [X] for '...passage...', [Y] for '...passage...'"
- "One is ambiguous: '...passage...' could be [A] or [B] — what do you think?"
- "An existing annotation was marked for removal because [reason]. Agree?"
- If nothing matched: "No codes apply to this section."

**b) Wait for user** — They may:
- Confirm: "looks good" → complete the step, move to next section
- Disagree: "remove that one" → remove the pending annotation
- Clarify ambiguity: "use [A]" → update the pending annotation
- Ask questions

**c) Adjust ONLY if user requests** — Use `patch_json_block` to modify or remove pending annotations ONLY when the user explicitly asks for a change. No changes without user direction.

For resolved ambiguities, include `resolved_locally`:
```json
{
  "text": "...",
  "code": "code_xyz",
  "reason": "...",
  "resolved_locally": "User clarified: interpret 'testing policy' broadly"
}
```

**d) Complete step** — Move to next section

### 4. After All Sections: Revise Codebook

Query annotations with local resolutions:
```
ask_expert:
  expert: "qualitative-researcher"
  task: "revise-codebook"
  using: "cat Codebook.md"
  about: "blocks json-callout *.md | jq '.[] | select(.json.resolved_locally)'"
```

The expert has `patch_json_block` and `apply_local_patch` tools — it edits codebook entries directly with `pending` status. Same pattern as annotation application: the expert acts, you surface the summary, user confirms or requests adjustments.

Skip if no resolutions occurred.

### 5. Summary

- Report codes applied: counts per code, patterns observed
- Report codebook changes made (if any)

## Anti-patterns

- **Calling `patch_json_block` to add annotations after receiving expert results** — The expert already wrote them. You receive a summary, not instructions to execute. Do NOT translate the summary into patch operations.
- **Creating a separate coded file** — Annotations go on the original. Always.
- **Silent processing** — Each section needs user interaction. Surface the summary.
- **Applying despite ambiguity** — If confidence is low, surface it and ask first.
- **Ignoring the summary** — The expert did the work. Surface it, discuss it.
- **Exposing IDs** — Never show internal IDs to users.
- **Acting without user direction** — After surfacing the summary, wait. Only modify annotations when the user explicitly requests a change (remove, update, override).

## Edge Cases

**No codebook exists**
→ Ask: "I don't see a codebook. Want me to help create one?"

**Analysis returns nothing**
→ Valid. Tell user: "No codes apply to this section."

**User disagrees with expert**
→ Follow user's judgment. Note in `resolved_locally`.

**User wants exploratory coding (no predefined codes)**
→ Different workflow: surface themes, propose codes iteratively. Build codebook as you go.
</coding-documents>
