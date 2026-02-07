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

### 1. Identify Files, Codebook, and Existing Annotations

Determine:
- Which files to code (from conversation context or ask user)
- Which codebook(s) to use (partial read to confirm structure—the expert will receive the full codebook)
- Whether the files already have annotations:

```
blocks json-attributes {files} | jq '.[] | .annotations | length'
```

If files already have annotations, ask the user what to do:
- "These files already have some codings. Should I continue from where the coding left off, or review everything fresh?"

Pass their answer as `instructions` to the expert. Common patterns:
- **Continue from uncoded**: "This content has existing annotations. Skip the entire coded region — do not add codes between or around existing annotations. Start coding only from the first passage after the last annotation onward."
- **Review all**: "Review all passages including already-coded ones. Use mark_for_deletion if existing annotations are incorrect."
- **Specific focus**: "Only code for [specific codes]. Skip passages already coded with those codes."

**Why "skip the region" not "skip individual passages":** Qualitative coding can always find something more between two existing codes — a nuance, an overlap, a sub-theme. If the expert is told to "skip coded passages" it will wedge new annotations between existing ones endlessly. "Skip the coded region" means: treat everything up to and including the last annotation as done. Only code fresh content after that point.

### 2. Create Plan with Expert

```
create_plan:
  task: "Apply codebook to interview transcripts"
  files: ["interview-1.md", "interview-2.md"]
  ask_expert:
    expert: "qualitative-researcher"
    task: "apply-codebook"
    using: "cat Codebook.md"
    instructions: "{based on user's answer about existing annotations}"
  steps:
    - per_section:
        - title: "Read applied annotations and surface to user"
          expected: "Read pending annotations from file. Combined with expert summary, relay findings to user in plain language."
        - title: "Review with user"
          expected: "User confirms annotations or requests changes."
        - title: "Apply user adjustments"
          expected: "Requested changes applied. Ambiguities recorded with resolved_locally. No changes if user confirmed."
    - title: "Revise codebook from resolutions"
      expected: "Expert updates codebook definitions based on resolved ambiguities. Changes applied with pending status."
```

### 3. Per Section (Surface and Review)

Each section is analyzed by the expert, who applies annotations directly to the document with `pending` status.

**The expert already applied all annotations.** Do NOT call `patch_json_block` to add, re-apply, or duplicate what the expert did. The annotations are already in the file.

**This is a conversation, not silent processing.**

**a) Read the applied annotations** — The file is your primary source. Query the pending annotations to see exactly what the expert wrote — passages, codes, confidence, ambiguity notes. This is cheaper than having the expert describe its work twice.

```
blocks json-attributes {current_file} | jq '.[] | .annotations | map(select(.status == "pending"))'
```

The expert's `orchestrator_summary` supplements this with context the file doesn't capture: patterns, gaps, edge cases, passages that didn't fit any code. Use both together.

**b) Surface to user** — Build the user-facing message from what you read in the file, enriched by the expert's observations. Present in plain non-technical language. No JSON, IDs, blocks, patches, or implementation details — the user is not a computer person. Use the actual passage text and code names (not IDs) from what you read.
- "I found 3 passages that match codes: [X] for '...passage...', [Y] for '...passage...'"
- "One is ambiguous: '...passage...' could be [A] or [B] — what do you think?"
- "I'd suggest removing a previous coding on '...passage...' because [reason]. Agree?"
- If nothing matched: "Nothing in this section fits the codebook."

**c) Wait for user** — They may:
- Confirm: "looks good" → complete the step, move to next section
- Disagree: "remove that one" → remove the pending annotation
- Clarify ambiguity: "use [A]" → update the pending annotation
- Ask questions

**d) Adjust ONLY if user requests** — Use `patch_json_block` to modify or remove pending annotations ONLY when the user explicitly asks for a change. No changes without user direction.

For resolved ambiguities, include `resolved_locally`:
```json
{
  "text": "...",
  "code": "code_xyz",
  "reason": "...",
  "resolved_locally": "User clarified: interpret 'testing policy' broadly"
}
```

**e) Complete step** — Move to next section

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

**Files already have annotations**
→ Check annotation count before planning. Ask user: skip coded passages, review everything, or focus on specific codes. Pass decision as `instructions` to expert.

**User wants exploratory coding (no predefined codes)**
→ Different workflow: surface themes, propose codes iteratively. Build codebook as you go.
</coding-documents>
