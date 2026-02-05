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
        - title: "Review analysis and apply codes"
          expected: "Codes discussed with user, then applied to original file"
    - title: "Revise codebook from resolutions"
      expected: "Codebook updated based on resolved ambiguities"
```

### 3. Per Section (Interactive)

Each section arrives with an `<analysis>` block from the expert containing:
- Proposed annotations (text, code, reason, confidence, ambiguity)
- Suggested deletions of existing annotations (id, reason)
- Notes on patterns or gaps

**This is a conversation, not silent processing.**

**a) Surface to user** — Tell them what the analysis found:
- "This section has 3 potential codes: [X] for '...passage...', [Y] for '...passage...'"
- "One is ambiguous: the expert flagged '...passage...' as possibly [A] or [B] because [reason]. What do you think?"
- "The expert suggests removing an existing annotation because [reason]. Agree?"
- If nothing matched: "No codes apply to this section based on the current codebook."

**b) Wait for user** — Let them respond. They may:
- Confirm: "looks good, apply it"
- Clarify ambiguity: "use [A], it's about policy not economy"
- Disagree: "skip that one" or "actually that's [C]"
- Ask questions

**c) Apply to original file** — Use `apply_local_patch` to add/update/remove annotations in the document's annotation block. NOT a new file.

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

The expert returns recommended updates. Review and apply to codebook via `apply_local_patch`.

Skip if no resolutions occurred.

### 5. Summary

- Report codes applied: counts per code, patterns observed
- Report codebook changes made (if any)

## Anti-patterns

- **Creating a separate coded file** — Annotations go on the original. Always.
- **Silent processing** — Each section needs user interaction before applying.
- **Applying despite ambiguity** — If confidence is low, ask first.
- **Batch-applying at the end** — Apply each section after user confirms.
- **Ignoring the analysis** — The expert did the work. Surface it, discuss it.
- **Exposing IDs** — Never show internal IDs to users.

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
