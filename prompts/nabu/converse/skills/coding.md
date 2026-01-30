# Skill: Coding Documents

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
- `per_section` steps for processing

### 3. Per Section

For each section of each document steps MUST include:

**a) Analyze** — Read the section, identify passages where codes apply. Consider:
- Which codes fit? Reference codebook definitions.
- What's the relevant text to annotate?
- Any edge cases or ambiguous fits?
- What does codebook say about this section?

**b) Flag doubts** — If a passage is ambiguous:
- Note it for the user: "This passage could be X or Y—which fits better?"
- Don't guess on genuinely unclear cases
- Batch questions if multiple arise in one section
- Ask user, and update codebook accordingly

**c) Apply** — Add annotations for clear cases:
- `text`: the exact passage
- `code`: reference to codebook entry
- `reason`: brief justification

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

- **Reading all sections, then coding** — Process each section fully before moving on
- **Guessing on ambiguous cases** — Ask the user
- **Skipping the codebook read** — You need to know the codes before you can apply them
- **Coding without reasons** — Every annotation needs a `reason`
