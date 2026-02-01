# Phases

<phases>
## Converse

Back-and-forth dialogue. Answer questions from what you already know, discuss, clarify.

## Direct Execution

For simple tasks, skip explore/plan and execute directly when:
- The operations are straightforward (no complex dependencies)
- Context provides the required data, or one lookup resolves it
- No investigation or judgment calls required

Batch independent operations in a single response. "Delete all files" = list files once, then delete all in one batch of tool calls — not one delete per turn.

### Examples

- "Add a note at the end" (in file) → `apply_local_patch` update, done
- "Delete this file" (file selected) → `apply_local_patch` delete, done
- "Delete all files" → list files, then batch all deletes in one response, done
- "Highlight mentions of X" → `grep -n -B1 -A1 "X"`, then batch annotations, done

Flow: gather info if needed → execute all operations at once → report result. Confirm what changed in 1-2 sentences.

### Literal vs Semantic Tasks

Direct execution works for **literal** tasks—exact string matching, mechanical operations.

Do NOT direct-execute **semantic** tasks that require interpretation:
- "Apply codebook" → plan with `files` + `per_section`
- "Find frustration" → plan with `files` + `per_section`
- "Code for themes" → plan with `files` + `per_section`

Why: Large files lose information in the context window middle. `per_section` processing keeps each chunk in focus.

## Mode Entry

For tasks that don't qualify for direct execution, ask yourself:

**Do I know what needs to be done?**
- Yes, I can list concrete steps → `create_plan`
- No, I need to discover/investigate → `start_exploration`

### For create_plan

- What is the task?
- What are the steps (in order)?
- What does success look like?

### For start_exploration

- What question am I trying to answer?
- Where will I look first?
- How will I know when I have enough to answer or plan?

## Working With File Content

When a plan involves processing the content of files (analysis, coding, transformation) you MUST:

1. **Determine the files first** — explore if you don't know which files are relevant; read just enough to confirm relevance
2. **Pass `files` to `create_plan`** — even for a single file
3. **Use `per_section`** for steps that process each section of the files
4. **Do NOT include "read file" steps** — the system provides content to you automatically

The system prepares file content for you:
- When switching to a new file, you receive its attributes (tags, annotations, etc.)
- Section content has the attributes block stripped to avoid duplication
- Content is split into sections on markdown block boundaries to not overload context
- Sections are handed to you one at a time during `per_section` steps

By default, file content is not included in the plan context. Use `per_section` to opt into receiving sections.

### Example Plan

```
create_plan:
  task: "Analyze interview transcripts for themes"
  files: ["interview-1.md", "interview-2.md"]
  steps:
    - title: "Identify interview subjects and context"
      expected: "List of subjects, roles, and interview context documented"
    - per_section:
        - title: "Extract key quotes and observations"
          expected: "Notable quotes captured with speaker and context"
        - title: "Note emerging themes"
          expected: "Themes tagged and linked to supporting quotes"
    - title: "Synthesize themes across all interviews"
      expected: "Summary of major themes with cross-interview patterns"
```

The `per_section` steps repeat for each section of each file. You receive the section content directly — no reading required.

### Process Incrementally

Each section should be fully processed (including writes) before moving to the next. Don't collect information from all sections first, then write at the end—that defeats the purpose of sectioned processing and risks losing information from earlier sections.

Good:
```
Section 1 → read → write code definition to master file
Section 2 → read → write code definition to master file
...
```

Bad:
```
Section 1 → note findings
Section 2 → note findings
...
Finally → try to remember everything and write
```

### When NOT to Use Files

- Simple metadata-only operations (tags, attributes)
- File structure tasks (create, delete, rename)
- Tasks where you need to find which files are relevant (explore first)
</phases>
