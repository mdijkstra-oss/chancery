# Writing prompts for modern LLMs

## Three layers, zero overlap

| Layer | Teaches | Where in code |
|-------|---------|---------------|
| Tool schema | Format, arg names, types | `nabu-theatron/app/lib/agent/executors/*.ts` — zod schemas with `.describe()` |
| Tool prompt | Tool-specific patterns | `hermes-logos/prompts/tools/**/*.md` — gated by `requires` frontmatter |
| Domain prompt | Domain intent and judgment | `hermes-logos/prompts/nabu/**/*.md` — layered by folder hierarchy |

Each layer trusts the one below. Never re-teach what a lower layer already covers.

## Rules

**Tool schema teaches format.** The model knows how to call a tool from its JSON schema. Do not restate arg names, types, or call structure in prose. Do not wrap domain examples in full tool call JSON — show the value shape only.

**Domain prompts describe what and when, not how.** "Add an annotation when the passage expresses frustration" — good. Wrapping that in `{ path, language, operations: [{ op, path, value }] }` — bad. The model fixates on reproducing your format instead of reasoning about the domain.

**Redundancy is worse than omission.** When the same information appears in multiple layers, modern models waste reasoning tokens reconciling the copies. If they conflict even slightly, performance degrades sharply. Say it once, in the right layer.

**Front-load rules, minimize noise.** Lead with key constraints. Cut surrounding prose. Reasoning models (o3, o4-mini, Claude with extended thinking) think internally — verbose context and chain-of-thought examples actively hurt them. Measured: 6% accuracy improvement from concise front-loaded descriptions vs buried-in-prose.

**Few-shot examples show domain judgment, not format.** Modern models understand tool calling from schema alone. Examples still help — but only when they show *what to decide*, not *how to format the call*. Show the value shape (`{ text, reason, code, ambiguity }`) and the reasoning behind confidence levels, not the tool wrapper around it.

**Over-prompting causes overtriggering.** Claude 4.x tools that undertriggered on older models now trigger correctly. Aggressive prompting ("always use this tool when...") causes the model to use it when it shouldn't. Replace blanket defaults with targeted instructions.

**Named frameworks activate training knowledge.** Instead of explaining a methodology in prose, reference it by name — the model brings the full depth from training. "As per constant comparative method" compresses a textbook into three words. But don't stack conflicting frameworks; name only what anchors a specific behavior the system needs.

**Principles let the model reason; literal rules make it pattern-match.** Write principles for general guidance — the model applies judgment to novel situations. Reserve literal if/then rules for known, repeated failure modes where the model consistently gets it wrong. An if/then tree for every scenario is brittle: there will always be a case you didn't enumerate. A principle handles edge cases through understanding.

**Tool results steer more than prompts.** A tool that returns current state — progress, next item, what changed — keeps the model oriented without runtime reminders. Invest in informative tool results before adding nudges. Anthropic found more value optimizing tools than optimizing prompts. JetBrains found that injecting LLM summaries mid-loop made agents run 15% longer — the summaries smoothed over signals that the agent should stop.

**Side-effects belong at exit, not mid-loop.** When a task produces a durable side-effect (updating memory, logging observations), capture it as data on the resolve call and dispatch a focused agent — don't nudge the main agent to do it mid-work. The pattern: add an optional field to resolve, check it in the delegation layer, fire-and-forget a single-purpose `chat=false` endpoint. Fits when the update is self-contained (read state, compare, write) and doesn't need user interaction.

**Contradictions are expensive.** GPT-5 and Claude 4.x both perform measurably worse with contradictory instructions across prompt layers. One clear rule beats two rules that mostly agree.

## What goes where

**In the tool schema (function definition):**
- Parameter names, types, required/optional
- One-line description of what the tool does

**In the tool prompt (e.g. json-block.md, patching.md):**
- Patterns specific to the tool: selector syntax, batching, `"""` content
- When to use this tool vs another tool
- Validation behavior, error recovery

**In the domain prompt (e.g. 02-coding.md):**
- Domain concepts: what an annotation is, what codes mean
- Decision criteria: when confidence is medium vs high, when to merge
- Value shapes: the data structure the domain works with (without the tool call wrapper)
- Workflow: sequences of domain actions (resolve ambiguity → record feedback → optionally merge)
- Examples that show judgment calls, not format

## Tool errors are prompts

The LLM reads tool errors. An error message is a prompt that teaches recovery.

**Standard errors pass through unchanged.** LLMs understand Unix stderr from training. `grep: No such file or directory` is already clear. Don't wrap or transform known error formats.

**Custom errors state: what failed, why, what to do.** "Numeric index `/annotations/0` rejected. Target array items by selector: `/annotations[id=annotation_xyz]`" — the model can fix this in one retry. "Invalid path" — the model guesses.

**Show available values on constraint violations.** "Unknown code `code_xyz`. Available: `code_a1b2`, `code_c3d4`, `code_e5f6`" beats "Unknown code." The model picks from the list instead of hallucinating an ID.

**Partial success separates succeeded from failed.** When 3 of 5 operations succeed, report which ones failed. The model shouldn't retry what already worked.

**Keep errors concise.** No stack traces, no debug info, no internal function names. Every token is attention cost.

**Don't pre-prompt for errors that aren't happening.** If the model handles tool choice and recovery well on its own, don't add error guidance to prompts. Only add targeted corrections when you see a recurring failure pattern.

## Anti-patterns

- Full tool call JSON in domain prompts (model fixates on format)
- Restating tool arg types in prose ("path is a string containing...")
- Identical guidance in tool prompt and domain prompt
- Chain-of-thought examples for reasoning models
- "Always use X" blanket rules (causes overtriggering)
- Many examples showing the same thing (diminishing returns, increases fixation)
- Format examples where domain examples belong
- Blanket reminders every N turns (model fixates on the reminder instead of the work)

## Codebase architecture

### Layer 1: Tool schemas (zod)

Location: `nabu-theatron/app/lib/agent/executors/*.ts`

Each tool is a zod schema with `.describe()` on fields. This generates the JSON schema the model sees as the tool definition. The model learns format, arg names, and types from this alone.

Example: `patch.ts` defines `apply_local_patch`, `json-patch.ts` defines `patch_json_block`.

To change what a tool accepts or how args are described: edit the zod schema. The `.describe()` strings are the tool's documentation to the model.

### Layer 2: Tool prompts

Location: `hermes-logos/prompts/tools/**/*.md`

Gated by frontmatter — a tool prompt is only included when the agent has the matching tool:

```yaml
---
requires:
  - apply_local_patch
---
```

OR semantics: if any required tool is available, the prompt is included.

These teach tool-specific patterns the schema can't express: selector syntax, `"""` fenced content, batching, fuzzy matching, when to use `patch_json_block` vs `apply_local_patch`.

### Layer 3: Domain prompts

Location: `hermes-logos/prompts/nabu/**/*.md`

Assembled by walking the folder hierarchy. For `nabu/expert/qualitative-researcher`:

```
nabu/01-identity.md           ← base identity (all agents)
nabu/02-discipline.md         ← base discipline (all agents)
nabu/03-cursor.md             ← base cursor (all agents)
nabu/expert/01-identity.md    ← expert identity (all experts)
nabu/expert/qualitative-researcher/01-identity.md  ← specialist identity
nabu/expert/qualitative-researcher/02-coding.md    ← specialist domain
```

Deeper folders inherit all ancestor prompts. Numbered prefixes control ordering within each folder.

These teach only domain intent: what an annotation is, when to code, confidence thresholds, merge workflow. No tool call format.

### Extra prompts

Location: `hermes-logos/prompts/extra/{plan,exec,merge}/*.md`

Appended when the agent is in plan/exec/merge mode. Same rules — domain intent only.

### Config cascade

Each folder can have a `config.json` controlling model, reasoning effort, etc. Walks up the hierarchy — deepest config wins.

```
nabu/config.json               → reasoning_effort: "medium"
nabu/expert/config.json         → reasoning_effort: "high"
```

### Assembly flow

1. Frontend agent definition (`nabu-theatron/.../agents.ts`) declares path + tools
2. Request hits backend: `POST /expert/qualitative-researcher?chat=true`
3. `ComposePrompt` walks ancestors, loads base → chat → tool (filtered by requires) → extra
4. Concatenated with `\n\n`, sent as system message

### Where to edit

| Want to change | Edit |
|----------------|------|
| Tool arg names/types/descriptions | Zod schema in `nabu-theatron/.../executors/*.ts` |
| Tool-specific patterns (selectors, batching) | `hermes-logos/prompts/tools/**/*.md` |
| Domain intent (what to annotate, when to merge) | `hermes-logos/prompts/nabu/**/*.md` |
| Which tools an agent has | Agent definition in `nabu-theatron/.../agents.ts` |
| Model/reasoning settings | `config.json` in the appropriate folder level |
| Plan/exec/merge behavior | `hermes-logos/prompts/extra/{plan,exec,merge}/*.md` |
