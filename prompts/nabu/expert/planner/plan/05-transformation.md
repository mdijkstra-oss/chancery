# Plan: Content Transformation

## When

The intent involves:
- Restructuring, reformatting, or merging documents
- Summarizing content into a new document
- Converting between formats or structures
- Extracting and reorganizing content
- "Merge these files into one"
- "Restructure this document"
- "Summarize all interviews into a report"

## Plan Structure — New File

For transforms that rewrite majority of content, write to a new file:

```
create_plan:
  task: "Restructure [file] into [new format]"
  files: ["source.md"]
  steps:
    - title: "Create output file"
      expected: "New file 'source (new).md' created with title and structure"
    - per_section:
        - title: "Transform section content"
          expected: "Section content rewritten and appended to new file"
    - title: "Report to user"
      expected: "User informed: new version at 'source (new).md', original unchanged"
```

## Plan Structure — Inline Patches

For transforms that change minority of sections:

```
create_plan:
  task: "Update [specific aspects] of [file]"
  files: ["document.md"]
  steps:
    - per_section:
        - title: "Apply targeted changes"
          expected: "Section updated with [specific change]"
    - title: "Confirm changes"
      expected: "Summary of what changed and where"
```

## Plan Structure — Multi-File Merge

```
create_plan:
  task: "Merge [files] into single document"
  files: ["file-1.md", "file-2.md", "file-3.md"]
  steps:
    - title: "Create merged document"
      expected: "New file 'merged.md' created with combined structure"
    - per_section:
        - title: "Process and integrate section content"
          expected: "Section content merged into new document, deduplication applied"
    - title: "Report to user"
      expected: "User informed of merged result, originals unchanged"
```

## With Expert Analysis

When transformation requires interpretation (not just mechanical restructuring), add an expert:

```
create_plan:
  task: "Summarize healthcare discussions from interviews"
  files: ["interview-1.md", "interview-2.md"]
  ask_expert:
    expert: "analyst"
    using: "echo 'Identify and summarize healthcare-related discussions. Note key arguments, positions, and tensions.'"
  steps:
    - title: "Create summary document"
      expected: "New file 'healthcare-summary.md' created"
    - per_section:
        - title: "Extract healthcare content from analysis"
          expected: "Relevant findings written to summary document"
    - title: "Compile and structure summary"
      expected: "Summary organized by theme, presented to user"
```

## Key Points

- New file convention: `filename (new).md` — original stays untouched
- The user decides what to keep after reviewing
- For mechanical transforms (format conversion, concatenation), no expert needed
- For semantic transforms (summarize, merge with deduplication), consider adding an analyst expert
- Each section should be fully processed (including writes) before moving to the next
