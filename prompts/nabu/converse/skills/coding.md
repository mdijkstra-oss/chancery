# Coding Documents

<coding-documents>
Apply qualitative codes from a codebook to documents.

## Trigger

User asks to:
- "Code this file/document"
- "Apply codebook to..."
- "Annotate with codes"
- "Do qualitative coding on..."

## Prerequisites

- Codebook exists in workspace (file tagged `codebook` or named as such)
- Documents to code are identified

## Workflow

This is the typical flow—adapt based on user's specific needs or instructions.

### 1. Orient

Before planning, understand what you're working with:
- Locate the codebook
- Identify target documents (ask if ambiguous)
- Analyze codebook to understand the codes available

### 2. Plan

Create a plan with:
- `files`: the documents to code (not the codebook—you've already read it)
- `per_section` steps in this order:

```
create_plan:
  task: "Apply codebook to interview transcripts"
  files: ["interview-1.md", "interview-2.md"]
  steps:
    - per_section:
        - title: "Analyze which codes apply"
          expected: "List of codes and passages, ambiguities noted"
        - title: "Resolve ambiguities with user"
          expected: "User clarification received, codebook updated if needed"
        - title: "Apply annotations"
          expected: "Annotations added for this section"
    - title: "Summary"
      expected: "Counts per code, patterns observed"
```

**Order matters:** Understand before you apply. If there's ambiguity, ask the user before annotating. Don't apply with uncertainty—get clarity first, update codebook if needed, then apply.

If a section has no ambiguity, the "Resolve" step is quick (just note "no ambiguities"). If it does, you ask and wait for user response before continuing to "Apply."

### 3. Per Section

For each section, follow this order:

**a) Analyze** — Identify passages where codes apply. Keep notes brief—code name + short marker.

Good: "Appeal to expertise: RIVM/OMT mentions"
Bad: "multiple explicit expert-warrant statements (RIVM/OMT/WHO) → Appeal to expertise (callout_m66odckb)"

Never include code IDs in analysis or user-facing output. Reference codes by title only.

**b) Resolve ambiguities** — If anything is unclear:
- Ask the user: "This passage could be X or Y—which fits better?"
- Wait for response before continuing
- Update codebook if the answer reveals a gap or unclear definition
- Then proceed to Apply

If no ambiguities, note "No ambiguities" and continue.

**c) Apply** — Now that you understand, annotate using `upsert_annotations`:

```
upsert_annotations:
  path: "document.md"
  annotations:
    - text: "the exact passage from document"
      code: "callout_abc123"  # internal ID, never shown to user
      reason: "Why this code applies"
```

Set `code` OR `color`, never both. Use `code` when applying codebook codes. Use `color` only for ad-hoc highlights without a codebook reference.

Do NOT use `apply_local_patch` for annotations—it will fail. Use `remove_annotations` to delete.

**d) Note gaps** — If content suggests a missing code:
- Flag for user: "I'm seeing patterns about X that don't fit existing codes—want to add one?"

### 4. After All Sections

- Summarize what was coded: counts per code, any patterns
- List any unresolved questions
- Offer to update codebook if gaps were noted

## Edge Cases

**No codebook exists**
→ Ask: "I don't see a codebook. Want me to help create one, or do you have codes in mind?"

**Codebook exists but is empty/minimal**
→ Note: "The codebook does not seem fleshed out. Want to proceed, or develop it further first?"

**User wants exploratory coding (no predefined codes)**
→ Different workflow: read sections, surface themes, propose codes iteratively. Build codebook as you go rather than applying existing codes.

**Very large document**
→ The `per_section` approach handles this automatically. Don't try to read the whole thing at once.

**User disagrees with a coding decision**
→ Update the annotation. If it reveals a codebook gap, offer to clarify the code definition.

## Anti-patterns

- **Applying before understanding** — Don't annotate if you're uncertain. Ask first, then apply.
- **Reading all sections, then coding** — Process each section fully before moving on
- **Guessing on ambiguous cases** — Ask the user
- **Skipping the codebook read** — You need to know the codes before you can apply them
- **Coding without reasons** — Every annotation needs a `reason`
- **Exposing IDs** — Never show callout_xxx or code_xxx to users
</coding-documents>
